// Package collectors provides domain-specific status collectors.
//
// DatabaseCollector uses PostgreSQL's information_schema and pg_stat_activity
// to collect database health metrics including table existence, version info,
// database size, and active connections.
package collectors

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

// DatabaseCollector checks PostgreSQL connectivity and metrics.
type DatabaseCollector struct {
	RepoRoot string
	runner   runner.Runner
	compose  containerIDResolver
}

// NewDatabaseCollector creates a new database collector
func NewDatabaseCollector(repoRoot string) *DatabaseCollector {
	return &DatabaseCollector{
		RepoRoot: repoRoot,
		runner:   newCollectorRunner(repoRoot),
		compose:  newCollectorCompose(repoRoot),
	}
}

// SetRunner sets a custom runner (for testing)
func (c *DatabaseCollector) SetRunner(r runner.Runner) {
	c.runner = r
}

// SetContainerResolver sets a custom container resolver (for testing)
func (c *DatabaseCollector) SetContainerResolver(resolver containerIDResolver) {
	c.compose = resolver
}

// Name returns the collector's domain name.
func (c DatabaseCollector) Name() string {
	return "database"
}

func (c DatabaseCollector) resolvePostgresContainer(ctx context.Context) (string, error) {
	return resolvePostgresContainer(ctx, resolveCollectorRuntime(c.RepoRoot, c.runner, c.compose))
}

func (c DatabaseCollector) runQuery(ctx context.Context, containerID, query string) *runner.Result {
	return runPostgresQuery(ctx, resolveCollectorRuntime(c.RepoRoot, c.runner, c.compose), containerID, query)
}

// Collect gathers database status information.
func (c DatabaseCollector) Collect(ctx context.Context) status.ComponentStatus {
	// Check docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Docker not available",
			Suggestions: []string{
				"Install Docker: https://docs.docker.com/get-docker/",
			},
		}
	}

	containerID, err := c.resolvePostgresContainer(ctx)
	if err != nil {
		message := "PostgreSQL container not running"
		if !strings.Contains(err.Error(), "not found") {
			message = "Failed to locate PostgreSQL container"
		}

		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: message,
			Details: map[string]any{
				"lookup_error": runner.SanitizeForDisplay(err.Error()),
			},
			Suggestions: []string{
				"Start services: make up",
				"Check container status: docker ps",
			},
		}
	}

	// Test database connectivity with a simple query
	testResult := c.runQuery(ctx, containerID, "SELECT 1;")
	if testResult.Error != nil {
		errMsg := "PostgreSQL not accepting connections"
		details := map[string]any{
			"exit_code": testResult.ExitCode,
		}
		if testResult.Stderr != "" {
			details["stderr"] = runner.SanitizeForDisplay(testResult.Stderr)
		}
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: errMsg,
			Details: details,
			Suggestions: []string{
				"Check PostgreSQL logs: docker compose logs postgres",
				"Restart services: make restart",
			},
		}
	}

	// Verify we got a result
	if !strings.Contains(testResult.Stdout, "1") {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "PostgreSQL responded unexpectedly",
			Details: map[string]any{
				"response": testResult.Stdout,
			},
		}
	}

	// Collect metrics
	details := make(map[string]any)

	// Get table counts
	tableQuery := `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('LiteLLM_VerificationToken', 'LiteLLM_UserTable', 'LiteLLM_BudgetTable', 'LiteLLM_SpendLogs');
	`
	tablesResult := c.runQuery(ctx, containerID, tableQuery)
	if tablesResult.Error == nil {
		tableCount, _ := strconv.Atoi(strings.TrimSpace(tablesResult.Stdout))
		details["expected_tables"] = tableCount
	} else if tablesResult.Stderr != "" {
		details["tables_error"] = runner.SanitizeForDisplay(tablesResult.Stderr)
	}

	// Get version
	versionResult := c.runQuery(ctx, containerID, "SELECT version();")
	if versionResult.Error == nil {
		version := strings.TrimSpace(versionResult.Stdout)
		if idx := strings.Index(version, "PostgreSQL"); idx != -1 {
			end := idx + 20
			if end > len(version) {
				end = len(version)
			}
			version = strings.TrimSpace(version[idx:end])
		}
		details["version"] = version
	} else if versionResult.Stderr != "" {
		details["version_error"] = runner.SanitizeForDisplay(versionResult.Stderr)
	}

	// Get database size
	sizeResult := c.runQuery(ctx, containerID, "SELECT pg_size_pretty(pg_database_size('litellm'));")
	if sizeResult.Error == nil {
		details["size"] = strings.TrimSpace(sizeResult.Stdout)
	}

	// Get active connections
	connResult := c.runQuery(ctx, containerID, "SELECT count(*) FROM pg_stat_activity WHERE datname = 'litellm';")
	if connResult.Error == nil {
		connections := strings.TrimSpace(connResult.Stdout)
		details["connections"] = connections
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: "Connected",
		Details: details,
	}
}

// dbQuery executes a SQL query against the PostgreSQL database.
func (c DatabaseCollector) dbQuery(ctx context.Context, query string) (string, error) {
	containerID, err := c.resolvePostgresContainer(ctx)
	if err != nil {
		return "", fmt.Errorf("postgres container lookup failed: %w", err)
	}

	result := c.runQuery(ctx, containerID, query)
	if result.Error != nil {
		errMsg := fmt.Sprintf("query failed (exit %d)", result.ExitCode)
		if result.Stderr != "" {
			errMsg = fmt.Sprintf("%s: %s", errMsg, runner.SanitizeForDisplay(result.Stderr))
		}
		return "", fmt.Errorf("%s", errMsg)
	}

	return strings.TrimSpace(result.Stdout), nil
}
