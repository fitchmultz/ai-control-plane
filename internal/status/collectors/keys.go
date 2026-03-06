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

// KeysCollector checks virtual key counts and status.
type KeysCollector struct {
	RepoRoot string
	runner   runner.Runner
	compose  containerIDResolver
}

// NewKeysCollector creates a new keys collector.
func NewKeysCollector(repoRoot string) *KeysCollector {
	return &KeysCollector{
		RepoRoot: repoRoot,
		runner:   newCollectorRunner(repoRoot),
		compose:  newCollectorCompose(repoRoot),
	}
}

// SetRunner sets a custom runner (for testing).
func (c *KeysCollector) SetRunner(r runner.Runner) {
	c.runner = r
}

// SetContainerResolver sets a custom container resolver (for testing).
func (c *KeysCollector) SetContainerResolver(resolver containerIDResolver) {
	c.compose = resolver
}

// Name returns the collector's domain name.
func (c KeysCollector) Name() string {
	return "keys"
}

// Collect gathers virtual key status information.
func (c KeysCollector) Collect(ctx context.Context) status.ComponentStatus {
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

	// Get total key count
	countQuery := `SELECT COUNT(*) FROM "LiteLLM_VerificationToken";`
	countResult := runPostgresQuery(ctx, runtime, containerID, countQuery)
	if countResult.Error != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Could not query key count",
			Details: map[string]any{
				"exit_code": countResult.ExitCode,
				"stderr":    runner.SanitizeForDisplay(countResult.Stderr),
			},
			Suggestions: []string{
				"Table may not exist yet - LiteLLM creates tables on first use",
			},
		}
	}

	countStr := strings.TrimSpace(countResult.Stdout)
	totalCount, err := strconv.Atoi(countStr)
	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Failed to parse key count",
		}
	}

	details := map[string]any{
		"total_keys": totalCount,
	}

	// If no keys, warn that configuration is incomplete
	if totalCount == 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "No virtual keys configured",
			Details: details,
			Suggestions: []string{
				"Generate a key: acpctl key gen my-key --budget 10.00",
				"Or: make key-gen ALIAS=my-key BUDGET=10.00",
			},
		}
	}

	// Get active (non-expired) key count
	activeQuery := `
		SELECT COUNT(*) FROM "LiteLLM_VerificationToken"
		WHERE expires IS NULL OR expires > NOW();
	`
	activeResult := runPostgresQuery(ctx, runtime, containerID, activeQuery)
	if activeResult.Error == nil {
		activeCount, _ := strconv.Atoi(strings.TrimSpace(activeResult.Stdout))
		details["active_keys"] = activeCount
		details["expired_keys"] = totalCount - activeCount

		if activeCount < totalCount {
			return status.ComponentStatus{
				Name:    c.Name(),
				Level:   status.HealthLevelWarning,
				Message: fmt.Sprintf("%d keys, %d expired", totalCount, totalCount-activeCount),
				Details: details,
				Suggestions: []string{
					"Review expired keys: acpctl db status",
					"Revoke unused keys: acpctl key revoke <alias>",
				},
			}
		}
	} else if activeResult.Stderr != "" {
		details["active_query_error"] = runner.SanitizeForDisplay(activeResult.Stderr)
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: fmt.Sprintf("%d active keys", totalCount),
		Details: details,
	}
}
