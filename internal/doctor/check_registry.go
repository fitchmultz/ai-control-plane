// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Define the stable default doctor check ordering without coupling that
//	registry to concrete check implementations.
//
// Responsibilities:
//   - Return the canonical execution order for default doctor checks.
//
// Scope:
//   - Check registration only.
//
// Usage:
//   - Called by cmd/acpctl doctor command execution.
//
// Invariants/Assumptions:
//   - Check order stays deterministic for human output and tests.
package doctor

// DefaultChecks returns all available diagnostic checks in execution order.
func DefaultChecks() []Check {
	return []Check{
		dockerAvailableCheck{},
		portsFreeCheck{},
		envVarsSetCheck{},
		gatewayHealthyCheck{},
		certificateHealthyCheck{},
		dbConnectableCheck{},
		configValidCheck{},
		credentialsValidCheck{},
		budgetFindingsCheck{},
		detectionsFindingsCheck{},
	}
}
