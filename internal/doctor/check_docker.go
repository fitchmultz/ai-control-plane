// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement the Docker availability check through the canonical subprocess
//	layer so operator-facing failures and timeouts are shaped consistently.
//
// Responsibilities:
//   - Verify the docker binary exists.
//   - Verify the Docker daemon is accessible.
//
// Non-scope:
//   - Does not manage Docker service lifecycle.
//
// Invariants/Assumptions:
//   - All subprocess execution flows through internal/proc.
//
// Scope:
//   - Docker prerequisite diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package doctor

import (
	"context"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

var runDockerInfo = func(ctx context.Context) proc.Result {
	return proc.Run(ctx, proc.Request{Name: "docker", Args: []string{"info"}})
}

type dockerAvailableCheck struct{}

func (c dockerAvailableCheck) ID() string { return "docker_available" }

func (c dockerAvailableCheck) Run(ctx context.Context, opts Options) CheckResult {
	result := runDockerInfo(ctx)
	if result.Err != nil {
		message := "Docker daemon not accessible"
		suggestions := []string{
			"Ensure Docker service is running",
			"Add your user to the docker group if required",
		}
		if proc.IsNotFound(result.Err) {
			message = "Docker not found in PATH"
			suggestions = []string{
				"Install Docker: https://docs.docker.com/get-docker/",
				"Verify installation: docker --version",
			}
		} else if strings.Contains(strings.ToLower(result.Stderr), "permission denied") {
			message = "Docker daemon requires permissions"
		}
		return withCheckDetails(
			newCheckResult(c.ID(), "Docker Available", status.HealthLevelUnhealthy, SeverityPrereq, message),
			status.ComponentDetails{
				Error: strings.TrimSpace(result.Stderr),
			},
			suggestions...,
		)
	}

	return newCheckResult(c.ID(), "Docker Available", status.HealthLevelHealthy, SeverityDomain, "Docker is available and daemon is accessible")
}

func (c dockerAvailableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return noopFix(ctx, opts)
}
