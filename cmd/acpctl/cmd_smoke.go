// cmd_smoke.go - Runtime smoke gate command implementation.
//
// Purpose:
//   - Run truthful runtime smoke validation for operator-facing production gates.
//
// Responsibilities:
//   - Define the typed smoke command surface.
//   - Reuse the canonical runtime inspection stack for smoke checks.
//   - Enforce gateway auth, model reachability, and database readiness.
//
// Scope:
//   - File-local smoke command execution and output.
//
// Usage:
//   - Invoked via `acpctl smoke` and make targets that delegate to it.
//
// Invariants/Assumptions:
//   - Smoke is a real gate and must not silently pass on warnings or bad inputs.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const smokeCommandTimeout = 30 * time.Second

type smokeInspector = runtimeStatusInspector

var newSmokeInspector = newRuntimeStatusInspector

type smokeOptions struct {
	Verbose bool
}

func smokeCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "smoke",
		Summary:     "Run truthful runtime smoke checks",
		Description: "Run truthful runtime smoke checks against the active ACP deployment.",
		Examples: []string{
			"acpctl smoke",
			"acpctl smoke --verbose",
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
				return smokeOptions{Verbose: input.Bool("verbose")}, nil
			},
			NativeRun: runSmokeTest,
		},
	}
}

func runSmokeTest(ctx context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(smokeOptions)
	out := output.New()
	logger := workflowLogger(runCtx, "runtime_smoke", "verbose", options.Verbose)
	workflowStart(logger)
	if ok, code := requireDockerRuntime(runCtx, logger, out); !ok {
		return code
	}
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, newSmokeInspector)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	report, smokeCtx, cancel := collectRuntimeStatusReport(ctx, inspector, runCtx.RepoRoot, options.Verbose, smokeCommandTimeout)
	defer cancel()
	logger.Info("workflow.report_collected", "overall", report.Overall)
	if code := writeRuntimeReport(runCtx.Stdout, runCtx.Stderr, logger, out, "=== Runtime Smoke Checks ===", report, options.Verbose); code != exitcodes.ACPExitSuccess {
		return code
	}
	if handled, code := runtimeCommandContextResult(smokeCtx, logger, out, "Smoke check timed out", "Smoke check canceled", runCtx.Stderr); handled {
		return code
	}

	readiness := runtimeinspect.EvaluateReadiness(report, runtimeinspect.DefaultReadinessComponents)
	if !readiness.Ready {
		workflowWarn(logger, "status", "not_ready", "missing_components", readiness.Missing)
		fmt.Fprintln(runCtx.Stderr, out.Fail("Runtime smoke failed: required components are not ready"))
		for _, name := range readiness.Missing {
			component, ok := readiness.Pending[name]
			switch {
			case ok && strings.TrimSpace(component.Message) != "":
				fmt.Fprintf(runCtx.Stderr, "  - %s: %s\n", name, component.Message)
			default:
				fmt.Fprintf(runCtx.Stderr, "  - %s: not ready\n", name)
			}
		}
		return exitcodes.ACPExitDomain
	}

	switch report.Overall {
	case status.HealthLevelHealthy:
		workflowComplete(logger, "status", "healthy")
		fmt.Fprintln(runCtx.Stdout, out.Green("Runtime smoke checks passed"))
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		workflowWarn(logger, "status", string(report.Overall))
		fmt.Fprintln(runCtx.Stderr, out.Fail("Runtime smoke checks failed"))
		return exitcodes.ACPExitDomain
	default:
		workflowFailure(logger, fmt.Errorf("unknown runtime smoke status: %s", string(report.Overall)))
		fmt.Fprintln(runCtx.Stderr, out.Fail("Runtime smoke returned unknown status"))
		return exitcodes.ACPExitRuntime
	}
}

func runSmokeTestCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"smoke"}, args, stdout, stderr)
}
