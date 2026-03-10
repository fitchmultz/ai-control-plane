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
	"fmt"
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

const healthCommandTimeout = 30 * time.Second

type healthOptions struct {
	Verbose bool
}

type healthInspector = runtimeStatusInspector

var newHealthInspector = newRuntimeStatusInspector

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
	logger := workflowLogger(runCtx, "runtime_health", "verbose", opts.Verbose)
	workflowStart(logger)

	if ok, code := requireDockerRuntime(runCtx, logger, out); !ok {
		return code
	}
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, newHealthInspector)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	report, collectCtx, cancel := collectRuntimeStatusReport(ctx, inspector, runCtx.RepoRoot, opts.Verbose, healthCommandTimeout)
	defer cancel()
	logger.Info("workflow.report_collected", "overall", report.Overall)
	if code := writeRuntimeReport(runCtx.Stdout, runCtx.Stderr, logger, out, "", report, opts.Verbose); code != exitcodes.ACPExitSuccess {
		return code
	}
	if handled, code := runtimeCommandContextResult(collectCtx, logger, out, "Health check timed out", "Health check canceled", runCtx.Stderr); handled {
		return code
	}

	switch code := exitCodeForHealthLevel(report.Overall); code {
	case exitcodes.ACPExitSuccess:
		workflowComplete(logger, "status", "healthy")
		return code
	case exitcodes.ACPExitDomain:
		workflowWarn(logger, "status", string(report.Overall))
		return code
	default:
		workflowFailure(logger, fmt.Errorf("unknown health status: %s", string(report.Overall)))
		return code
	}
}

func runHealthCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"health"}, args, stdout, stderr)
}
