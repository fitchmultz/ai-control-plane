// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Verify license policy validation and restricted-component scanning paths.
//
// Responsibilities:
//   - Cover malformed and incomplete policy documents.
//   - Verify path/content regex matching and exclusion handling.
//   - Lock down deterministic finding ordering for CI-facing output.
//
// Scope:
//   - Unit tests for license-boundary validation only.
//
// Usage:
//   - Run with `go test ./internal/security`.
//
// Invariants/Assumptions:
//   - Tests provision isolated repository fixtures on disk.
//   - Findings remain repository-relative and stably sorted.
package security

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestValidateLicensePolicy_MissingFileFails(t *testing.T) {
	repoRoot := t.TempDir()

	_, err := ValidateLicensePolicy(repoRoot)
	if err == nil {
		t.Fatal("expected missing license policy error")
	}
	if !strings.Contains(err.Error(), "THIRD_PARTY_LICENSE_MATRIX.json") {
		t.Fatalf("expected policy path in error, got %v", err)
	}
}

func TestValidateLicensePolicy_ReportsMissingRequiredFields(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "docs/policy/THIRD_PARTY_LICENSE_MATRIX.json", `{"schema_version":"1.0.0","policy_id":"","scan_scope":{"include":[]},"restricted_components":[]}`)

	findings, err := ValidateLicensePolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateLicensePolicy returned error: %v", err)
	}
	want := []string{"docs/policy/THIRD_PARTY_LICENSE_MATRIX.json: missing required policy fields"}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("findings = %v, want %v", findings, want)
	}
}

func TestValidateLicensePolicy_RejectsInvalidRegex(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "docs/policy/THIRD_PARTY_LICENSE_MATRIX.json", `{
  "schema_version": "1.0.0",
  "policy_id": "third-party-license-boundary",
  "scan_scope": {"include": ["docs/**"]},
  "restricted_components": [
    {"name": "restricted-docs", "match": {"content_regex": ["("]}}
  ]
}`)

	_, err := ValidateLicensePolicy(repoRoot)
	if err == nil {
		t.Fatal("expected invalid regex error")
	}
	if !strings.Contains(err.Error(), "missing closing") {
		t.Fatalf("expected regex compilation error, got %v", err)
	}
}

func TestValidateLicensePolicy_FindsRestrictedReferencesDeterministically(t *testing.T) {
	repoRoot := t.TempDir()
	testutil.WriteRepoFile(t, repoRoot, "docs/policy/THIRD_PARTY_LICENSE_MATRIX.json", `{
  "schema_version": "1.0.0",
  "policy_id": "third-party-license-boundary",
  "scan_scope": {
    "include": ["docs/**", "vendor/**"],
    "exclude": ["docs/excluded/**"]
  },
  "restricted_components": [
    {
      "name": "copyleft-binary",
      "match": {
        "path_regex": ["vendor/.+\\.jar$"],
        "content_regex": ["GPL-3\\.0", "LGPL-2\\.1"]
      }
    }
  ]
}`)
	testutil.WriteRepoFile(t, repoRoot, "docs/guide.md", "Allowed text only\n")
	testutil.WriteRepoFile(t, repoRoot, "docs/reference.md", "This package is licensed under GPL-3.0-only\n")
	testutil.WriteRepoFile(t, repoRoot, "docs/licenses/lgpl.txt", "LGPL-2.1 text\n")
	testutil.WriteRepoFile(t, repoRoot, "docs/excluded/ignore.md", "GPL-3.0-only\n")
	testutil.WriteRepoFile(t, repoRoot, "vendor/client.jar", "binary placeholder\n")
	testutil.WriteRepoFile(t, repoRoot, "vendor/library.txt", "LGPL-2.1 text\n")

	findings, err := ValidateLicensePolicy(repoRoot)
	if err != nil {
		t.Fatalf("ValidateLicensePolicy returned error: %v", err)
	}
	want := []string{"docs/licenses/lgpl.txt", "docs/reference.md"}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("findings = %v, want %v", findings, want)
	}
}

func TestFlattenLicenseRegexpsSkipsEmptyPatterns(t *testing.T) {
	components := []licenseRestrictedComponent{{
		Name: "copyleft",
		Match: licenseRestrictedMatch{
			PathRegex:    []string{"vendor/.+", ""},
			ContentRegex: []string{"GPL", " "},
		},
	}}

	pathPatterns := flattenLicensePathRegexps(components)
	if !reflect.DeepEqual(pathPatterns, []string{"vendor/.+", ""}) {
		t.Fatalf("path patterns = %v", pathPatterns)
	}
	contentPatterns := flattenLicenseContentRegexps(components)
	if !reflect.DeepEqual(contentPatterns, []string{"GPL", " "}) {
		t.Fatalf("content patterns = %v", contentPatterns)
	}

	compiled, err := compileRegexps(append(pathPatterns, contentPatterns...))
	if err != nil {
		t.Fatalf("compileRegexps returned error: %v", err)
	}
	if len(compiled) != 2 {
		t.Fatalf("expected 2 compiled regexps, got %d", len(compiled))
	}
	if !matchesAnyRegexp(filepath.ToSlash("vendor/component.so"), compiled[:1]) {
		t.Fatal("expected path regexp to match")
	}
	if !matchesAnyRegexp("GPL-3.0", compiled[1:]) {
		t.Fatal("expected content regexp to match")
	}
}
