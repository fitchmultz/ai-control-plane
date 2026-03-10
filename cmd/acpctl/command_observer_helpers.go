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
