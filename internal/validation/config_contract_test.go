// config_contract_test.go - Focused coverage for config-contract validation.
//
// Purpose:
//   - Verify schema-backed config-contract validation and cross-file invariants.
//
// Responsibilities:
//   - Cover invalid preset names, RBAC model drift, and overlay validation.
//   - Cover YAML-to-JSON normalization helpers used by schema validation.
//
// Scope:
//   - Config-contract helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixture repositories and deterministic contract files.
package validation

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidateConfigContractFlagsCrossFileIssues(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidConfigContractRepo(t, repoRoot)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.example.yml"), "all:\n  vars:\n    acp_runtime_overlays: [bad-overlay]\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "roles.yaml"), "roles:\n  developer:\n    description: Developer\n    model_access: [missing-model]\n    budget_ceiling: 25\n    can_approve: false\n    can_assign_roles: false\n    can_create_keys: true\n    read_only: false\n    approval_authority: null\ndefault_role: developer\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "demo_presets.yaml"), "presets:\n  Bad Preset:\n    name: Bad Preset\n    description: Invalid preset name\n    timeout_minutes: 5\n    scenarios: [1]\n    stop_on_fail: true\n    intro_message: hello\nsettings:\n  default_timeout_minutes: 5\n  scenario_delay_seconds: 0\n  colors_enabled: true\n")

	issues, err := ValidateConfigContract(repoRoot)
	if err != nil {
		t.Fatalf("ValidateConfigContract() error = %v", err)
	}
	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		`unknown model alias "missing-model"`,
		`preset name "Bad Preset" does not match contract pattern`,
		`unsupported acp_runtime_overlays value "bad-overlay"`,
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, issues)
		}
	}
}

func TestValidateConfigContractFlagsUnsupportedHostInventoryURLContract(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidConfigContractRepo(t, repoRoot)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.example.yml"), "all:\n  children:\n    gateway:\n      hosts:\n        gateway:\n          acp_runtime_overlays: []\n          acp_public_url: http://gateway.example.com:4000\n")

	issues, err := ValidateConfigContract(repoRoot)
	if err != nil {
		t.Fatalf("ValidateConfigContract() error = %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, `acp_public_url "http://gateway.example.com:4000" must stay loopback-only`) {
		t.Fatalf("expected host inventory URL contract issue, got %v", issues)
	}
}

func TestNormalizeYAMLAndOverlayValuesHelpers(t *testing.T) {
	normalized := normalizeYAML(map[any]any{
		"outer": []any{map[any]any{"inner": true}},
	})
	mapped, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("normalizeYAML() type = %T", normalized)
	}
	if _, ok := mapped["outer"]; !ok {
		t.Fatalf("normalizeYAML() missing converted key: %v", mapped)
	}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte("[tls, ui]"), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	values := overlayValues(node.Content[0])
	if strings.Join(values, ",") != "tls,ui" {
		t.Fatalf("overlayValues() = %v", values)
	}

	if issues := validateInventoryOverlayContract(filepath.Join(t.TempDir(), "missing.yml"), []string{"tls"}); len(issues) != 0 {
		t.Fatalf("validateInventoryOverlayContract() missing file issues = %v", issues)
	}
}

func TestLoadConfigContractManifestRejectsInvalidManifest(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, configContractManifestRelativePath), "version: 0\nschemas: []\n")
	if _, err := loadConfigContractManifest(repoRoot); err == nil {
		t.Fatal("expected loadConfigContractManifest() to reject invalid manifest")
	}
}
