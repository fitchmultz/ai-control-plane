// profile_test.go - Tests for benchmark profile configuration loading.
//
// Purpose:
//
//	Verify the benchmark profile catalog can be loaded and resolved from the
//	canonical JSON configuration.
//
// Responsibilities:
//   - Validate profile catalog parsing
//   - Verify unknown profile names are rejected
//
// Scope:
//   - Covers internal profile loading only
//
// Usage:
//   - Run via `go test ./internal/performance`
//
// Invariants/Assumptions:
//   - Tests use temporary JSON fixtures instead of the repository file
package performance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProfileCatalogAndResolveProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := os.WriteFile(path, []byte(`{
  "schema_version": "1.0.0",
  "defaults": {"warmup_requests": 5, "measured_requests": 40, "concurrency": 2},
  "profiles": {
    "interactive": {
      "description": "Interactive",
      "workload": {"warmup_requests": 5, "measured_requests": 30, "concurrency": 1}
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	catalog, err := LoadProfileCatalog(path)
	if err != nil {
		t.Fatalf("LoadProfileCatalog() error = %v", err)
	}
	profile, err := catalog.ResolveProfile("interactive")
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if profile.Workload.MeasuredRequests != 30 {
		t.Fatalf("measured requests = %d, want 30", profile.Workload.MeasuredRequests)
	}
}

func TestResolveProfileUnknown(t *testing.T) {
	catalog := &ProfileCatalog{
		Profiles: map[string]BenchmarkProfile{
			"interactive": {Description: "Interactive"},
		},
	}
	_, err := catalog.ResolveProfile("burst")
	if err == nil {
		t.Fatal("expected unknown profile error")
	}
	if !strings.Contains(err.Error(), "interactive") {
		t.Fatalf("expected available profile list in error, got %v", err)
	}
}
