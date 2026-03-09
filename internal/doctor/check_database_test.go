// check_database_test.go - Focused coverage for database doctor checks.
//
// Purpose:
//   - Verify database doctor checks adapt runtime component state consistently.
//
// Responsibilities:
//   - Cover missing runtime, ambiguous-config severity, and healthy passthrough.
//
// Scope:
//   - Database diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic runtime component reports.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestDBConnectableCheckID(t *testing.T) {
	t.Parallel()
	if (dbConnectableCheck{}).ID() != "db_connectable" {
		t.Fatalf("expected ID db_connectable")
	}
}

func TestDBConnectableCheckRunMissingRuntime(t *testing.T) {
	t.Parallel()

	result := (dbConnectableCheck{}).Run(context.Background(), Options{})
	if result.Level != status.HealthLevelUnknown || result.Severity != SeverityRuntime {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDBConnectableCheckRunAmbiguousConfig(t *testing.T) {
	result := (dbConnectableCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("database", status.ComponentDetails{
			LookupError: status.LookupErrorDatabaseConfigAmbiguous,
		}, status.HealthLevelUnhealthy, "Database configuration is ambiguous"),
	})

	if result.Severity != SeverityPrereq {
		t.Fatalf("expected prereq severity, got %+v", result)
	}
}

func TestDBConnectableCheckRunHealthyPassthrough(t *testing.T) {
	result := (dbConnectableCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("database", status.ComponentDetails{}, status.HealthLevelHealthy, "Database is reachable"),
	})

	if result.Level != status.HealthLevelHealthy || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}
