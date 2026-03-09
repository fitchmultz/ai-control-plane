// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Provide shared status-construction helpers so collectors can focus on
//	domain-specific summary interpretation instead of repeating boilerplate.
//
// Responsibilities:
//   - Centralize common ComponentStatus construction.
//   - Stamp shared readonly-query failure guidance consistently.
//   - Attach query errors to status detail payloads in one place.
//
// Scope:
//   - Collector-local helper logic only.
//
// Usage:
//   - Used by sibling collector implementations in this package.
//
// Invariants/Assumptions:
//   - Helpers remain presentation-only and do not perform I/O or queries.
package collectors

import "github.com/mitchfultz/ai-control-plane/internal/status"

const readonlyBootstrapSuggestion = "Table may not exist yet - LiteLLM creates tables on first use"

func componentStatus(name string, level status.HealthLevel, message string, details status.ComponentDetails, suggestions ...string) status.ComponentStatus {
	return status.ComponentStatus{
		Name:        name,
		Level:       level,
		Message:     message,
		Details:     details,
		Suggestions: append([]string(nil), suggestions...),
	}
}

func withDetailError(details status.ComponentDetails, err error) status.ComponentDetails {
	if err == nil {
		return details
	}
	details.Error = err.Error()
	return details
}

func readonlyQueryWarning(name string, message string, details status.ComponentDetails, err error) status.ComponentStatus {
	return componentStatus(
		name,
		status.HealthLevelWarning,
		message,
		withDetailError(details, err),
		readonlyBootstrapSuggestion,
	)
}
