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
	"context"
	"fmt"
	"strings"

	sharedhealth "github.com/mitchfultz/ai-control-plane/internal/health"
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

func newCheckResult(id string, name string, level sharedhealth.Level, severity Severity, message string) CheckResult {
	return CheckResult{
		ID:       id,
		Name:     name,
		Level:    level,
		Severity: severity,
		Message:  message,
	}
}

func withCheckDetails(result CheckResult, details status.ComponentDetails, suggestions ...string) CheckResult {
	result.Details = details
	if len(suggestions) > 0 {
		result.Suggestions = append([]string(nil), suggestions...)
	}
	return result
}

func withComponentStatus(result CheckResult, component status.ComponentStatus) CheckResult {
	result.Details = component.Details
	result.Suggestions = append([]string(nil), component.Suggestions...)
	return result
}

func componentCheckResult(id string, name string, component status.ComponentStatus, severity Severity) CheckResult {
	return withComponentStatus(
		newCheckResult(id, name, component.Level, severity, component.Message),
		component,
	)
}

func runtimeComponentCheck(opts Options, id string, name string, componentKey string, componentLabel string, adapt func(status.ComponentStatus) CheckResult) CheckResult {
	component, ok := runtimeComponent(opts, componentKey)
	if !ok {
		return runtimeInspectionMissing(id, name, componentLabel)
	}
	return adapt(component)
}

func runtimeInspectionMissing(id string, name string, component string) CheckResult {
	return newCheckResult(
		id,
		name,
		sharedhealth.LevelUnknown,
		SeverityRuntime,
		fmt.Sprintf("%s runtime inspection did not produce a result", component),
	)
}

type noFixCheck struct{}

func (noFixCheck) Fix(context.Context, Options) (bool, string, error) {
	return false, "", nil
}
