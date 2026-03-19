// runtime_test.go - Tests for RBAC runtime helpers.
//
// Purpose:
//   - Verify tracked role semantics stay aligned with the RBAC contract.
//
// Responsibilities:
//   - Cover role-name resolution.
//   - Cover wildcard model expansion and least-privileged inference.
//
// Scope:
//   - in-memory internal/rbac runtime helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/rbac`.
//
// Invariants/Assumptions:
//   - Tests use deterministic in-memory RBAC fixtures.
package rbac

import (
	"slices"
	"testing"
)

func TestResolveRoleUsesExplicitConfiguredAndDefaultValues(t *testing.T) {
	cfg := Config{DefaultRole: "developer"}

	if got := cfg.ResolveRole("admin", "team-lead"); got != "admin" {
		t.Fatalf("ResolveRole() explicit = %q, want admin", got)
	}
	if got := cfg.ResolveRole("", "team-lead"); got != "team-lead" {
		t.Fatalf("ResolveRole() configured = %q, want team-lead", got)
	}
	if got := cfg.ResolveRole("", ""); got != "developer" {
		t.Fatalf("ResolveRole() default = %q, want developer", got)
	}
}

func TestModelsForRoleExpandsWildcardAndIntersectsApprovedAliases(t *testing.T) {
	cfg := Config{
		Roles: map[string]Role{
			"admin":     {ModelAccess: []string{"all"}},
			"developer": {ModelAccess: []string{"claude-haiku-4-5", "openai-gpt5.2", "missing-model"}},
		},
		DefaultRole: "developer",
	}
	approved := []string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5"}

	if got := cfg.ModelsForRole("admin", approved); !slices.Equal(got, []string{"claude-haiku-4-5", "claude-sonnet-4-5", "openai-gpt5.2"}) {
		t.Fatalf("admin models = %v", got)
	}
	if got := cfg.ModelsForRole("developer", approved); !slices.Equal(got, []string{"claude-haiku-4-5", "openai-gpt5.2"}) {
		t.Fatalf("developer models = %v", got)
	}
}

func TestInferLeastPrivilegedRolePrefersLowerPrivilegeMatch(t *testing.T) {
	cfg := Config{
		Roles: map[string]Role{
			"admin": {
				ModelAccess:    []string{"all"},
				CanCreateKeys:  true,
				CanApprove:     true,
				CanAssignRoles: true,
				BudgetCeiling:  500,
			},
			"team-lead": {
				ModelAccess:    []string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5"},
				CanCreateKeys:  true,
				CanApprove:     true,
				CanAssignRoles: false,
				BudgetCeiling:  100,
			},
			"developer": {
				ModelAccess:    []string{"openai-gpt5.2", "claude-haiku-4-5"},
				CanCreateKeys:  true,
				CanApprove:     false,
				CanAssignRoles: false,
				BudgetCeiling:  25,
			},
			"auditor": {
				ModelAccess:    []string{},
				CanCreateKeys:  false,
				CanApprove:     false,
				CanAssignRoles: false,
				BudgetCeiling:  0,
			},
		},
		DefaultRole: "developer",
	}
	approved := []string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5"}

	if got := cfg.InferLeastPrivilegedRole([]string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5"}, approved); got != "team-lead" {
		t.Fatalf("InferLeastPrivilegedRole() exact = %q, want team-lead", got)
	}
	if got := cfg.InferLeastPrivilegedRole([]string{"openai-gpt5.2"}, approved); got != "developer" {
		t.Fatalf("InferLeastPrivilegedRole() subset = %q, want developer", got)
	}
	if got := cfg.InferLeastPrivilegedRole(nil, approved); got != "auditor" {
		t.Fatalf("InferLeastPrivilegedRole() empty = %q, want auditor", got)
	}
}
