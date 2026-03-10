// command_runtime_helpers.go - Shared runtime command helpers.
//
// Purpose:
//   - Keep runtime-facing commands aligned on one prerequisite, rendering, and
//     terminal error-handling contract.
//
// Responsibilities:
//   - Validate Docker and repository-root prerequisites for runtime commands.
//   - Render collected runtime reports consistently.
//   - Map timeout/cancel/report-write failures to stable exit behavior.
//
// Scope:
//   - Shared helpers for health, smoke, and adjacent runtime-inspection
//     commands.
//
// Usage:
//   - Used by native commands that collect `status.StatusReport` from the
//     runtime inspection stack.
//
// Invariants/Assumptions:
//   - Runtime commands remain read-only.
//   - Prerequisite failures stay user-actionable on stderr.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func requireDockerRuntime(runCtx commandRunContext, logger *slog.Logger, out *output.Output) (bool, int) {
	if prereq.CommandExists("docker") {
		return true, exitcodes.ACPExitSuccess
	}
	workflowFailure(logger, fmt.Errorf("docker not found"))
	fmt.Fprintln(runCtx.Stderr, out.Fail("Docker not found"))
	fmt.Fprintln(runCtx.Stderr, "Install Docker from https://docs.docker.com/get-docker/")
	return false, exitcodes.ACPExitPrereq
}

func requireRuntimeRepoRoot(runCtx commandRunContext, logger *slog.Logger, out *output.Output) (bool, int) {
	if runCtx.RepoRoot != "" {
		return true, exitcodes.ACPExitSuccess
	}
	workflowFailure(logger, fmt.Errorf("repository root not detected"))
	fmt.Fprintln(runCtx.Stderr, out.Fail("Failed to detect repository root"))
	return false, exitcodes.ACPExitRuntime
}

func writeRuntimeReport(stdout *os.File, stderr *os.File, logger *slog.Logger, out *output.Output, title string, report status.StatusReport, wide bool) int {
	if title != "" {
		printCommandSection(stdout, out, title)
	}
	if err := report.WriteHuman(stdout, wide); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(stderr, out.Fail("Failed to render runtime output: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func runtimeCommandContextResult(ctx context.Context, logger *slog.Logger, out *output.Output, timeoutMessage string, canceledMessage string, stderr *os.File) (bool, int) {
	if contextError := ctx.Err(); contextError != nil {
		workflowFailure(logger, contextError, "status", contextError.Error())
		switch {
		case errors.Is(contextError, context.DeadlineExceeded):
			fmt.Fprintln(stderr, out.Fail(timeoutMessage))
		case errors.Is(contextError, context.Canceled):
			fmt.Fprintln(stderr, out.Fail(canceledMessage))
		default:
			fmt.Fprintf(stderr, out.Fail("%v\n"), contextError)
		}
		return true, exitcodes.ACPExitRuntime
	}
	return false, exitcodes.ACPExitSuccess
}
