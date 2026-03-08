// client_test.go - Tests for the typed database connector and service split.
//
// Purpose:
//   - Verify the database connector resolves mode/configuration consistently.
//
// Responsibilities:
//   - Assert explicit ACP_DATABASE_MODE values drive connector mode.
//   - Assert ambiguous DATABASE_URL-only configurations are surfaced.
//   - Assert admin service construction rejects unsupported external mode.
//
// Scope:
//   - Configuration and constructor behavior only.
//
// Usage:
//   - Used through its package exports and test entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Tests do not require a live PostgreSQL runtime.
package db

import (
	"os"
	"testing"
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
	_ = os.Unsetenv("ACP_DATABASE_MODE")
	_ = os.Unsetenv("DATABASE_URL")

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
