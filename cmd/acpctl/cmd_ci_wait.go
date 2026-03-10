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
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type ciWaitInspector = runtimeStatusInspector

var newCIWaitInspector = newRuntimeStatusInspector

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
			NativeBind: bindParsed(bindCIWaitOptions),
			NativeRun:  executeCIWaitCommand,
		},
	}
}

func bindCIWaitOptions(input parsedCommandInput) (ciWaitOptions, error) {
	timeoutSeconds, err := input.IntDefault("timeout", 120)
	if err != nil || timeoutSeconds <= 0 {
		return ciWaitOptions{}, fmt.Errorf("invalid --timeout value: %q (must be a positive integer)", input.String("timeout"))
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
	logger := workflowLogger(runCtx, "ci_wait", "timeout", opts.Timeout.String(), "interval", opts.Interval.String(), "verbose", opts.Verbose)
	workflowStart(logger)

	if ok, code := requireDockerRuntime(runCtx, logger, out); !ok {
		return code
	}
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, newCIWaitInspector)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	fmt.Fprintln(runCtx.Stdout, out.Bold("Waiting for services to become healthy..."))
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stdout, "Timeout: %s, Check interval: %s\n", opts.Timeout, opts.Interval)
	}

	waitCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	reportedReady := make(map[string]struct{})
	probe := func() status.StatusReport {
		report := inspector.Collect(waitCtx, status.Options{RepoRoot: runCtx.RepoRoot, Wide: opts.Verbose})
		readiness := runtimeinspect.EvaluateReadiness(report, runtimeinspect.DefaultReadinessComponents)
		for _, name := range runtimeinspect.DefaultReadinessComponents {
			if _, alreadyReported := reportedReady[name]; alreadyReported {
				continue
			}
			component, ok := report.Components[name]
			if ok && component.Level == status.HealthLevelHealthy {
				logger.Info("workflow.component_ready", "component", name)
				fmt.Fprintln(runCtx.Stdout, out.Pass(ciWaitReadyMessage(name)))
				reportedReady[name] = struct{}{}
				continue
			}
			if opts.Verbose {
				fmt.Fprintf(runCtx.Stdout, "  %s not ready yet\n", ciWaitPendingMessage(name, readiness.Pending[name]))
			}
		}
		return report
	}

	for {
		report := probe()
		if runtimeinspect.EvaluateReadiness(report, runtimeinspect.DefaultReadinessComponents).Ready {
			fmt.Fprintln(runCtx.Stdout)
			fmt.Fprintln(runCtx.Stdout, out.Green("All services are healthy and ready"))
			workflowComplete(logger, "status", "ready")
			return exitcodes.ACPExitSuccess
		}

		select {
		case <-waitCtx.Done():
			fmt.Fprintln(runCtx.Stdout)
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				workflowFailure(logger, ctx.Err(), "status", "canceled")
				fmt.Fprintln(runCtx.Stderr, out.Fail("CI wait canceled"))
				return exitcodes.ACPExitRuntime
			}
			workflowWarn(logger, "status", "timeout")
			fmt.Fprintf(runCtx.Stdout, out.Fail("Timeout: Services did not become healthy within %s\n"), opts.Timeout)
			finalReport, _, statusCancel := collectRuntimeStatusReport(ctx, inspector, runCtx.RepoRoot, true, 5*time.Second)
			defer statusCancel()
			if len(finalReport.Components) > 0 {
				fmt.Fprintln(runCtx.Stdout)
				fmt.Fprintln(runCtx.Stdout, "Current runtime status:")
				if err := finalReport.WriteHuman(runCtx.Stdout, true); err != nil {
					workflowFailure(logger, err)
					fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to render runtime status: %v\n"), err)
					return exitcodes.ACPExitRuntime
				}
			}
			return exitcodes.ACPExitDomain
		case <-ticker.C:
			continue
		}
	}
}

func ciWaitReadyMessage(component string) string {
	switch component {
	case "database":
		return "PostgreSQL is healthy"
	case "gateway":
		return "LiteLLM API is responding (authorized HTTP 200)"
	default:
		return component + " is ready"
	}
}

func ciWaitPendingMessage(component string, pending status.ComponentStatus) string {
	if pending.Message != "" {
		return fmt.Sprintf("%s (%s)", component, pending.Message)
	}
	return component
}

func runCIWaitCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"ci", "wait"}, args, stdout, stderr)
}
