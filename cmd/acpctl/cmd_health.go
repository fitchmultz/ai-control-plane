// cmd_health.go - Health check command implementation
//
// Purpose: Provide native Go implementation of health checks.
//
// Responsibilities:
//   - Define the typed health command surface.
//   - Check Docker container status and ACP runtime health.
//   - Render human-readable health output.
//
// Non-scope:
//   - Does not start services.
//   - Does not fix issues.
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
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const healthCommandTimeout = 30 * time.Second

type healthOptions struct {
	Verbose bool
}

func healthCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "health",
		Summary:     "Run service health checks",
		Description: "Run health checks for AI Control Plane services.",
		Examples: []string{
			"acpctl health",
			"acpctl health --verbose",
		},
		Options: []commandOptionSpec{
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Sections: []commandHelpSection{
			{
				Title: "Environment",
				Lines: []string{
					"GATEWAY_HOST",
					"LITELLM_PORT",
					"LITELLM_MASTER_KEY",
					"ACP_DATABASE_MODE",
				},
			},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
				return healthOptions{Verbose: input.Bool("verbose")}, nil
			},
			NativeRun: runHealth,
		},
	}
}

func runHealth(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(healthOptions)
	out := output.New()

	if !prereq.CommandExists("docker") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Docker not found"))
		fmt.Fprintln(runCtx.Stderr, "Install Docker from https://docs.docker.com/get-docker/")
		return exitcodes.ACPExitPrereq
	}

	if runCtx.RepoRoot == "" {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Failed to detect repository root"))
		return exitcodes.ACPExitRuntime
	}

	inspector := runtimeinspect.NewInspector(runCtx.RepoRoot)
	defer inspector.Close()

	ctx, cancel := context.WithTimeout(ctx, healthCommandTimeout)
	defer cancel()
	report := inspector.Collect(ctx, status.Options{RepoRoot: runCtx.RepoRoot, Wide: opts.Verbose})
	if err := report.WriteHuman(runCtx.Stdout, opts.Verbose); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to render health output: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Health check timed out"))
		return exitcodes.ACPExitRuntime
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Health check canceled"))
		return exitcodes.ACPExitRuntime
	}

	switch report.Overall {
	case status.HealthLevelHealthy:
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		return exitcodes.ACPExitDomain
	default:
		return exitcodes.ACPExitRuntime
	}
}

func runHealthCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"health"}, args, stdout, stderr)
}
