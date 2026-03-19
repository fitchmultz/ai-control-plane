// validator.go - Key generation validation logic
//
// Purpose: Validate key generation configuration
//
// Responsibilities:
//   - Validate alias format
//   - Validate role values against the tracked RBAC contract
//   - Resolve effective roles and model access for key workflows
//   - Check prerequisites
//
// Non-scope:
//   - Does not parse arguments (see parser.go)
//   - Does not generate keys
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package keygen

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateAlias checks if the alias format is valid
func ValidateAlias(alias string) error {
	if alias == "" {
		return &ValidationError{Field: "alias", Message: "alias is required"}
	}
	if len(alias) > 64 {
		return &ValidationError{Field: "alias", Message: "alias must be 64 characters or less"}
	}
	// Allow alphanumeric, ., _, -
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, alias)
	if !matched {
		return &ValidationError{Field: "alias", Message: "must be alphanumeric + . _ -"}
	}
	return nil
}

// ValidateRole checks if the role is valid.
func ValidateRole(role string) error {
	trimmed := strings.TrimSpace(role)
	if trimmed == "" {
		return nil
	}
	cfg, _, err := loadTrackedRoleContract()
	if err != nil {
		return fmt.Errorf("load tracked RBAC contract: %w", err)
	}
	if cfg.HasRole(trimmed) {
		return nil
	}
	return &ValidationError{Field: "role", Message: fmt.Sprintf("invalid role: %s", trimmed)}
}

// GetModelsForRole returns the allowed models for a role.
func GetModelsForRole(role string) ([]string, error) {
	cfg, approvedModels, err := loadTrackedRoleContract()
	if err != nil {
		return nil, fmt.Errorf("load tracked RBAC contract: %w", err)
	}
	models := cfg.ModelsForRole(role, approvedModels)
	if models == nil {
		return nil, &ValidationError{Field: "role", Message: fmt.Sprintf("invalid role: %s", strings.TrimSpace(role))}
	}
	return models, nil
}

// ResolveRole determines the effective role.
func ResolveRole(explicitRole string) (string, error) {
	cfg, _, err := loadTrackedRoleContract()
	if err != nil {
		return "", fmt.Errorf("load tracked RBAC contract: %w", err)
	}
	role := cfg.ResolveRole(explicitRole, config.NewLoader().Tooling().Role)
	if strings.TrimSpace(role) == "" {
		return "", &ValidationError{Field: "role", Message: "no default role configured"}
	}
	return role, nil
}

// CheckPrerequisites verifies required environment and tools
func CheckPrerequisites(requireMasterKey bool) error {
	if requireMasterKey {
		masterKey := config.NewLoader().Gateway(true).MasterKey
		if masterKey == "" {
			return &ValidationError{
				Field:   "LITELLM_MASTER_KEY",
				Message: "environment variable is required",
			}
		}
	}
	return nil
}

// ValidRoles returns the list of valid role names.
func ValidRoles() []string {
	cfg, _, err := loadTrackedRoleContract()
	if err != nil {
		return nil
	}
	return cfg.RoleNames()
}

// FormatValidationErrors formats multiple validation errors
func FormatValidationErrors(errors []error) string {
	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}
