// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement database connectivity diagnostics through the shared typed
//	database service so embedded and external modes behave consistently.
//
// Responsibilities:
//   - Verify the effective PostgreSQL runtime accepts connections.
//   - Surface typed database mode and container metadata when available.
//
// Non-scope:
//   - Does not execute schema mutations or remediation actions.
//
// Invariants/Assumptions:
//   - Database diagnostics consume the shared typed database service.
//
// Scope:
//   - Database health diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package doctor

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type dbConnectableCheck struct{}

func (c dbConnectableCheck) ID() string { return "db_connectable" }

func (c dbConnectableCheck) Run(ctx context.Context, opts Options) CheckResult {
	component, ok := runtimeComponent(opts, "database")
	if !ok {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnknown,
			Severity: SeverityRuntime,
			Message:  "Database runtime inspection did not produce a result",
		}
	}

	return CheckResult{
		ID:          c.ID(),
		Name:        "Database Connectable",
		Level:       component.Level,
		Severity:    databaseSeverity(component),
		Message:     component.Message,
		Details:     component.Details,
		Suggestions: component.Suggestions,
	}
}

func (c dbConnectableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}

func databaseSeverity(component status.ComponentStatus) Severity {
	if component.Message == "Database configuration is ambiguous" {
		return SeverityPrereq
	}
	return severityForLevel(component.Level)
}
