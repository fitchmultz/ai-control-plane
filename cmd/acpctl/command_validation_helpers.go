// command_validation_helpers.go - Shared validation command helpers.
//
// Purpose:
//   - Keep validation and security command adapters aligned on one rendering and
//     issue-reporting contract.
//
// Responsibilities:
//   - Load shared validation contracts for detection/SIEM validators.
//   - Render consistent success and issue output for string-based validators.
//   - Centralize tracked-file enumeration for security validators.
//
// Scope:
//   - Command-layer helpers only; validation and security policy stays in
//     internal packages.
//
// Usage:
//   - Used by `cmd_validate.go` and `cmd_security.go`.
//
// Invariants/Assumptions:
//   - Validators remain side-effect free.
//   - Findings remain deterministic and machine-scannable.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/security"
)

type issueValidationConfig struct {
	Title           string
	SuccessMessage  string
	FailureMessage  string
	RuntimeErrorMsg string
	ColorSuccess    bool
}

func runIssueValidation(runCtx commandRunContext, logger *slog.Logger, config issueValidationConfig, validate func() ([]string, error)) int {
	out := output.New()
	if config.Title != "" {
		printCommandSection(runCtx.Stdout, out, config.Title)
	}

	issues, err := validate()
	if err != nil {
		if logger != nil {
			workflowFailure(logger, err)
		}
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, config.RuntimeErrorMsg)
	}
	if len(issues) > 0 {
		if logger != nil {
			workflowWarn(logger, "issues", len(issues))
		}
		return failValidation(runCtx.Stderr, out, issues, config.FailureMessage)
	}
	if logger != nil {
		workflowComplete(logger, "issues", 0)
	}
	if config.ColorSuccess {
		fmt.Fprintln(runCtx.Stdout, out.Green(config.SuccessMessage))
	} else {
		fmt.Fprintln(runCtx.Stdout, config.SuccessMessage)
	}
	return exitcodes.ACPExitSuccess
}

func withValidationContracts(runCtx commandRunContext, title string, verbose bool, loadFailureMessage string, fn func(*output.Output, validationContracts) int) int {
	out := output.New()
	if title != "" {
		printCommandSection(runCtx.Stdout, out, title)
	}
	artifacts, err := loadValidationContracts(runCtx.RepoRoot)
	if err != nil {
		return failCommand(runCtx.Stderr, out, mapValidationLoadExitCode(err), err, loadFailureMessage)
	}
	if verbose {
		printValidationContractPaths(runCtx.Stdout, artifacts)
	}
	return fn(out, artifacts)
}

func withTrackedFiles(ctx context.Context, runCtx commandRunContext, logger *slog.Logger, out *output.Output, failureMessage string, fn func([]string) int) int {
	trackedFiles, err := security.ListTrackedFiles(ctx, runCtx.RepoRoot)
	if err != nil {
		if logger != nil {
			workflowFailure(logger, err)
		}
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitPrereq, err, failureMessage)
	}
	return fn(trackedFiles)
}

func printIssueList(out *os.File, banner string, issues []string) {
	if banner != "" {
		fmt.Fprintln(out, banner)
	}
	for _, issue := range issues {
		fmt.Fprintln(out, issue)
	}
}
