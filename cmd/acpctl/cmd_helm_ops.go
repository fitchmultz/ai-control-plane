// cmd_helm_ops.go - Helm validation and smoke gate command implementation.
//
// Purpose:
//   - Provide truthful operator-facing Helm validation and smoke gates.
//
// Responsibilities:
//   - Define the typed Helm command tree.
//   - Validate tracked Helm surfaces with the typed validation package.
//   - Run `helm lint` through the canonical subprocess wrapper.
//
// Scope:
//   - File-local Helm command execution and gate output only.
//
// Usage:
//   - Invoked via `acpctl helm validate` and `acpctl helm smoke`.
//
// Invariants/Assumptions:
//   - Helm smoke must not claim success without executing real validation.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

const helmLintTimeout = 30 * time.Second

type helmGateConfig struct {
	title          string
	successMessage string
}

var runHelmLint = func(ctx context.Context, repoRoot string, stdout *os.File, stderr *os.File) proc.Result {
	helmDir := filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane")
	return proc.Run(ctx, proc.Request{
		Name:    "helm",
		Args:    []string{"lint", helmDir},
		Dir:     repoRoot,
		Stdout:  stdout,
		Stderr:  stderr,
		Timeout: helmLintTimeout,
	})
}

func helmCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "helm",
		Summary:     "Helm chart validation and smoke gates",
		Description: "Helm chart validation and smoke gates.",
		Examples: []string{
			"acpctl helm validate",
			"acpctl helm smoke",
		},
		Children: []*commandSpec{
			helmGateCommandSpec(
				"validate",
				"Validate Helm deployment surfaces",
				"Validate tracked Helm deployment surfaces and run helm lint.",
				helmGateConfig{
					title:          "=== Helm Validation ===",
					successMessage: "Helm validation passed",
				},
				nil,
			),
			helmGateCommandSpec(
				"smoke",
				"Run truthful Helm smoke validation",
				"Run truthful Helm smoke validation for repository-managed deployment surfaces.",
				helmGateConfig{
					title:          "=== Helm Smoke Checks ===",
					successMessage: "Helm smoke checks passed",
				},
				[]commandHelpSection{
					{
						Title: "Notes",
						Lines: []string{
							"This validates repository-managed Helm artifacts only.",
							"It does not probe a live cluster or silently ignore missing context.",
						},
					},
				},
			),
		},
	}
}

func helmGateCommandSpec(name string, summary string, description string, config helmGateConfig, sections []commandHelpSection) *commandSpec {
	return &commandSpec{
		Name:        name,
		Summary:     summary,
		Description: description,
		Examples:    []string{"acpctl helm " + name},
		Sections:    sections,
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindStaticOptions(config),
			NativeRun:  runHelmGateCommand,
		},
	}
}

func runHelmGateCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	return runHelmGate(ctx, runCtx, raw.(helmGateConfig))
}

func runHelmValidateCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"helm", "validate"}, args, stdout, stderr)
}

func runHelmSmokeCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"helm", "smoke"}, args, stdout, stderr)
}

func runHelmGate(ctx context.Context, runCtx commandRunContext, config helmGateConfig) int {
	out := output.New()
	logger := workflowLogger(runCtx, "helm_gate", "title", config.title)
	workflowStart(logger)
	stdout := runCtx.Stdout
	stderr := runCtx.Stderr
	repoRoot := runCtx.RepoRoot
	if repoRoot == "" {
		workflowFailure(logger, fmt.Errorf("repository root not detected"))
		fmt.Fprintln(stderr, out.Fail("Failed to detect repository root"))
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, out.Bold(config.title))

	issues, err := validation.ValidateHelmSurfaces(repoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(stderr, out.Fail("Helm validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) > 0 {
		workflowWarn(logger, "status", "surface_validation_failed", "issues", len(issues))
		fmt.Fprintln(stderr, out.Fail("Helm deployment surface validation failed"))
		for _, issue := range issues {
			fmt.Fprintf(stderr, "  - %s\n", issue)
		}
		return exitcodes.ACPExitDomain
	}

	result := runHelmLint(ctx, repoRoot, stdout, stderr)
	if result.Err != nil {
		workflowFailure(logger, result.Err)
		switch {
		case proc.IsNotFound(result.Err):
			fmt.Fprintln(stderr, out.Fail("helm not found in PATH"))
		case proc.IsTimeout(result.Err):
			fmt.Fprintln(stderr, out.Fail("helm lint timed out"))
		case proc.IsCanceled(result.Err):
			fmt.Fprintln(stderr, out.Fail("helm lint canceled"))
		case proc.IsStart(result.Err):
			fmt.Fprintf(stderr, out.Fail("helm lint could not start: %v\n"), result.Err)
		default:
			fmt.Fprintln(stderr, out.Fail("helm lint failed"))
			if proc.IsExit(result.Err) {
				return exitcodes.ACPExitDomain
			}
		}
		return proc.ACPExitCode(result.Err)
	}

	fmt.Fprintln(stdout, out.Green(config.successMessage))
	workflowComplete(logger, "status", "passed")
	return exitcodes.ACPExitSuccess
}
