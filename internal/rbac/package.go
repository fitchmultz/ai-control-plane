// Package rbac owns the tracked RBAC contract loaded from demo/config/roles.yaml.
//
// Purpose:
//   - Provide typed loading and validation for the repository RBAC contract.
//
// Responsibilities:
//   - Load roles, tiers, defaults, and user-role assignments from YAML.
//   - Validate role references and model references against known aliases.
//   - Expose a stable typed contract for future RBAC-aware workflows.
//
// Scope:
//   - RBAC config loading and validation only.
//
// Usage:
//   - Used by validation and future RBAC-aware workflows.
//
// Invariants/Assumptions:
//   - The special model wildcard is `all`.
//   - default_role must resolve to a declared role.
package rbac

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config captures the tracked RBAC contract.
type Config struct {
	Roles               map[string]Role     `yaml:"roles"`
	ModelTiers          map[string][]string `yaml:"model_tiers"`
	DefaultRole         string              `yaml:"default_role"`
	UserRoleAssignments map[string]string   `yaml:"user_role_assignments"`
}

// Role captures one tracked RBAC role definition.
type Role struct {
	Description       string            `yaml:"description"`
	ModelAccess       []string          `yaml:"model_access"`
	BudgetCeiling     float64           `yaml:"budget_ceiling"`
	CanApprove        bool              `yaml:"can_approve"`
	CanAssignRoles    bool              `yaml:"can_assign_roles"`
	CanCreateKeys     bool              `yaml:"can_create_keys"`
	ReadOnly          bool              `yaml:"read_only"`
	ApprovalAuthority ApprovalAuthority `yaml:"approval_authority"`
}

// ApprovalAuthority captures the supported approval-authority YAML shapes.
type ApprovalAuthority struct {
	Present   bool
	All       bool
	MaxBudget *float64 `yaml:"max_budget,omitempty"`
	Models    []string `yaml:"models,omitempty"`
}

// UnmarshalYAML accepts null, [all], or a mapping with model/budget limits.
func (a *ApprovalAuthority) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind == 0 || value.Tag == "!!null" {
		*a = ApprovalAuthority{}
		return nil
	}

	a.Present = true

	switch value.Kind {
	case yaml.ScalarNode:
		if strings.EqualFold(strings.TrimSpace(value.Value), "all") {
			a.All = true
			return nil
		}
		return fmt.Errorf("approval_authority scalar must be 'all' or null")
	case yaml.SequenceNode:
		if len(value.Content) == 1 && strings.EqualFold(strings.TrimSpace(value.Content[0].Value), "all") {
			a.All = true
			return nil
		}
		return fmt.Errorf("approval_authority sequence form only supports ['all']")
	case yaml.MappingNode:
		var raw struct {
			MaxBudget *float64 `yaml:"max_budget"`
			Models    []string `yaml:"models"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		a.MaxBudget = raw.MaxBudget
		a.Models = raw.Models
		return nil
	default:
		return fmt.Errorf("unsupported approval_authority YAML shape")
	}
}

// LoadFile loads the tracked RBAC YAML file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read RBAC config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse RBAC config %s: %w", path, err)
	}
	return cfg, nil
}

// ValidateKnownModels validates role, tier, and assignment references.
func (c Config) ValidateKnownModels(knownModels map[string]struct{}, roleNamePattern *regexp.Regexp) []string {
	issues := make([]string, 0)

	if len(c.Roles) == 0 {
		issues = append(issues, "demo/config/roles.yaml: roles must not be empty")
	}

	if strings.TrimSpace(c.DefaultRole) == "" {
		issues = append(issues, "demo/config/roles.yaml: default_role is required")
	} else if _, ok := c.Roles[c.DefaultRole]; !ok {
		issues = append(issues, fmt.Sprintf("demo/config/roles.yaml: default_role %q does not exist in roles", c.DefaultRole))
	}

	for roleName, role := range c.Roles {
		if roleNamePattern != nil && !roleNamePattern.MatchString(roleName) {
			issues = append(issues, fmt.Sprintf("demo/config/roles.yaml: role name %q does not match contract pattern", roleName))
		}
		issues = append(issues, validateModelRefs(role.ModelAccess, knownModels, fmt.Sprintf("roles.%s.model_access", roleName))...)
		if !role.ApprovalAuthority.All {
			issues = append(issues, validateModelRefs(role.ApprovalAuthority.Models, knownModels, fmt.Sprintf("roles.%s.approval_authority.models", roleName))...)
		}
	}

	for user, roleName := range c.UserRoleAssignments {
		if _, ok := c.Roles[roleName]; !ok {
			issues = append(issues, fmt.Sprintf("demo/config/roles.yaml: user_role_assignments[%q] references undefined role %q", user, roleName))
		}
	}

	for tierName, models := range c.ModelTiers {
		issues = append(issues, validateModelRefs(models, knownModels, fmt.Sprintf("model_tiers.%s", tierName))...)
	}

	sort.Strings(issues)
	return issues
}

func validateModelRefs(models []string, knownModels map[string]struct{}, field string) []string {
	if len(models) == 0 || len(knownModels) == 0 {
		return nil
	}
	issues := make([]string, 0)
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || strings.EqualFold(model, "all") {
			continue
		}
		if _, ok := knownModels[model]; !ok {
			issues = append(issues, fmt.Sprintf("demo/config/roles.yaml: %s references unknown model alias %q", field, model))
		}
	}
	return issues
}
