// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Lock down the tracked host-first playbook postcheck contract.
//
// Responsibilities:
//   - Ensure generic host health and smoke checks remain in place.
//   - Ensure supported overlays trigger their expected postchecks.
//
// Scope:
//   - Repository-contract tests against the tracked Ansible playbook.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Host overlay validation stays additive after generic health/smoke.
//   - UI, TLS, and DLP overlays retain explicit postchecks.
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrackedGatewayHostPlaybookRunsOverlaySpecificPostchecks(t *testing.T) {
	repoRoot := repoRootForTrackedComposeContracts(t)
	path := filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)

	requiredSnippets := []string{
		"argv:\n          - make\n          - health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - prod-smoke\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - librechat-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - tls-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - dlp-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		`when: "'ui' in acp_runtime_overlays"`,
		`when: "'tls' in acp_runtime_overlays"`,
		`when: "'dlp' in acp_runtime_overlays"`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("gateway_host.yml missing required overlay postcheck contract %q", snippet)
		}
	}
}
