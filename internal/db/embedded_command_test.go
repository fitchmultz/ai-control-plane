// embedded_command_test.go - Embedded command-surface coverage for database helpers.
//
// Purpose:
//   - Exercise embedded database subprocess paths without requiring Docker.
//
// Responsibilities:
//   - Stub the `docker` executable with deterministic script behavior.
//   - Cover embedded connector ping/query success through the canonical proc layer.
//   - Cover admin backup and restore success using the shared timeout helpers.
//
// Scope:
//   - Embedded-mode command execution behavior only.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests manipulate PATH in-process and restore it after completion.
//   - The fake docker script remains deterministic and local to the test tempdir.
package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func TestConnectorEmbeddedCommandHelpersSucceed(t *testing.T) {
	restorePath := installFakeDocker(t)
	defer restorePath()

	connector := &Connector{
		settings:  config.DatabaseSettings{Mode: config.DatabaseModeEmbedded},
		container: "postgres-test",
	}

	probe := connector.ping(context.Background())
	if !probe.Healthy {
		t.Fatalf("expected healthy embedded probe, got %+v", probe)
	}

	value, err := connector.scalarString(context.Background(), "SELECT 42;")
	if err != nil {
		t.Fatalf("scalarString() error = %v", err)
	}
	if value != "42" {
		t.Fatalf("scalarString() = %q, want trimmed embedded value", value)
	}
}

func TestAdminServiceBackupAndRestoreSucceedWithEmbeddedCommandSurface(t *testing.T) {
	restorePath := installFakeDocker(t)
	defer restorePath()

	service, err := NewAdminService(&Connector{
		settings:  config.DatabaseSettings{Mode: config.DatabaseModeEmbedded},
		container: "postgres-test",
	})
	if err != nil {
		t.Fatalf("NewAdminService() error = %v", err)
	}

	backup, err := service.Backup(context.Background())
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	if !strings.Contains(backup, "backup payload") {
		t.Fatalf("Backup() = %q, want fake dump payload", backup)
	}

	if err := service.Restore(context.Background(), strings.NewReader("select 1;")); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
}

func installFakeDocker(t *testing.T) func() {
	t.Helper()

	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "docker")
	script := `#!/bin/sh
set -eu
args=" $* "
case "$args" in
  *" pg_isready "*)
    exit 0
    ;;
  *" pg_dump "*)
    printf 'backup payload\n'
    ;;
  *" psql "*"-c "*)
    printf ' 42 \n'
    ;;
  *" psql "*)
    cat >/dev/null
    ;;
  *)
    printf 'unexpected docker args: %s\n' "$*" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", scriptPath, err)
	}

	originalPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("Setenv(PATH) error = %v", err)
	}

	return func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			t.Fatalf("restore PATH error = %v", err)
		}
	}
}
