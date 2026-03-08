// connector_mode_test.go - Mode and constructor coverage for the database connector.
//
// Purpose:
//   - Verify database mode selection and constructor guard rails.
//
// Responsibilities:
//   - Assert explicit external mode is honored.
//   - Assert ambiguous DATABASE_URL-only setups are rejected.
//   - Assert default embedded mode and admin-service mode checks stay stable.
//
// Scope:
//   - Constructor and mode-selection behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests do not require a live PostgreSQL runtime.
package db

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func TestNewConnectorExternalMode(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "external")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")

	connector := NewConnector("")
	defer connector.Close()

	if !connector.IsExternal() {
		t.Fatal("expected external mode")
	}
	if connector.IsEmbedded() {
		t.Fatal("did not expect embedded mode")
	}
	if connector.ConfigError() != nil {
		t.Fatalf("unexpected config error: %v", connector.ConfigError())
	}
}

func TestNewConnectorFlagsAmbiguousDatabaseURL(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")

	connector := NewConnector("")
	defer connector.Close()

	if connector.ConfigError() == nil {
		t.Fatal("expected ambiguous database configuration error")
	}
}

func TestDefaultEmbeddedModeWithoutOverrides(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "")
	t.Setenv("DATABASE_URL", "")

	connector := NewConnector("")
	defer connector.Close()

	if !connector.IsEmbedded() {
		t.Fatal("expected embedded mode by default")
	}
}

func TestNewAdminServiceRejectsExternalMode(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "external")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")

	connector := NewConnector("")
	defer connector.Close()

	if _, err := NewAdminService(connector); err == nil {
		t.Fatal("expected admin service construction to reject external mode")
	}
}

func TestNewAdminServiceRejectsNilConnector(t *testing.T) {
	if _, err := NewAdminService(nil); err == nil {
		t.Fatal("expected nil connector to be rejected")
	}
}

func TestAdminServiceBackupAndRestoreSurfaceContainerLookupErrors(t *testing.T) {
	service, err := NewAdminService(&Connector{
		settings: config.DatabaseSettings{Mode: config.DatabaseModeEmbedded},
	})
	if err != nil {
		t.Fatalf("NewAdminService() error = %v", err)
	}

	if _, err := service.Backup(context.Background()); err == nil {
		t.Fatal("expected backup to fail without compose-backed container")
	}
	if err := service.Restore(context.Background(), strings.NewReader("select 1;")); err == nil {
		t.Fatal("expected restore to fail without compose-backed container")
	}
}
