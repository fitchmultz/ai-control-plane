// support_matrix_test.go - Coverage for tracked support-matrix helpers.
//
// Purpose:
//   - Verify support matrix loading and surface filtering behavior.
//
// Responsibilities:
//   - Cover not-found and parse failures for the tracked support matrix.
//   - Cover supported/incubating surface filtering.
//
// Scope:
//   - Support matrix loader and helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/catalog`.
//
// Invariants/Assumptions:
//   - Fixtures are temporary and deterministic.
package catalog

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadSupportMatrixErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	if _, err := LoadSupportMatrix(missing); err == nil {
		t.Fatalf("expected not found error")
	}

	badPath := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(badPath, []byte("surfaces: ["), 0o644); err != nil {
		t.Fatalf("write bad support matrix: %v", err)
	}
	if _, err := LoadSupportMatrix(badPath); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestSupportMatrixSurfaceFilters(t *testing.T) {
	matrix := SupportMatrix{
		Surfaces: []SupportSurface{
			{ID: "docker", Status: "supported"},
			{ID: "helm", Status: "incubating"},
			{ID: "terraform", Status: "incubating"},
		},
	}

	if got := len(matrix.SupportedSurfaces()); got != 1 {
		t.Fatalf("SupportedSurfaces() len = %d", got)
	}
	if got := len(matrix.IncubatingSurfaces()); got != 2 {
		t.Fatalf("IncubatingSurfaces() len = %d", got)
	}
}

func TestLoadSupportMatrixSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "support-matrix.yaml")
	content := `public_docs:
  - README.md
reference_docs:
  - docs/reference/acpctl.md
incubating_terms:
  - helm
surfaces:
  - id: docker
    label: Docker
    status: supported
    summary: Host-first runtime
    owner: platform
    validation:
      - make ci
  - id: helm
    label: Helm
    status: incubating
    summary: Incubating track
    owner: platform
    validation:
      - internal only
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write support matrix: %v", err)
	}

	matrix, err := LoadSupportMatrix(path)
	if err != nil {
		t.Fatalf("LoadSupportMatrix() error = %v", err)
	}
	if !slices.Equal(matrix.PublicDocs, []string{"README.md"}) {
		t.Fatalf("PublicDocs = %v", matrix.PublicDocs)
	}
	if !strings.Contains(matrix.IncubatingSurfaces()[0].Summary, "Incubating") {
		t.Fatalf("unexpected incubating surface summary: %+v", matrix.IncubatingSurfaces()[0])
	}
}
