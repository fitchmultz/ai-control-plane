// package_test.go - Tests for RBAC config loading and validation.
//
// Purpose:
//   - Verify the tracked RBAC loader and cross-reference validation helpers.
//
// Responsibilities:
//   - Cover approval-authority parsing.
//   - Cover model and role reference validation.
//
// Scope:
//   - internal/rbac package behavior only.
//
// Usage:
//   - Run via `go test ./internal/rbac`.
//
// Invariants/Assumptions:
//   - Tests use temporary YAML fixtures and deterministic known-model sets.
package rbac

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestLoadFileAndValidateKnownModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roles.yaml")
	testutil.WriteFile(t, path, `roles:
  admin:
    description: Full access
    model_access: [all]
    budget_ceiling: 100
    can_approve: true
    can_assign_roles: true
    can_create_keys: true
    read_only: false
    approval_authority: [all]
  team-lead:
    description: Lead access
    model_access: [openai-gpt5.2]
    budget_ceiling: 50
    can_approve: true
    can_assign_roles: false
    can_create_keys: true
    read_only: false
    approval_authority:
      max_budget: 25
      models: [openai-gpt5.2]
model_tiers:
  standard: [openai-gpt5.2]
default_role: team-lead
user_role_assignments:
  alice: team-lead
`)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if !cfg.Roles["admin"].ApprovalAuthority.All {
		t.Fatal("expected admin approval authority to parse as all")
	}

	known := map[string]struct{}{"openai-gpt5.2": {}}
	issues := cfg.ValidateKnownModels(known, regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`))
	if len(issues) != 0 {
		t.Fatalf("ValidateKnownModels() issues = %v", issues)
	}
}

func TestValidateKnownModelsFlagsInvalidReferences(t *testing.T) {
	cfg := Config{
		Roles: map[string]Role{
			"Bad Role": {
				ModelAccess: []string{"unknown-model"},
				ApprovalAuthority: ApprovalAuthority{
					Present: true,
					Models:  []string{"other-unknown"},
				},
			},
		},
		ModelTiers:          map[string][]string{"standard": {"missing-model"}},
		DefaultRole:         "missing-role",
		UserRoleAssignments: map[string]string{"alice": "missing-role"},
	}

	issues := cfg.ValidateKnownModels(map[string]struct{}{"openai-gpt5.2": {}}, regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`))
	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		`default_role "missing-role" does not exist`,
		`role name "Bad Role" does not match contract pattern`,
		`references unknown model alias "unknown-model"`,
		`references unknown model alias "other-unknown"`,
		`references undefined role "missing-role"`,
		`references unknown model alias "missing-model"`,
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, issues)
		}
	}
}
