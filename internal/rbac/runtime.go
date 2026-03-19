// Package rbac owns typed RBAC contract semantics used by operator workflows.
//
// Purpose:
//   - Provide deterministic role-name, role-resolution, and model-access helpers
//     on top of the tracked RBAC contract.
//
// Responsibilities:
//   - Resolve the effective role from explicit, configured, and default values.
//   - Expand role model access against the approved model catalog.
//   - Infer the least-privileged matching role for an existing model set.
//
// Scope:
//   - In-memory RBAC contract semantics only.
//
// Usage:
//   - Used by key-generation and future RBAC-aware workflows.
//
// Invariants/Assumptions:
//   - Approved model aliases are already normalized repository-approved names.
//   - The special RBAC wildcard remains `all`.
package rbac

import (
	"slices"
	"sort"
	"strings"
)

// RoleNames returns deterministic tracked role names.
func (c Config) RoleNames() []string {
	names := make([]string, 0, len(c.Roles))
	for name := range c.Roles {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	sort.Strings(names)
	return names
}

// HasRole reports whether the tracked contract contains the named role.
func (c Config) HasRole(role string) bool {
	_, ok := c.Roles[strings.TrimSpace(role)]
	return ok
}

// DefaultRoleName returns the tracked default role.
func (c Config) DefaultRoleName() string {
	return strings.TrimSpace(c.DefaultRole)
}

// ResolveRole returns the effective role from explicit, configured, or default values.
func (c Config) ResolveRole(explicitRole string, configuredRole string) string {
	if trimmed := strings.TrimSpace(explicitRole); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(configuredRole); trimmed != "" {
		return trimmed
	}
	return c.DefaultRoleName()
}

// ModelsForRole expands the tracked role's model access against approved aliases.
func (c Config) ModelsForRole(role string, approvedModels []string) []string {
	resolved := strings.TrimSpace(role)
	if resolved == "" {
		resolved = c.DefaultRoleName()
	}
	roleConfig, ok := c.Roles[resolved]
	if !ok {
		return nil
	}
	if containsAll(roleConfig.ModelAccess) {
		return dedupeSortedStrings(approvedModels)
	}
	allowed := make(map[string]struct{}, len(roleConfig.ModelAccess))
	for _, model := range roleConfig.ModelAccess {
		trimmed := strings.TrimSpace(model)
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}
	models := make([]string, 0, len(allowed))
	for _, model := range dedupeSortedStrings(approvedModels) {
		if _, ok := allowed[model]; ok {
			models = append(models, model)
		}
	}
	return models
}

// InferLeastPrivilegedRole picks the least-privileged tracked role for a model set.
func (c Config) InferLeastPrivilegedRole(models []string, approvedModels []string) string {
	normalized := dedupeSortedStrings(models)
	if len(normalized) == 0 {
		if c.HasRole("auditor") {
			return "auditor"
		}
		return c.DefaultRoleName()
	}

	candidates := c.rolesByPrivilege(approvedModels)
	for _, role := range candidates {
		if slices.Equal(normalized, c.ModelsForRole(role, approvedModels)) {
			return role
		}
	}
	for _, role := range candidates {
		if isModelSubset(normalized, c.ModelsForRole(role, approvedModels)) {
			return role
		}
	}
	return c.DefaultRoleName()
}

func (c Config) rolesByPrivilege(approvedModels []string) []string {
	names := c.RoleNames()
	sort.SliceStable(names, func(i, j int) bool {
		left := c.Roles[names[i]]
		right := c.Roles[names[j]]

		leftModels := len(c.ModelsForRole(names[i], approvedModels))
		rightModels := len(c.ModelsForRole(names[j], approvedModels))
		if leftModels != rightModels {
			return leftModels < rightModels
		}
		if left.CanCreateKeys != right.CanCreateKeys {
			return !left.CanCreateKeys && right.CanCreateKeys
		}
		if left.CanApprove != right.CanApprove {
			return !left.CanApprove && right.CanApprove
		}
		if left.CanAssignRoles != right.CanAssignRoles {
			return !left.CanAssignRoles && right.CanAssignRoles
		}
		if left.BudgetCeiling != right.BudgetCeiling {
			return left.BudgetCeiling < right.BudgetCeiling
		}
		return names[i] < names[j]
	})
	return names
}

func containsAll(values []string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), "all") {
			return true
		}
	}
	return false
}

func dedupeSortedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func isModelSubset(current []string, candidate []string) bool {
	candidateSet := make(map[string]struct{}, len(candidate))
	for _, model := range candidate {
		candidateSet[model] = struct{}{}
	}
	for _, model := range current {
		if _, ok := candidateSet[model]; !ok {
			return false
		}
	}
	return true
}
