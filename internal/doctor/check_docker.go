// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement the Docker availability check as a focused module.
//
// Responsibilities:
//   - Verify the docker binary exists.
//   - Verify the Docker daemon is accessible.
//
// Scope:
//   - Docker prerequisite diagnostics only.
package doctor

import (
	"context"
	"os/exec"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type dockerAvailableCheck struct{}

func (c dockerAvailableCheck) ID() string { return "docker_available" }

func (c dockerAvailableCheck) Run(ctx context.Context, opts Options) CheckResult {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Docker Available",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "Docker not found in PATH",
			Suggestions: []string{
				"Install Docker: https://docs.docker.com/get-docker/",
				"Verify installation: docker --version",
			},
		}
	}

	cmd := exec.CommandContext(ctx, dockerPath, "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := "Docker daemon not accessible"
		suggestions := []string{
			"Ensure Docker service is running: sudo systemctl start docker",
			"Add user to docker group: sudo usermod -aG docker $USER",
			"Re-login or run: newgrp docker",
		}
		if strings.Contains(string(output), "permission denied") {
			msg = "Docker daemon requires permissions"
			suggestions = []string{
				"Add user to docker group: sudo usermod -aG docker $USER",
				"Re-login or run: newgrp docker",
			}
		}
		return CheckResult{
			ID:          c.ID(),
			Name:        "Docker Available",
			Level:       status.HealthLevelUnhealthy,
			Severity:    SeverityPrereq,
			Message:     msg,
			Suggestions: suggestions,
			Details: map[string]any{
				"docker_path": dockerPath,
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Docker Available",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "Docker is available and daemon is accessible",
		Details: map[string]any{
			"docker_path": dockerPath,
		},
	}
}

func (c dockerAvailableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
