// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Verify repo-policy helpers and source-policy walkers directly.
//
// Responsibilities:
//   - Cover required header detection helpers.
//   - Verify direct env-access enforcement and allowlists.
//   - Lock down deterministic issue shaping for CI-facing policy checks.
//
// Scope:
//   - Unit tests for repository policy helpers only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use isolated temporary repository fixtures.
//   - Policy exclusions for tests and internal/config remain intentional.
package validation

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestHasRequiredHeader(t *testing.T) {
	t.Parallel()

	valid := []byte(`// sample.go - Demonstrate a valid header.
//
// Purpose:
//   - Verify header detection.
//
// Responsibilities:
//   - Exercise required fields.
//
// Scope:
//   - Unit tests only.
//
// Usage:
//   - Used by repo policy tests.
//
// Invariants/Assumptions:
//   - Header parsing stops before package declarations.
package sample
`)
	if !hasRequiredHeader(valid) {
		t.Fatal("expected valid header to pass")
	}

	invalid := []byte(`// missing sections
//
// Purpose:
//   - Incomplete.
package sample
`)
	if hasRequiredHeader(invalid) {
		t.Fatal("expected incomplete header to fail")
	}
}

func TestValidateGoHeaders_OnlyFlagsGoFilesMissingRequiredFields(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "internal/good.go", `// good.go - Valid header.
//
// Purpose:
//   - Demonstrate a valid repo policy header.
//
// Responsibilities:
//   - Keep header coverage deterministic.
//
// Scope:
//   - Unit tests only.
//
// Usage:
//   - Used by ValidateGoHeaders tests.
//
// Invariants/Assumptions:
//   - Equivalent inputs produce equivalent issues.
package internal
`)
	testutil.WriteRepoFile(t, repoRoot, "internal/bad.go", "package internal\n")
	testutil.WriteRepoFile(t, repoRoot, "README.md", "# ignored\n")

	issues, err := ValidateGoHeaders(repoRoot)
	if err != nil {
		t.Fatalf("ValidateGoHeaders returned error: %v", err)
	}
	want := []string{"internal/bad.go: missing required top-of-file purpose header fields"}
	if !reflect.DeepEqual(issues, want) {
		t.Fatalf("issues = %v, want %v", issues, want)
	}
}

func TestValidateDirectEnvAccess_RespectsAllowlistAndIgnoresTests(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "internal/app/main.go", `package app
import (
  "os"
  "github.com/mitchfultz/ai-control-plane/internal/envfile"
)
func getenv() string {
  _ = os.LookupEnv("A")
  _ = envfile.LookupFile("demo/.env", "B")
  return os.Getenv("C")
}
`)
	testutil.WriteRepoFile(t, repoRoot, "internal/config/allowed.go", `package config
import "os"
func getenv() string { return os.Getenv("ALLOWED") }
`)
	testutil.WriteRepoFile(t, repoRoot, "internal/app/main_test.go", `package app
import "os"
func getenv() string { return os.Getenv("IGNORED_IN_TESTS") }
`)
	testutil.WriteRepoFile(t, repoRoot, filepath.ToSlash("internal/validation/repo_policy.go"), `package validation
import "os"
func getenv() string { return os.Getenv("SELF_ALLOWED") }
`)

	issues, err := ValidateDirectEnvAccess(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDirectEnvAccess returned error: %v", err)
	}
	want := []string{
		`internal/app/main.go: direct config access "envfile.LookupFile(" is forbidden outside internal/config`,
		`internal/app/main.go: direct config access "os.Getenv(" is forbidden outside internal/config`,
		`internal/app/main.go: direct config access "os.LookupEnv(" is forbidden outside internal/config`,
	}
	if !reflect.DeepEqual(issues, want) {
		t.Fatalf("issues = %v, want %v", issues, want)
	}
}

func TestValidateDirectEnvAccess_SkipsVendorAndGitDirs(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "vendor/example/ignored.go", `package ignored
import "os"
func getenv() string { return os.Getenv("IGNORED") }
`)
	testutil.WriteRepoFile(t, repoRoot, ".git/hooks/ignored.go", `package ignored
import "os"
func getenv() string { return os.Getenv("IGNORED") }
`)

	issues, err := ValidateDirectEnvAccess(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDirectEnvAccess returned error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected ignored directories to be skipped, got %v", issues)
	}
}

func TestValidateGoHeaders_RequiresCommentBlockBeforePackage(t *testing.T) {
	t.Parallel()

	data := []byte(strings.TrimSpace(`
package sample

// Purpose:
//   - Too late.
`))
	if hasRequiredHeader(data) {
		t.Fatal("expected package-first file to fail header detection")
	}
}
