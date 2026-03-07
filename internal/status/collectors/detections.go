// detections.go implements the detection findings status collector.
//
// Purpose:
//
//	Summarize recent LiteLLM detection findings and prerequisite detection
//	rules state so ACP health reports reflect audit-signal readiness.
//
// Responsibilities:
//   - Verify detection rules configuration exists locally.
//   - Resolve the PostgreSQL container and query recent spend-log findings.
//   - Translate recent finding counts into operator-facing health summaries.
//
// Scope:
//   - Covers detection configuration presence and recent finding aggregation only.
//
// Usage:
//   - Construct `NewDetectionsCollector(repoRoot)` and call `Collect(ctx)`.
//
// Invariants/Assumptions:
//   - Detection findings are derived from LiteLLM spend-log data.
//   - Missing configuration is reported before runtime queries are attempted.
package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

// DetectionsCollector summarizes recent detection findings.
type DetectionsCollector struct {
	RepoRoot string
	runner   runner.Runner
	compose  containerIDResolver
}

// NewDetectionsCollector creates a new detections collector.
func NewDetectionsCollector(repoRoot string) *DetectionsCollector {
	return &DetectionsCollector{
		RepoRoot: repoRoot,
		runner:   newCollectorRunner(repoRoot),
		compose:  newCollectorCompose(repoRoot),
	}
}

// SetRunner sets a custom runner (for testing).
func (c *DetectionsCollector) SetRunner(r runner.Runner) {
	c.runner = r
}

// SetContainerResolver sets a custom container resolver (for testing).
func (c *DetectionsCollector) SetContainerResolver(resolver containerIDResolver) {
	c.compose = resolver
}

// Name returns the collector's domain name.
func (c DetectionsCollector) Name() string {
	return "detections"
}

// Collect gathers detection status information.
func (c DetectionsCollector) Collect(ctx context.Context) status.ComponentStatus {
	// Check if detection rules config exists
	configPath := filepath.Join(c.RepoRoot, "demo", "config", "detection_rules.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Detection rules config not found",
			Suggestions: []string{
				"Verify installation: detection_rules.yaml should exist",
			},
		}
	}

	// Check docker is available
	if _, err := exec.LookPath("docker"); err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Docker not available",
		}
	}

	runtime := resolveCollectorRuntime(c.RepoRoot, c.runner, c.compose)
	containerID, err := resolvePostgresContainer(ctx, runtime)
	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "PostgreSQL unavailable",
			Details: map[string]any{
				"lookup_error": runner.SanitizeForDisplay(err.Error()),
			},
		}
	}

	// Check if SpendLogs table exists
	tableQuery := `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`
	tableResult := runPostgresQuery(ctx, runtime, containerID, tableQuery)
	if tableResult.Error != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnknown,
			Message: "Could not check for SpendLogs table",
			Details: map[string]any{
				"exit_code": tableResult.ExitCode,
				"stderr":    runner.SanitizeForDisplay(tableResult.Stderr),
			},
		}
	}

	tableCount, _ := strconv.Atoi(strings.TrimSpace(tableResult.Stdout))
	if tableCount == 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelHealthy,
			Message: "No audit log data yet",
			Details: map[string]any{
				"high_severity":   0,
				"medium_severity": 0,
				"low_severity":    0,
			},
			Suggestions: []string{
				"Logs will appear after API requests are made",
			},
		}
	}

	// Check recent high-severity findings (last 24 hours)
	highQuery := `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`
	highResult := runPostgresQuery(ctx, runtime, containerID, highQuery)

	highCount := 0
	if highResult.Error == nil {
		highCount, _ = strconv.Atoi(strings.TrimSpace(highResult.Stdout))
	}

	// Check for unusual model usage patterns (potential policy violation)
	modelQuery := `
		SELECT COUNT(DISTINCT model) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`
	modelResult := runPostgresQuery(ctx, runtime, containerID, modelQuery)

	modelCount := 0
	if modelResult.Error == nil {
		modelCount, _ = strconv.Atoi(strings.TrimSpace(modelResult.Stdout))
	}

	// Get total entries in last 24h
	totalQuery := `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`
	totalResult := runPostgresQuery(ctx, runtime, containerID, totalQuery)

	totalEntries := 0
	if totalResult.Error == nil {
		totalEntries, _ = strconv.Atoi(strings.TrimSpace(totalResult.Stdout))
	}

	details := map[string]any{
		"high_severity_findings": highCount,
		"unique_models_24h":      modelCount,
		"total_entries_24h":      totalEntries,
	}

	// Determine status based on findings
	if highCount > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("%d high-severity findings in last 24h", highCount),
			Details: details,
			Suggestions: []string{
				"Run detections: acpctl validate detections",
				"Review audit logs: acpctl db status",
				"Check for anomalous spend patterns",
			},
		}
	}

	// Check for medium severity (elevated spend > $5)
	mediumQuery := `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 5.0 AND spend <= 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`
	mediumResult := runPostgresQuery(ctx, runtime, containerID, mediumQuery)

	mediumCount := 0
	if mediumResult.Error == nil {
		mediumCount, _ = strconv.Atoi(strings.TrimSpace(mediumResult.Stdout))
		details["medium_severity_findings"] = mediumCount
	} else if mediumResult.Stderr != "" {
		details["medium_query_error"] = runner.SanitizeForDisplay(mediumResult.Stderr)
	}

	if mediumCount > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("No high-severity, %d medium in last 24h", mediumCount),
			Details: details,
			Suggestions: []string{
				"Review elevated spend patterns",
				"Run full detection scan: acpctl validate detections",
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: "No recent findings",
		Details: details,
	}
}
