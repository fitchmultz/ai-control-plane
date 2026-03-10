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
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const healthCommandTimeout = 30 * time.Second

type healthOptions struct {
	Verbose bool
}

var newHealthInspector = newRuntimeStatusInspector

func healthCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
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
		Bind: bindParsedValue(func(input parsedCommandInput) healthOptions {
			return healthOptions{Verbose: input.Bool("verbose")}
		}),
		Run: runHealth,
	})
}

func runHealth(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(healthOptions)
	logger := workflowLogger(runCtx, "runtime_health", "verbose", opts.Verbose)
	workflowStart(logger)

	return runRuntimeReportCommand(ctx, runCtx, logger, newHealthInspector, runtimeReportCommandConfig{
		RequireDocker:   true,
		Wide:            opts.Verbose,
		Timeout:         healthCommandTimeout,
		TimeoutMessage:  "Health check timed out",
		CanceledMessage: "Health check canceled",
	}, func(out *output.Output, report status.StatusReport) int {
		logger.Info("workflow.report_collected", "overall", report.Overall)
		if code := writeRuntimeReportOutput(runCtx, logger, out, "", report, false, opts.Verbose); code != exitcodes.ACPExitSuccess {
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
	})
}
