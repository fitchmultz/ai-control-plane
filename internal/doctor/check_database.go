// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement database connectivity diagnostics as a focused module.
//
// Responsibilities:
//   - Discover the running PostgreSQL container.
//   - Verify PostgreSQL accepts connections.
//
// Scope:
//   - Database health diagnostics only.
package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type dbConnectableCheck struct{}

func (c dbConnectableCheck) ID() string { return "db_connectable" }

func (c dbConnectableCheck) Run(ctx context.Context, opts Options) CheckResult {
	if _, err := exec.LookPath("docker"); err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnknown,
			Severity: SeverityPrereq,
			Message:  "Docker not available",
			Suggestions: []string{
				"Install Docker: https://docs.docker.com/get-docker/",
			},
		}
	}

	containerCmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=postgres", "--format", "{{.ID}}")
	containerOutput, err := containerCmd.Output()
	containerID := firstNonEmptyLine(string(containerOutput))
	if err != nil || containerID == "" {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "PostgreSQL container not running",
			Suggestions: []string{
				"Start services: make up",
				"Check container status: docker ps",
			},
		}
	}

	testCmd := exec.CommandContext(ctx, "docker", "exec", containerID, "psql", "-U", "litellm", "-d", "litellm", "-t", "-c", "SELECT 1;")
	testOutput, err := testCmd.Output()
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "PostgreSQL not accepting connections",
			Suggestions: []string{
				fmt.Sprintf("Check PostgreSQL logs: docker logs %s", containerID),
				"Restart services: make restart",
			},
		}
	}
	if !strings.Contains(string(testOutput), "1") {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  "PostgreSQL responded unexpectedly",
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Database Connectable",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "PostgreSQL is accepting connections",
	}
}

func (c dbConnectableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
