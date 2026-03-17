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

func (c dbConnectableCheck) Run(_ context.Context, opts Options) CheckResult {
	return runtimeComponentCheck(opts, c.ID(), "Database Connectable", "database", "Database", func(component status.ComponentStatus) CheckResult {
		return componentCheckResult(c.ID(), "Database Connectable", component, databaseSeverity(component))
	})
}

func (c dbConnectableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return recoverEmbeddedDatabase(ctx, opts)
}

func databaseSeverity(component status.ComponentStatus) Severity {
	if component.Details.LookupError == status.LookupErrorDatabaseConfigAmbiguous {
		return SeverityPrereq
	}
	return severityForLevel(component.Level)
}
