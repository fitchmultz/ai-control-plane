// Package docker provides Docker and Docker Compose utilities.
//
// Purpose:
//
//	Detect Docker Compose availability and provide a consistent interface
//	for Docker Compose operations.
//
// Responsibilities:
//   - Detect Docker Compose command (V2 "docker compose" vs V1 "docker-compose")
//   - Provide Docker Compose execution helpers
//   - Check Docker daemon availability
//
// Non-scope:
//   - Does not manage containers directly (use docker CLI or SDK)
//   - Does not handle image building
//
// Invariants/Assumptions:
//   - Prefers Docker Compose V2 (docker compose) over V1 (docker-compose)
//   - All commands require Docker to be running
package docker

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

// Compose provides Docker Compose operations
type Compose struct {
	cmd        string
	argsPrefix []string
	projectDir string
}

// DetectCompose detects the available Docker Compose command
func DetectCompose() (*Compose, error) {
	// Prefer V2: docker compose
	if cmd := exec.Command("docker", "compose", "version"); cmd.Run() == nil {
		return &Compose{
			cmd:        "docker",
			argsPrefix: []string{"compose"},
		}, nil
	}

	// Fall back to V1: docker-compose
	if path, err := exec.LookPath("docker-compose"); err == nil {
		return &Compose{
			cmd: path,
		}, nil
	}

	return nil, fmt.Errorf("neither 'docker compose' (V2) nor 'docker-compose' (V1) is available")
}

// NewCompose creates a new Compose instance with the given project directory
func NewCompose(projectDir string) (*Compose, error) {
	compose, err := DetectCompose()
	if err != nil {
		return nil, err
	}
	compose.projectDir = projectDir
	return compose, nil
}

// Command returns the base command string for display purposes
func (c *Compose) Command() string {
	if len(c.argsPrefix) > 0 {
		return fmt.Sprintf("%s %s", c.cmd, strings.Join(c.argsPrefix, " "))
	}
	return c.cmd
}

// PS lists running containers
func (c *Compose) PS(ctx context.Context) (string, error) {
	args := c.buildArgs("ps")
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w", err)
	}
	return string(output), nil
}

// PSFilter lists containers filtered by service name
func (c *Compose) PSFilter(ctx context.Context, service string) (string, error) {
	args := c.buildArgs("ps", service)
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker compose ps %s failed: %w", service, err)
	}
	return string(output), nil
}

// ContainerID returns the container ID for a service
func (c *Compose) ContainerID(ctx context.Context, service string) (string, error) {
	args := c.buildArgs("ps", "-q", service)
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get container ID for %s: %w", service, err)
	}
	id := strings.TrimSpace(string(output))
	if id == "" {
		return "", fmt.Errorf("container not found for service: %s", service)
	}
	return id, nil
}

// IsServiceRunning checks if a service container is running
func (c *Compose) IsServiceRunning(ctx context.Context, service string) bool {
	output, err := c.PSFilter(ctx, service)
	if err != nil {
		return false
	}
	return strings.Contains(output, "Up")
}

// Up starts services
func (c *Compose) Up(ctx context.Context, detach bool, profiles ...string) error {
	args := c.buildArgs("up")
	if detach {
		args = append(args, "-d")
	}
	for _, profile := range profiles {
		args = append(args, "--profile", profile)
	}
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Down stops services
func (c *Compose) Down(ctx context.Context) error {
	args := c.buildArgs("down")
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	return cmd.Run()
}

// Logs shows service logs
func (c *Compose) Logs(ctx context.Context, service string, follow bool) error {
	args := c.buildArgs("logs")
	if follow {
		args = append(args, "-f")
	}
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Exec executes a command in a container
func (c *Compose) Exec(ctx context.Context, service string, command ...string) (string, error) {
	args := c.buildArgs(append([]string{"exec", service}, command...)...)
	cmd := exec.CommandContext(ctx, c.cmd, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker compose exec failed: %w", err)
	}
	return string(output), nil
}

// ExecInContainer executes a command in a specific container by ID
func ExecInContainer(ctx context.Context, containerID string, command ...string) (string, error) {
	args := append([]string{"exec", containerID}, command...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker exec failed: %w", err)
	}
	return string(output), nil
}

// ExecInContainerWithStdin executes a command in a specific container by ID with streamed stdin.
func ExecInContainerWithStdin(ctx context.Context, containerID string, stdin io.Reader, command ...string) (string, error) {
	args := append([]string{"exec", "-i", containerID}, command...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = stdin
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker exec failed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// buildArgs builds the full argument list including project directory
func (c *Compose) buildArgs(args ...string) []string {
	result := make([]string, 0, len(c.argsPrefix)+2+len(args))
	result = append(result, c.argsPrefix...)
	if c.projectDir != "" {
		result = append(result, "-f", filepath.Join(c.projectDir, "docker-compose.yml"))
		result = append(result, "--project-directory", c.projectDir)
	}
	result = append(result, args...)
	return result
}

// DefaultProjectDir returns the default compose project directory (demo/)
func DefaultProjectDir(repoRoot string) string {
	return filepath.Join(repoRoot, "demo")
}

// IsDockerRunning checks if Docker daemon is accessible
func IsDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}
