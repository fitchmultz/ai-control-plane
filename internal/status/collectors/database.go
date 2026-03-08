// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose a database status collector that consumes the shared typed database
//	service for both embedded and external runtime modes.
//
// Responsibilities:
//   - Convert typed database runtime summaries into status.ComponentStatus.
//   - Surface uniform operator guidance for configuration and connectivity failures.
//
// Non-scope:
//   - Does not execute collector-local SQL or subprocess logic.
//
// Invariants/Assumptions:
//   - Database summaries come from the shared typed database service.
//
// Scope:
//   - Database status collection only.
//
// Usage:
//   - Construct with NewDatabaseCollector(client) and call Collect(ctx).
package collectors

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// DatabaseCollector checks PostgreSQL connectivity and metrics.
type DatabaseCollector struct {
	runtime db.RuntimeServiceReader
}

// NewDatabaseCollector creates a new database collector.
func NewDatabaseCollector(runtime db.RuntimeServiceReader) DatabaseCollector {
	return DatabaseCollector{runtime: runtime}
}

// Name returns the collector's domain name.
func (c DatabaseCollector) Name() string {
	return "database"
}

// Collect gathers database status information.
func (c DatabaseCollector) Collect(ctx context.Context) status.ComponentStatus {
	summary, err := c.runtime.Summary(ctx)
	details := status.ComponentDetails{
		Mode:           summary.Mode.String(),
		DatabaseName:   summary.DatabaseName,
		DatabaseUser:   summary.DatabaseUser,
		ContainerID:    summary.ContainerID,
		ExpectedTables: summary.ExpectedTables,
		Version:        summary.Version,
		Size:           summary.Size,
		Connections:    summary.Connections,
	}
	if summary.Ping.Error != "" {
		details.Error = summary.Ping.Error
	}

	if c.runtime.ConfigError() != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: "Database configuration is ambiguous",
			Details: details,
			Suggestions: []string{
				"Set ACP_DATABASE_MODE=embedded for the local demo stack",
				"Or set ACP_DATABASE_MODE=external when using DATABASE_URL",
			},
		}
	}

	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: "PostgreSQL is not accepting connections",
			Details: details,
			Suggestions: []string{
				"Check PostgreSQL connectivity and credentials",
				"Start or restart services: make up / make restart",
			},
		}
	}

	if summary.ExpectedTables < 4 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Database accessible, but schema is incomplete",
			Details: details,
			Suggestions: []string{
				"Run the stack long enough for LiteLLM schema initialization",
				"Verify LiteLLM database migrations completed",
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: "Connected",
		Details: details,
	}
}
