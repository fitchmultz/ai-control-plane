// client_test.go - Tests for the typed database client.
//
// Purpose:
//
//	Verify the shared database service resolves mode and configuration
//	consistently for embedded and external runtime paths.
//
// Responsibilities:
//   - Assert mode detection honors explicit ACP_DATABASE_MODE values.
//   - Assert ambiguous DATABASE_URL-only configurations are surfaced.
//
// Non-scope:
//   - Does not require a live PostgreSQL instance.
//
// Invariants/Assumptions:
//   - Tests validate constructor/runtime configuration behavior only.
//
// Scope:
//   - Configuration and constructor behavior only.
//
// Usage:
//   - Used through its package exports and test entrypoints as applicable.
package db

import (
	"os"
	"testing"
)

func TestDetectDatabaseMode(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "embedded")
	if got := detectDatabaseMode(); got != "embedded" {
		t.Fatalf("expected embedded mode, got %q", got)
	}

	t.Setenv("ACP_DATABASE_MODE", "external")
	if got := detectDatabaseMode(); got != "external" {
		t.Fatalf("expected external mode, got %q", got)
	}
}

func TestNewClientExternalMode(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "external")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")

	client := NewClient("")
	defer client.Close()

	if !client.IsExternal() {
		t.Fatal("expected external mode")
	}
	if client.IsEmbedded() {
		t.Fatal("did not expect embedded mode")
	}
	if client.ConfigError() != nil {
		t.Fatalf("unexpected config error: %v", client.ConfigError())
	}
}

func TestNewClientFlagsAmbiguousDatabaseURL(t *testing.T) {
	t.Setenv("ACP_DATABASE_MODE", "")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")

	client := NewClient("")
	defer client.Close()

	if client.ConfigError() == nil {
		t.Fatal("expected ambiguous database configuration error")
	}
}

func TestDefaultEmbeddedModeWithoutOverrides(t *testing.T) {
	_ = os.Unsetenv("ACP_DATABASE_MODE")
	_ = os.Unsetenv("DATABASE_URL")

	client := NewClient("")
	defer client.Close()

	if !client.IsEmbedded() {
		t.Fatal("expected embedded mode by default")
	}
}
