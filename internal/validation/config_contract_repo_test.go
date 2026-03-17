// config_contract_repo_test.go - Repository config contract regression tests.
//
// Purpose:
//   - Verify the tracked repository configuration satisfies the machine-readable
//     config contract.
//
// Responsibilities:
//   - Run full config-contract validation against the live repository tree.
//
// Scope:
//   - Repository config-contract validation only.
//
// Usage:
//   - Run via `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - The repository root is discoverable from the current working tree.
package validation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryConfigContract(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	issues, err := ValidateConfigContract(repoRoot)
	if err != nil {
		t.Fatalf("ValidateConfigContract() error = %v", err)
	}
	if len(issues) > 0 {
		t.Fatalf("config contract issues:\n%s", strings.Join(issues, "\n"))
	}
}
