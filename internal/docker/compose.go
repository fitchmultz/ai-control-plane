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
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package docker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const (
	detectComposeTimeout  = 5 * time.Second
	composeCommandTimeout = 30 * time.Second
	dockerInfoTimeout     = 5 * time.Second
)

// Compose provides Docker Compose operations
type Compose struct {
	cmd         string
	argsPrefix  []string
	projectDir  string
	projectName string
	files       []string
}

// ComposeOptions configures project-scoped compose execution details.
type ComposeOptions struct {
	ProjectName string
	Files       []string
}

// DetectCompose detects the available Docker Compose command
func DetectCompose() (*Compose, error) {
	// Prefer V2: docker compose
	if res := proc.Run(context.Background(), proc.Request{
		Name:    "docker",
		Args:    []string{"compose", "version"},
		Timeout: detectComposeTimeout,
	}); res.Err == nil {
		return &Compose{
			cmd:        "docker",
			argsPrefix: []string{"compose"},
		}, nil
	}

	// Fall back to V1: docker-compose
	if res := proc.Run(context.Background(), proc.Request{
		Name:    "docker-compose",
		Args:    []string{"version"},
		Timeout: detectComposeTimeout,
	}); res.Err == nil {
		return &Compose{
			cmd: "docker-compose",
		}, nil
	}

	return nil, fmt.Errorf("neither 'docker compose' (V2) nor 'docker-compose' (V1) is available")
}

// NewCompose creates a new Compose instance with the given project directory
func NewCompose(projectDir string) (*Compose, error) {
	return NewComposeWithOptions(projectDir, ComposeOptions{})
}

// NewACPCompose creates a compose instance using ACP slot-aware project defaults.
func NewACPCompose(repoRoot string, files []string) (*Compose, error) {
	loader := config.NewLoader().WithRepoRoot(repoRoot)
	tooling := loader.Tooling()
	projectName := strings.TrimSpace(tooling.ComposeProject)
	if projectName == "" {
		slot := strings.TrimSpace(tooling.Slot)
		if slot == "" {
			slot = "active"
		}
		projectName = "ai-control-plane-" + slot
	}
	resolvedFiles := append([]string(nil), files...)
	if len(resolvedFiles) == 0 && strings.EqualFold(strings.TrimSpace(tooling.Slot), "ci-runtime") {
		resolvedFiles = []string{"docker-compose.offline.yml"}
	}
	return NewComposeWithOptions(DefaultProjectDir(repoRoot), ComposeOptions{
		ProjectName: projectName,
		Files:       resolvedFiles,
	})
}

// NewComposeWithOptions creates a new Compose instance with explicit project/file options.
func NewComposeWithOptions(projectDir string, opts ComposeOptions) (*Compose, error) {
	compose, err := DetectCompose()
	if err != nil {
		return nil, err
	}
	compose.projectDir = projectDir
	compose.projectName = strings.TrimSpace(opts.ProjectName)
	compose.files = append([]string(nil), opts.Files...)
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
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("docker compose ps failed: %w", res.Err)
	}
	return res.Stdout, nil
}

// PSFilter lists containers filtered by service name
func (c *Compose) PSFilter(ctx context.Context, service string) (string, error) {
	args := c.buildArgs("ps", service)
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("docker compose ps %s failed: %w", service, res.Err)
	}
	return res.Stdout, nil
}

// ContainerID returns the container ID for a service
func (c *Compose) ContainerID(ctx context.Context, service string) (string, error) {
	args := c.buildArgs("ps", "-q", service)
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("failed to get container ID for %s: %w", service, res.Err)
	}
	id := strings.TrimSpace(res.Stdout)
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
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	return res.Err
}

// Down stops services
func (c *Compose) Down(ctx context.Context) error {
	args := c.buildArgs("down")
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	return res.Err
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
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	return res.Err
}

// Exec executes a command in a container
func (c *Compose) Exec(ctx context.Context, service string, command ...string) (string, error) {
	args := c.buildArgs(append([]string{"exec", service}, command...)...)
	res := proc.Run(ctx, proc.Request{
		Name:    c.cmd,
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("docker compose exec failed: %w", res.Err)
	}
	return res.Stdout, nil
}

// ExecInContainer executes a command in a specific container by ID
func ExecInContainer(ctx context.Context, containerID string, command ...string) (string, error) {
	args := append([]string{"exec", containerID}, command...)
	res := proc.Run(ctx, proc.Request{
		Name:    "docker",
		Args:    args,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("docker exec failed: %w", res.Err)
	}
	return res.Stdout, nil
}

// ExecInContainerWithStdin executes a command in a specific container by ID with streamed stdin.
func ExecInContainerWithStdin(ctx context.Context, containerID string, stdin io.Reader, command ...string) (string, error) {
	args := append([]string{"exec", "-i", containerID}, command...)
	res := proc.Run(ctx, proc.Request{
		Name:    "docker",
		Args:    args,
		Stdin:   stdin,
		Timeout: composeCommandTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("docker exec failed: %s: %w", strings.TrimSpace(res.Stderr), res.Err)
	}
	return res.Stdout + res.Stderr, nil
}

// buildArgs builds the full argument list including project directory
func (c *Compose) buildArgs(args ...string) []string {
	result := make([]string, 0, len(c.argsPrefix)+2+(len(c.files)*2)+len(args))
	result = append(result, c.argsPrefix...)
	if c.projectName != "" {
		result = append(result, "--project-name", c.projectName)
	}
	if c.projectDir != "" {
		if len(c.files) == 0 {
			result = append(result, "-f", filepath.Join(c.projectDir, "docker-compose.yml"))
		} else {
			for _, file := range c.files {
				result = append(result, "-f", resolveComposeFile(c.projectDir, file))
			}
		}
		result = append(result, "--project-directory", c.projectDir)
	}
	result = append(result, args...)
	return result
}

func resolveComposeFile(projectDir string, file string) string {
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(projectDir, file)
}

// DefaultProjectDir returns the default compose project directory (demo/)
func DefaultProjectDir(repoRoot string) string {
	return repopath.DemoPath(repoRoot)
}

// IsDockerRunning checks if Docker daemon is accessible
func IsDockerRunning() bool {
	res := proc.Run(context.Background(), proc.Request{
		Name:    "docker",
		Args:    []string{"info"},
		Timeout: dockerInfoTimeout,
	})
	return res.Err == nil
}
