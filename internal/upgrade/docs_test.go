// docs_test.go - Documentation contract tests for the upgrade framework.
//
// Purpose:
//   - Keep the published upgrade docs aligned with the typed framework rules.
//
// Responsibilities:
//   - Ensure the compatibility matrix documents the framework guardrails.
//   - Ensure any future explicit catalog edge is listed in the matrix.
//
// Scope:
//   - Upgrade documentation contract tests only.
//
// Usage:
//   - Run via `go test ./internal/upgrade`.
//
// Invariants/Assumptions:
//   - Public upgrade docs must stay truthful when no in-place edges exist.
package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityMatrixDocumentsFrameworkGuardrails(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "deployment", "UPGRADE_COMPATIBILITY_MATRIX.md"))
	if err != nil {
		t.Fatalf("read compatibility matrix: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"pre-framework releases",
		"No in-place edges are shipped in `0.1.0`.",
		"Only when an explicit edge exists in the typed upgrade catalog",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("compatibility matrix missing %q", want)
		}
	}
}

func TestCompatibilityMatrixMentionsEveryCatalogEdge(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "deployment", "UPGRADE_COMPATIBILITY_MATRIX.md"))
	if err != nil {
		t.Fatalf("read compatibility matrix: %v", err)
	}
	content := string(data)

	for _, edge := range DefaultCatalog().Edges {
		row := fmt.Sprintf("| %s | %s |", edge.From, edge.To)
		if !strings.Contains(content, row) {
			t.Fatalf("compatibility matrix missing row for %s -> %s", edge.From, edge.To)
		}
	}
}
