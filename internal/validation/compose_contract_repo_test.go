// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Lock down the tracked host-first compose topology and overlay boundaries.
//
// Responsibilities:
//   - Ensure the base compose file stays limited to the supported core stack.
//   - Ensure overlay files add only their declared services and do not redefine
//     base services.
//
// Scope:
//   - Repository-contract tests against the tracked compose files.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - The supported base runtime is LiteLLM, PostgreSQL, and the production-only
//     OTEL collector.
//   - Optional overlays must remain additive and explicit.
package validation

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

type trackedComposeFile struct {
	Services map[string]any `yaml:"services"`
}

func TestTrackedComposeBaseStackRemainsMinimal(t *testing.T) {
	repoRoot := repoRootForTrackedComposeContracts(t)
	base := loadTrackedComposeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"))

	want := []string{"litellm", "otel-collector", "postgres"}
	if got := sortedServiceNames(base); !reflect.DeepEqual(got, want) {
		t.Fatalf("base compose services = %v, want %v", got, want)
	}
}

func TestTrackedComposeOverlaysRemainAdditive(t *testing.T) {
	repoRoot := repoRootForTrackedComposeContracts(t)
	base := loadTrackedComposeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"))
	baseServices := serviceNameSet(base)

	cases := []struct {
		path string
		want []string
	}{
		{path: filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), want: []string{"mock-upstream"}},
		{path: filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), want: []string{"librechat", "librechat-meilisearch", "librechat-mongodb"}},
		{path: filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), want: []string{"presidio-analyzer", "presidio-anonymizer"}},
	}

	for _, tc := range cases {
		overlay := loadTrackedComposeFile(t, tc.path)
		if got := sortedServiceNames(overlay); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%s services = %v, want %v", filepath.Base(tc.path), got, tc.want)
		}
		for service := range overlay.Services {
			if _, exists := baseServices[service]; exists {
				t.Fatalf("%s redefines base service %q", filepath.Base(tc.path), service)
			}
		}
	}
}

func repoRootForTrackedComposeContracts(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("could not locate repo root from %s", wd)
		}
		current = parent
	}
}

func loadTrackedComposeFile(t *testing.T, path string) trackedComposeFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var compose trackedComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return compose
}

func sortedServiceNames(compose trackedComposeFile) []string {
	names := make([]string, 0, len(compose.Services))
	for name := range compose.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func serviceNameSet(compose trackedComposeFile) map[string]struct{} {
	values := make(map[string]struct{}, len(compose.Services))
	for name := range compose.Services {
		values[name] = struct{}{}
	}
	return values
}
