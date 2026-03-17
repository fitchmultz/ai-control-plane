// env_test.go - Tests for explicit env-file migrations.
//
// Purpose:
//   - Verify typed env-file mutation behavior for upgrade workflows.
//
// Responsibilities:
//   - Cover rename, set, remove, and require mutations.
//   - Ensure migrated files are written deterministically.
//
// Scope:
//   - Migration env-file helper tests only.
//
// Usage:
//   - Run via `go test ./internal/migration`.
//
// Invariants/Assumptions:
//   - Tests operate on temporary private env files.
package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyEnvMutationRenameSetRemoveAndRequire(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(path, []byte("OLD_KEY=one\nKEEP_KEY=two\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	result, err := ApplyEnvMutation(path, EnvMutation{
		ID:      "rename-old-key",
		Summary: "Rename OLD_KEY and add NEW_REQUIRED",
		Rename:  map[string]string{"OLD_KEY": "NEW_KEY"},
		Set:     map[string]string{"NEW_REQUIRED": "present"},
		Remove:  []string{"KEEP_KEY"},
		Require: []string{"NEW_KEY", "NEW_REQUIRED"},
	}, true)
	if err != nil {
		t.Fatalf("ApplyEnvMutation() error = %v", err)
	}
	if !result.Changed {
		t.Fatal("expected mutation to report changes")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	content := string(data)
	for _, want := range []string{"NEW_KEY=one\n", "NEW_REQUIRED=present\n"} {
		if !strings.Contains(content, want) {
			t.Fatalf("migrated env missing %q\n%s", want, content)
		}
	}
	if strings.Contains(content, "OLD_KEY=") || strings.Contains(content, "KEEP_KEY=") {
		t.Fatalf("stale keys remained in migrated env\n%s", content)
	}
}
