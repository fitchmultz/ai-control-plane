// cmd_ci_wait.go - CI wait for health command
//
// Purpose: Wait for Docker services to be healthy before proceeding
//
// Responsibilities:
//   - Wait for Compose services to report healthy status
//   - Verify LiteLLM /health endpoint responds successfully
//   - Enforce configurable timeout and interval polling
//
// Non-scope:
//   - Does NOT start or create containers
//   - Does NOT run full health suite
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

type ciWaitCompose interface {
	PS(ctx context.Context) (string, error)
}

type ciWaitGateway interface {
	Health(ctx context.Context) (bool, int, error)
	HasMasterKey() bool
}

var newCIWaitCompose = func(repoRoot string) (ciWaitCompose, error) {
	tooling := config.NewLoader().Tooling()
	projectName := strings.TrimSpace(tooling.ComposeProject)
	if projectName == "" {
		slot := strings.TrimSpace(tooling.Slot)
		if slot == "" {
			slot = "active"
		}
		projectName = "ai-control-plane-" + slot
	}
	return docker.NewComposeWithOptions(docker.DefaultProjectDir(repoRoot), docker.ComposeOptions{
		ProjectName: projectName,
		Files:       []string{"docker-compose.offline.yml"},
	})
}

var newCIWaitGateway = func() ciWaitGateway {
	return gateway.NewClient()
}

type ciWaitOptions struct {
	Timeout  time.Duration
	Verbose  bool
	Interval time.Duration
}

func ciWaitCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "wait",
		Summary:     "Wait for services to become healthy",
		Description: "Wait for Docker services to be healthy before proceeding.",
		Examples: []string{
			"acpctl ci wait",
			"acpctl ci wait --timeout 60",
			"acpctl ci wait --verbose",
		},
		Options: []commandOptionSpec{
			{Name: "timeout", ValueName: "SECONDS", Summary: "Maximum time to wait", Type: optionValueInt, DefaultText: "120"},
			{Name: "verbose", Short: "v", Summary: "Enable verbose output", Type: optionValueBool},
		},
		Sections: []commandHelpSection{
			{
				Title: "Environment variables",
				Lines: []string{
					"LITELLM_MASTER_KEY  Master key for authorized gateway checks (required)",
				},
			},
		},
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindCIWaitOptions,
			NativeRun:  executeCIWaitCommand,
		},
	}
}

func bindCIWaitOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	timeoutSeconds := 120
	if input.String("timeout") != "" {
		value, err := input.Int("timeout")
		if err != nil || value <= 0 {
			return nil, fmt.Errorf("invalid --timeout value: %q (must be a positive integer)", input.String("timeout"))
		}
		timeoutSeconds = value
	}
	return ciWaitOptions{
		Timeout:  time.Duration(timeoutSeconds) * time.Second,
		Interval: 5 * time.Second,
		Verbose:  input.Bool("verbose"),
	}, nil
}

func executeCIWaitCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(ciWaitOptions)

	out := output.New()

	// Prerequisite checks
	if !prereq.CommandExists("docker") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("docker not found"))
		return exitcodes.ACPExitPrereq
	}

	compose, err := newCIWaitCompose(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	gw := newCIWaitGateway()
	if !gw.HasMasterKey() {
		fmt.Fprintln(runCtx.Stderr, out.Fail("LITELLM_MASTER_KEY is required for authorized gateway health checks"))
		fmt.Fprintln(runCtx.Stderr, "Set LITELLM_MASTER_KEY in your environment or demo/.env")
		return exitcodes.ACPExitUsage
	}

	fmt.Fprintln(runCtx.Stdout, out.Bold("Waiting for services to become healthy..."))
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stdout, "Timeout: %s, Check interval: %s\n", opts.Timeout, opts.Interval)
	}

	waitCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	postgresHealthy := false
	litellmHealthy := false
	litellmAPIReady := false

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	probe := func() {
		ps, _ := compose.PS(waitCtx)

		if strings.Contains(ps, "postgres") && strings.Contains(ps, "healthy") {
			if !postgresHealthy {
				fmt.Fprintln(runCtx.Stdout, out.Pass("PostgreSQL is healthy"))
				postgresHealthy = true
			}
		} else if opts.Verbose {
			fmt.Fprintln(runCtx.Stdout, "  PostgreSQL not healthy yet")
		}

		if strings.Contains(ps, "litellm") && strings.Contains(ps, "healthy") {
			if !litellmHealthy {
				fmt.Fprintln(runCtx.Stdout, out.Pass("LiteLLM is healthy"))
				litellmHealthy = true
			}
		} else if opts.Verbose {
			fmt.Fprintln(runCtx.Stdout, "  LiteLLM not healthy yet")
		}

		if !litellmAPIReady {
			healthy, _, err := gw.Health(waitCtx)
			if err == nil && healthy {
				fmt.Fprintln(runCtx.Stdout, out.Pass("LiteLLM API is responding (authorized HTTP 200)"))
				litellmAPIReady = true
			} else if opts.Verbose {
				fmt.Fprintln(runCtx.Stdout, "  LiteLLM API not responding yet")
			}
		}
	}

	for {
		probe()
		if postgresHealthy && litellmHealthy && litellmAPIReady {
			fmt.Fprintln(runCtx.Stdout)
			fmt.Fprintln(runCtx.Stdout, out.Green("All services are healthy and ready"))
			return exitcodes.ACPExitSuccess
		}

		select {
		case <-waitCtx.Done():
			fmt.Fprintln(runCtx.Stdout)
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				fmt.Fprintln(runCtx.Stderr, out.Fail("CI wait canceled"))
				return exitcodes.ACPExitRuntime
			}
			fmt.Fprintf(runCtx.Stdout, out.Fail("Timeout: Services did not become healthy within %s\n"), opts.Timeout)
			statusCtx, statusCancel := context.WithTimeout(ctx, 2*time.Second)
			defer statusCancel()
			if ps, err := compose.PS(statusCtx); err == nil && strings.TrimSpace(ps) != "" {
				fmt.Fprintln(runCtx.Stdout)
				fmt.Fprintln(runCtx.Stdout, "Current container status:")
				fmt.Fprintln(runCtx.Stdout, ps)
			}
			return exitcodes.ACPExitDomain
		case <-ticker.C:
			continue
		}
	}
}

func runCIWaitCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	spec := ciWaitCommandSpec()
	input, helpOnly, err := parseLeafInput(spec, args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if helpOnly {
		printCommandHelp(stdout, []*commandSpec{acpctlCommandSpec(), ciCommandSpec(), spec})
		return exitcodes.ACPExitSuccess
	}
	opts, err := bindCIWaitOptions(commandBindContext{RepoRoot: detectRepoRootWithContext(ctx)}, input)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	return executeCIWaitCommand(ctx, commandRunContext{
		RepoRoot: detectRepoRootWithContext(ctx),
		Stdout:   stdout,
		Stderr:   stderr,
	}, opts)
}
