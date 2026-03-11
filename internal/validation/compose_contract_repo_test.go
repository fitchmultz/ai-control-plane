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
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

func TestSupportedComposeProfilesResolveExpectedServices(t *testing.T) {
	repoRoot := repoRootForTrackedComposeContracts(t)
	compose, baseArgs := dockerComposeCommand(t, repoRoot)
	cases := []struct {
		name        string
		files       []string
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name:        "base",
			files:       []string{"demo/docker-compose.yml"},
			wantPresent: []string{"litellm", "postgres"},
			wantAbsent:  []string{"presidio-analyzer", "librechat", "mock-upstream", "caddy"},
		},
		{
			name:        "base-dlp",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.dlp.yml"},
			wantPresent: []string{"litellm", "postgres", "presidio-analyzer", "presidio-anonymizer"},
			wantAbsent:  []string{"librechat", "mock-upstream", "caddy"},
		},
		{
			name:        "base-ui",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.ui.yml"},
			wantPresent: []string{"litellm", "postgres", "librechat", "librechat-mongodb", "librechat-meilisearch"},
			wantAbsent:  []string{"presidio-analyzer", "mock-upstream", "caddy"},
		},
		{
			name:        "base-offline",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.offline.yml"},
			wantPresent: []string{"litellm", "postgres", "mock-upstream"},
			wantAbsent:  []string{"presidio-analyzer", "librechat", "caddy"},
		},
		{
			name:        "base-tls",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.tls.yml"},
			wantPresent: []string{"litellm", "postgres", "caddy"},
			wantAbsent:  []string{"presidio-analyzer", "librechat", "mock-upstream"},
		},
		{
			name:        "base-ui-dlp",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.ui.yml", "demo/docker-compose.dlp.yml"},
			wantPresent: []string{"litellm", "postgres", "librechat", "librechat-mongodb", "librechat-meilisearch", "presidio-analyzer", "presidio-anonymizer"},
			wantAbsent:  []string{"mock-upstream", "caddy"},
		},
		{
			name:        "base-tls-ui",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.tls.yml", "demo/docker-compose.ui.yml"},
			wantPresent: []string{"litellm", "postgres", "caddy", "librechat", "librechat-mongodb", "librechat-meilisearch"},
			wantAbsent:  []string{"presidio-analyzer", "mock-upstream"},
		},
		{
			name:        "base-tls-dlp",
			files:       []string{"demo/docker-compose.yml", "demo/docker-compose.tls.yml", "demo/docker-compose.dlp.yml"},
			wantPresent: []string{"litellm", "postgres", "caddy", "presidio-analyzer", "presidio-anonymizer"},
			wantAbsent:  []string{"librechat", "mock-upstream"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{}, baseArgs...)
			args = append(args, "--profile", "embedded-db")
			for _, file := range tc.files {
				args = append(args, "-f", filepath.Join(repoRoot, filepath.FromSlash(file)))
			}
			args = append(args, "config", "--services")
			cmd := exec.Command(compose, args...)
			cmd.Dir = repoRoot
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("compose config %s failed: %v\n%s", tc.name, err, string(output))
			}
			services := strings.Fields(string(output))
			present := make(map[string]struct{}, len(services))
			for _, service := range services {
				present[service] = struct{}{}
			}
			for _, service := range tc.wantPresent {
				if _, ok := present[service]; !ok {
					t.Fatalf("%s missing expected service %q in %v", tc.name, service, services)
				}
			}
			for _, service := range tc.wantAbsent {
				if _, ok := present[service]; ok {
					t.Fatalf("%s unexpectedly included %q in %v", tc.name, service, services)
				}
			}
		})
	}
}

func dockerComposeCommand(t *testing.T, repoRoot string) (string, []string) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "compose", "version")
		cmd.Dir = repoRoot
		if err := cmd.Run(); err == nil {
			return "docker", []string{"compose", "--env-file", filepath.Join(repoRoot, "demo", ".env"), "--project-name", "ai-control-plane-contracts"}
		}
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		cmd := exec.Command("docker-compose", "version")
		cmd.Dir = repoRoot
		if err := cmd.Run(); err == nil {
			return "docker-compose", []string{"--env-file", filepath.Join(repoRoot, "demo", ".env"), "--project-name", "ai-control-plane-contracts"}
		}
	}
	t.Skip("docker compose not available")
	return "", nil
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
