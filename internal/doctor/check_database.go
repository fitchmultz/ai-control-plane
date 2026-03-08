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

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type dbConnectableCheck struct{}

func (c dbConnectableCheck) ID() string { return "db_connectable" }

func (c dbConnectableCheck) Run(ctx context.Context, opts Options) CheckResult {
	client := db.NewClient(opts.RepoRoot)
	defer client.Close()

	summary, err := client.Summary(ctx)
	details := status.ComponentDetails{
		Mode:         summary.Mode.String(),
		DatabaseName: summary.DatabaseName,
		DatabaseUser: summary.DatabaseUser,
		ContainerID:  summary.ContainerID,
		Error:        summary.Ping.Error,
	}
	if client.ConfigError() != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "Database configuration is ambiguous",
			Details:  details,
			Suggestions: []string{
				"Set ACP_DATABASE_MODE=embedded for the local demo stack",
				"Or set ACP_DATABASE_MODE=external when using DATABASE_URL",
			},
		}
	}
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "PostgreSQL is not accepting connections",
			Details:  details,
			Suggestions: []string{
				"Check PostgreSQL connectivity and credentials",
				"Restart services: make restart",
			},
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Database Connectable",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "PostgreSQL is accepting connections",
		Details:  details,
	}
}

func (c dbConnectableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
