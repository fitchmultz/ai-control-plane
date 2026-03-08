// Package collectors provides shared helpers for PostgreSQL-backed status collectors.
//
// Purpose:
//
//	Standardize Docker/Compose-based PostgreSQL discovery and query execution
//	across status collectors.
//
// Responsibilities:
//   - Initialize default runner and compose helpers for collectors
//   - Resolve the active PostgreSQL container via Compose or docker ps fallback
//   - Execute bounded psql queries through docker exec
//
// Non-scope:
//   - Does not parse SQL query results
//   - Does not aggregate component health state
//
// Invariants/Assumptions:
//   - Collectors operate against the repo-local demo Compose project by default
//   - Compose lookup is preferred; docker ps is the compatibility fallback
//   - Query execution should always honor a deadline
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package collectors

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	acpdocker "github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

type containerIDResolver interface {
	ContainerID(ctx context.Context, service string) (string, error)
}

const (
	containerLookupTimeout = 5 * time.Second
	queryTimeout           = 10 * time.Second
)

type postgresRuntime struct {
	runner     runner.Runner
	compose    containerIDResolver
	projectDir string
}

func newCollectorRunner(repoRoot string) runner.Runner {
	return runner.NewDefaultRunner(filepath.Join(repoRoot, "demo"))
}

func newCollectorCompose(repoRoot string) containerIDResolver {
	compose, err := acpdocker.NewCompose(acpdocker.DefaultProjectDir(repoRoot))
	if err != nil {
		return nil
	}

	return compose
}

func resolveCollectorRuntime(repoRoot string, currentRunner runner.Runner, currentCompose containerIDResolver) postgresRuntime {
	if currentRunner == nil {
		currentRunner = newCollectorRunner(repoRoot)
	}
	if currentCompose == nil {
		currentCompose = newCollectorCompose(repoRoot)
	}

	return postgresRuntime{
		runner:     currentRunner,
		compose:    currentCompose,
		projectDir: filepath.Join(repoRoot, "demo"),
	}
}

func withCollectorTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, timeout)
}

func resolvePostgresContainer(ctx context.Context, runtime postgresRuntime) (string, error) {
	lookupCtx, cancel := withCollectorTimeout(ctx, containerLookupTimeout)
	defer cancel()

	if runtime.compose != nil {
		containerID, err := runtime.compose.ContainerID(lookupCtx, "postgres")
		if err == nil {
			containerID = strings.TrimSpace(containerID)
			if containerID != "" {
				return containerID, nil
			}
		}
	}

	args := []string{
		"ps",
		"--filter", "label=com.docker.compose.service=postgres",
	}
	if runtime.projectDir != "" {
		args = append(args, "--filter", fmt.Sprintf("label=com.docker.compose.project.working_dir=%s", runtime.projectDir))
	}
	args = append(args, "--format", "{{.ID}}")

	result := runtime.runner.Run(lookupCtx, "docker", args...)
	if result.Error != nil {
		return "", fmt.Errorf("postgres container lookup failed: %w", result.Error)
	}

	containerID := strings.TrimSpace(result.Stdout)
	if containerID == "" {
		return "", fmt.Errorf("postgres container not found")
	}
	if strings.Contains(containerID, "\n") {
		containerID = strings.Split(containerID, "\n")[0]
	}

	return containerID, nil
}

func runPostgresQuery(ctx context.Context, runtime postgresRuntime, containerID, query string) *runner.Result {
	queryCtx, cancel := withCollectorTimeout(ctx, queryTimeout)
	defer cancel()

	return runtime.runner.Run(queryCtx, "docker", "exec", containerID,
		"psql", "-U", "litellm", "-d", "litellm", "-t", "-c", query)
}
