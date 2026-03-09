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
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const smokeCommandTimeout = 30 * time.Second

type smokeInspector interface {
	Collect(ctx context.Context, opts status.Options) status.StatusReport
	Close() error
}

var newSmokeInspector = func(repoRoot string) smokeInspector {
	return runtimeinspect.NewInspector(repoRoot)
}

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
	if !prereq.CommandExists("docker") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Docker not found"))
		fmt.Fprintln(runCtx.Stderr, "Install Docker from https://docs.docker.com/get-docker/")
		return exitcodes.ACPExitPrereq
	}

	if runCtx.RepoRoot == "" {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Failed to detect repository root"))
		return exitcodes.ACPExitRuntime
	}

	inspector := newSmokeInspector(runCtx.RepoRoot)
	defer inspector.Close()

	smokeCtx, cancel := context.WithTimeout(ctx, smokeCommandTimeout)
	defer cancel()

	report := inspector.Collect(smokeCtx, status.Options{
		RepoRoot: runCtx.RepoRoot,
		Wide:     options.Verbose,
	})

	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Runtime Smoke Checks ==="))
	if err := report.WriteHuman(runCtx.Stdout, options.Verbose); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to render smoke output: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if errors.Is(smokeCtx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Smoke check timed out"))
		return exitcodes.ACPExitRuntime
	}
	if errors.Is(smokeCtx.Err(), context.Canceled) {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Smoke check canceled"))
		return exitcodes.ACPExitRuntime
	}

	readiness := runtimeinspect.EvaluateReadiness(report, runtimeinspect.DefaultReadinessComponents)
	if !readiness.Ready {
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
		fmt.Fprintln(runCtx.Stdout, out.Green("Runtime smoke checks passed"))
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		fmt.Fprintln(runCtx.Stderr, out.Fail("Runtime smoke checks failed"))
		return exitcodes.ACPExitDomain
	default:
		fmt.Fprintln(runCtx.Stderr, out.Fail("Runtime smoke returned unknown status"))
		return exitcodes.ACPExitRuntime
	}
}

func runSmokeTestCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"smoke"}, args, stdout, stderr)
}
