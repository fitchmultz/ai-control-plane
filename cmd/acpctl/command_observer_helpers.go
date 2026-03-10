// command_observer_helpers.go - Shared observer command helpers.
//
// Purpose:
//   - Normalize runtime-observer command setup, collection, and output
//     rendering across status, doctor, health, smoke, and CI wait flows.
//
// Responsibilities:
//   - Provide a shared runtime inspector contract for command-layer stubbing.
//   - Centralize repository-root validation when opening inspectors.
//   - Render JSON and human command output with one stable error contract.
//
// Scope:
//   - Shared observer helpers for read-only command execution paths.
//
// Usage:
//   - Used by native commands that collect runtime status or emit structured
//     observer reports.
//
// Invariants/Assumptions:
//   - Helpers preserve the existing stdout/stderr split.
//   - Runtime observer commands remain read-only.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type runtimeStatusInspector interface {
	Collect(context.Context, status.Options) status.StatusReport
	Close() error
}

var newRuntimeStatusInspector = func(repoRoot string) runtimeStatusInspector {
	return runtimeinspect.NewInspector(repoRoot)
}

type runtimeReportCommandConfig struct {
	RequireDocker   bool
	Wide            bool
	Timeout         time.Duration
	TimeoutMessage  string
	CanceledMessage string
}

func openRuntimeStatusInspector(runCtx commandRunContext, logger *slog.Logger, out *output.Output, factory func(string) runtimeStatusInspector) (runtimeStatusInspector, int) {
	if ok, code := requireRuntimeRepoRoot(runCtx, logger, out); !ok {
		return nil, code
	}
	return factory(runCtx.RepoRoot), exitcodes.ACPExitSuccess
}

func collectRuntimeStatusReport(ctx context.Context, inspector runtimeStatusInspector, repoRoot string, wide bool, timeout time.Duration) (status.StatusReport, context.Context, context.CancelFunc) {
	collectCtx, cancel := context.WithTimeout(ctx, timeout)
	report := inspector.Collect(collectCtx, status.Options{
		RepoRoot: repoRoot,
		Wide:     wide,
	})
	return report, collectCtx, cancel
}

func collectRuntimeReportOrExit(
	ctx context.Context,
	runCtx commandRunContext,
	logger *slog.Logger,
	out *output.Output,
	inspector runtimeStatusInspector,
	config runtimeReportCommandConfig,
) (status.StatusReport, int, bool) {
	report, collectCtx, cancel := collectRuntimeStatusReport(ctx, inspector, runCtx.RepoRoot, config.Wide, config.Timeout)
	defer cancel()
	if handled, code := runtimeCommandContextResult(collectCtx, logger, out, config.TimeoutMessage, config.CanceledMessage, runCtx.Stderr); handled {
		return status.StatusReport{}, code, false
	}
	return report, exitcodes.ACPExitSuccess, true
}

func runRuntimeReportCommand(
	ctx context.Context,
	runCtx commandRunContext,
	logger *slog.Logger,
	factory func(string) runtimeStatusInspector,
	config runtimeReportCommandConfig,
	handle func(*output.Output, status.StatusReport) int,
) int {
	out := output.New()
	if config.RequireDocker {
		if ok, code := requireDockerRuntime(runCtx, logger, out); !ok {
			return code
		}
	}
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, factory)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	report, code, ok := collectRuntimeReportOrExit(ctx, runCtx, logger, out, inspector, config)
	if !ok {
		return code
	}
	return handle(out, report)
}

func writeStructuredCommandOutput(stdout *os.File, stderr *os.File, jsonOutput bool, writeJSON func(io.Writer) error, writeHuman func(io.Writer) error) int {
	if jsonOutput {
		if err := writeJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write JSON output: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	}
	if err := writeHuman(stdout); err != nil {
		fmt.Fprintf(stderr, "Error: failed to write output: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func writeRuntimeReportOutput(runCtx commandRunContext, logger *slog.Logger, out *output.Output, title string, report status.StatusReport, jsonOutput bool, wide bool) int {
	if jsonOutput {
		return writeStructuredCommandOutput(runCtx.Stdout, runCtx.Stderr, true, report.WriteJSON, nil)
	}
	return writeRuntimeReport(runCtx.Stdout, runCtx.Stderr, logger, out, title, report, wide)
}

func exitCodeForHealthLevel(level status.HealthLevel) int {
	switch level {
	case status.HealthLevelHealthy:
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		return exitcodes.ACPExitDomain
	default:
		return exitcodes.ACPExitRuntime
	}
}

func writeWatchTermination(stdout *os.File, jsonOutput bool) {
	if jsonOutput {
		return
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Watch mode stopped.")
}

func commandContextCanceled(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}
