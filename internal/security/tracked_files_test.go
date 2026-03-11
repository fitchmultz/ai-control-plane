// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify tracked-file enumeration stays repository-relative and sorted.
//
// Responsibilities:
//   - Cover successful `git ls-files` enumeration.
//   - Verify path normalization across nested tracked files.
//
// Scope:
//   - Unit tests for tracked-file enumeration only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Tests create isolated temporary Git repositories.
//   - Enumeration uses the repository index rather than filesystem heuristics.
package security

import (
	"context"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestListTrackedFiles_ReturnsSortedRepoRelativePaths(t *testing.T) {
	repoRoot := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
		}
	}

	runGit("init")
	testutil.WriteRepoFile(t, repoRoot, "z-last.txt", "z\n")
	testutil.WriteRepoFile(t, repoRoot, "nested/alpha.txt", "a\n")
	testutil.WriteRepoFile(t, repoRoot, filepath.ToSlash("nested/beta.txt"), "b\n")
	testutil.WriteRepoFile(t, repoRoot, "deploy/incubating/terraform/README.md", "incubating\n")
	runGit("add", ".")

	paths, err := ListTrackedFiles(context.Background(), repoRoot)
	if err != nil {
		t.Fatalf("ListTrackedFiles returned error: %v", err)
	}

	want := []string{"nested/alpha.txt", "nested/beta.txt", "z-last.txt"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}
