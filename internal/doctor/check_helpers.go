// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Provide shared helper functions used across focused doctor check modules.
//
// Responsibilities:
//   - Reuse runtime inspection results across focused doctor checks.
//   - Normalize subprocess/network helper output for checks.
//   - Sanitize potentially sensitive subprocess output.
//
// Scope:
//   - Shared helper utilities only.
//
// Usage:
//   - Imported implicitly by other files in the doctor package.
//
// Invariants/Assumptions:
//   - Helpers never log raw secret material.
//   - Empty helper results are safe fallbacks for doctor checks.
package doctor

import (
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func firstNonEmptyLine(raw string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sanitizeOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "key") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "database_url") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func runtimeComponent(opts Options, name string) (status.ComponentStatus, bool) {
	if opts.RuntimeReport == nil {
		return status.ComponentStatus{}, false
	}
	component, ok := opts.RuntimeReport.Components[name]
	return component, ok
}
