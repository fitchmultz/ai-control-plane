// cmd_helm_ops.go - Helm validation and smoke gate command implementation.
//
// Purpose:
//   - Provide truthful operator-facing Helm validation and smoke gates.
//
// Responsibilities:
//   - Validate tracked Helm surfaces with the typed validation package.
//   - Run `helm lint` through the canonical subprocess wrapper.
//   - Return honest prerequisite, domain, and runtime exit codes.
//
// Scope:
//   - File-local Helm command parsing and gate execution only.
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

func runHelmValidateCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if err := ensureHelpOnlyArgs(args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		printHelmValidateHelp(stderr)
		return exitcodes.ACPExitUsage
	}
	if wantsHelp(args) {
		printHelmValidateHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	return runHelmGate(ctx, stdout, stderr, helmGateConfig{
		title:          "=== Helm Validation ===",
		successMessage: "Helm validation passed",
	})
}

func runHelmSmokeCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if err := ensureHelpOnlyArgs(args); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		printHelmSmokeHelp(stderr)
		return exitcodes.ACPExitUsage
	}
	if wantsHelp(args) {
		printHelmSmokeHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	return runHelmGate(ctx, stdout, stderr, helmGateConfig{
		title:          "=== Helm Smoke Checks ===",
		successMessage: "Helm smoke checks passed",
	})
}

func runHelmGate(ctx context.Context, stdout *os.File, stderr *os.File, config helmGateConfig) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	if repoRoot == "" {
		fmt.Fprintln(stderr, out.Fail("Failed to detect repository root"))
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, out.Bold(config.title))

	issues, err := validation.ValidateHelmSurfaces(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Helm validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) > 0 {
		fmt.Fprintln(stderr, out.Fail("Helm deployment surface validation failed"))
		for _, issue := range issues {
			fmt.Fprintf(stderr, "  - %s\n", issue)
		}
		return exitcodes.ACPExitDomain
	}

	result := runHelmLint(ctx, repoRoot, stdout, stderr)
	if result.Err != nil {
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
	return exitcodes.ACPExitSuccess
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func ensureHelpOnlyArgs(args []string) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			continue
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}
	return nil
}

func printHelmValidateHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl helm validate [OPTIONS]

Validate tracked Helm deployment surfaces and run helm lint.

Checks:
  - Helm chart, values, schema, and template structure
  - Helm production/demo values contract enforcement
  - helm lint against deploy/helm/ai-control-plane

Options:
  --help, -h        Show this help message

Examples:
  acpctl helm validate

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printHelmSmokeHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl helm smoke [OPTIONS]

Run truthful Helm smoke validation for repository-managed deployment surfaces.

Checks:
  - Helm chart, values, schema, and template structure
  - Helm production/demo values contract enforcement
  - helm lint against deploy/helm/ai-control-plane

Notes:
  - This command validates repository-managed Helm artifacts only.
  - It does not probe a live cluster or silently ignore missing context.

Options:
  --help, -h        Show this help message

Examples:
  acpctl helm smoke

Exit codes:
  0   Tests passed
  1   Tests failed
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}
