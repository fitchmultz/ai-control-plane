// connector_query_test.go - Query and helper coverage for the database connector.
//
// Purpose:
//   - Verify connector query helpers and error shaping across external and embedded paths.
//
// Responsibilities:
//   - Exercise ping/scalar helpers with deterministic sqlmock responses.
//   - Cover embedded-mode failures when compose wiring is unavailable.
//   - Cover timeout and error-formatting helper behavior.
//
// Scope:
//   - Connector helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests remain deterministic and do not shell out to Docker.
package db

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

func TestConnectorEnsureSQLDBRequiresURL(t *testing.T) {
	connector := &Connector{
		settings: config.DatabaseSettings{Mode: config.DatabaseModeExternal},
	}

	if _, err := connector.ensureSQLDB(context.Background()); err == nil {
		t.Fatal("expected missing DATABASE_URL error")
	}
}

func TestConnectorPingReturnsConfigError(t *testing.T) {
	connector := &Connector{
		settings: config.DatabaseSettings{AmbiguousErr: errors.New("ambiguous")},
	}

	probe := connector.ping(context.Background())
	if probe.Healthy {
		t.Fatal("expected unhealthy probe")
	}
	if probe.Error != "ambiguous" {
		t.Fatalf("unexpected probe error %q", probe.Error)
	}
}

func TestConnectorPingExternalSuccess(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	mock.ExpectPing()

	probe := connector.ping(context.Background())
	if !probe.Healthy {
		t.Fatalf("expected healthy probe, got %+v", probe)
	}
}

func TestConnectorScalarHelpersUseExternalDB(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)

	expectExactQuery(mock, "SELECT version();", exactQueryRows("version").AddRow(" PostgreSQL 17 "))
	expectExactQuery(mock, "SELECT COUNT(*) FROM counts;", exactQueryRows("count").AddRow("7"))

	version, err := connector.scalarString(context.Background(), "SELECT version();")
	if err != nil {
		t.Fatalf("scalarString() error = %v", err)
	}
	if version != "PostgreSQL 17" {
		t.Fatalf("scalarString() = %q, want trimmed value", version)
	}

	count, err := connector.scalarInt(context.Background(), "SELECT COUNT(*) FROM counts;")
	if err != nil {
		t.Fatalf("scalarInt() error = %v", err)
	}
	if count != 7 {
		t.Fatalf("scalarInt() = %d, want 7", count)
	}
}

func TestConnectorScalarIntRejectsNonIntegerResults(t *testing.T) {
	connector, mock := newExternalSQLMockConnector(t)
	expectExactQuery(mock, "SELECT COUNT(*) FROM counts;", exactQueryRows("count").AddRow("abc"))

	if _, err := connector.scalarInt(context.Background(), "SELECT COUNT(*) FROM counts;"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestConnectorEmbeddedFailuresWithoutCompose(t *testing.T) {
	connector := &Connector{
		settings: config.DatabaseSettings{Mode: config.DatabaseModeEmbedded},
	}

	if _, err := connector.containerID(context.Background()); err == nil {
		t.Fatal("expected missing compose error")
	}
	if _, err := connector.scalarString(context.Background(), "SELECT 1;"); err == nil {
		t.Fatal("expected embedded scalarString to fail without compose")
	}
	probe := connector.ping(context.Background())
	if probe.Healthy {
		t.Fatalf("expected unhealthy embedded probe, got %+v", probe)
	}
}

func TestConnectorExecErrorPrefersStructuredContext(t *testing.T) {
	connector := &Connector{}

	if got := connector.execError("database ping", proc.Result{Stderr: " permission denied "}); got != "database ping: permission denied" {
		t.Fatalf("execError(stderr) = %q", got)
	}
	if got := connector.execError("database ping", proc.Result{Err: errors.New("boom")}); got != "database ping: boom" {
		t.Fatalf("execError(err) = %q", got)
	}
	if got := connector.execError("database ping", proc.Result{}); got != "database ping" {
		t.Fatalf("execError(default) = %q", got)
	}
}

func TestWithTimeoutContextHonorsExistingDeadline(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	child := withTimeoutContext(parent, 10*time.Second)
	parentDeadline, _ := parent.Deadline()
	childDeadline, _ := child.Deadline()
	if !parentDeadline.Equal(childDeadline) {
		t.Fatal("expected existing deadline to be preserved")
	}
}

func TestWithTimeoutContextAddsDeadlineWhenMissing(t *testing.T) {
	child := withTimeoutContext(context.Background(), time.Second)
	deadline, ok := child.Deadline()
	if !ok {
		t.Fatal("expected deadline to be added")
	}
	if time.Until(deadline) <= 0 {
		t.Fatal("expected future deadline")
	}
}

func TestDatabaseNameAndUserFallbacks(t *testing.T) {
	connector := &Connector{}
	if connector.databaseName() != defaultDBName {
		t.Fatalf("databaseName() = %q, want %q", connector.databaseName(), defaultDBName)
	}
	if connector.databaseUser() != defaultDBUser {
		t.Fatalf("databaseUser() = %q, want %q", connector.databaseUser(), defaultDBUser)
	}

	connector.settings.Name = " custom-db "
	connector.settings.User = " custom-user "
	if got := connector.databaseName(); !strings.Contains(got, "custom-db") {
		t.Fatalf("databaseName() = %q", got)
	}
	if got := connector.databaseUser(); !strings.Contains(got, "custom-user") {
		t.Fatalf("databaseUser() = %q", got)
	}
}
