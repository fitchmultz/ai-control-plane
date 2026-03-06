// validator.go - Key generation validation logic
//
// Purpose: Validate key generation configuration
//
// Responsibilities:
//   - Validate alias format
//   - Validate role values
//   - Get models for roles
//   - Check prerequisites
//
// Non-scope:
//   - Does not parse arguments (see parser.go)
//   - Does not generate keys
package keygen

import (
	"fmt"
	"os"
	"regexp"
	"strings"
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

// ValidateRole checks if the role is valid
func ValidateRole(role string) error {
	validRoles := []string{"admin", "team-lead", "developer", "auditor", ""}
	for _, r := range validRoles {
		if role == r {
			return nil
		}
	}
	return &ValidationError{Field: "role", Message: fmt.Sprintf("invalid role: %s", role)}
}

// GetModelsForRole returns the allowed models for a role
func GetModelsForRole(role string) []string {
	switch role {
	case "admin":
		return []string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5", "claude-opus-4-5"}
	case "team-lead":
		return []string{"openai-gpt5.2", "claude-haiku-4-5", "claude-sonnet-4-5"}
	case "developer":
		return []string{"openai-gpt5.2", "claude-haiku-4-5"}
	case "auditor":
		return []string{} // No model access
	default:
		// Default to developer models
		return []string{"openai-gpt5.2", "claude-haiku-4-5"}
	}
}

// ResolveRole determines the effective role
func ResolveRole(explicitRole string) string {
	if explicitRole != "" {
		return explicitRole
	}
	role := os.Getenv("ACP_USER_ROLE")
	if role != "" {
		return role
	}
	return "developer" // default
}

// CheckPrerequisites verifies required environment and tools
func CheckPrerequisites(requireMasterKey bool) error {
	if requireMasterKey {
		masterKey := os.Getenv("LITELLM_MASTER_KEY")
		if masterKey == "" {
			return &ValidationError{
				Field:   "LITELLM_MASTER_KEY",
				Message: "environment variable is required",
			}
		}
	}
	return nil
}

// ValidRoles returns the list of valid role names
func ValidRoles() []string {
	return []string{"admin", "team-lead", "developer", "auditor"}
}

// FormatValidationErrors formats multiple validation errors
func FormatValidationErrors(errors []error) string {
	var msgs []string
	for _, err := range errors {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}
