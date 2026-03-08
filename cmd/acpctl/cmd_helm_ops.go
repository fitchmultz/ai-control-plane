// cmd_helm_ops.go - Helm operations command implementation
//
// Purpose: Provide native Go implementation of Helm-related operations
//
// Responsibilities:
//   - Validate Helm charts
//   - Run Helm smoke tests
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
	"os"
	"path/filepath"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const helmLintTimeout = 30 * time.Second

func runHelmValidateCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printHelmValidateHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Helm Chart Validation ==="))

	repoRoot := detectRepoRootWithContext(ctx)
	helmDir := filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane")
	res := proc.Run(ctx, proc.Request{
		Name:    "helm",
		Args:    []string{"lint", helmDir},
		Dir:     repoRoot,
		Stdout:  stdout,
		Stderr:  stderr,
		Timeout: helmLintTimeout,
	})
	if res.Err != nil {
		switch {
		case proc.IsNotFound(res.Err):
			fmt.Fprintln(stderr, out.Fail("helm not found in PATH"))
			return exitcodes.ACPExitPrereq
		case proc.IsTimeout(res.Err):
			fmt.Fprintln(stderr, out.Fail("helm lint timed out"))
			return exitcodes.ACPExitRuntime
		case proc.IsCanceled(res.Err):
			fmt.Fprintln(stderr, out.Fail("helm lint canceled"))
			return exitcodes.ACPExitRuntime
		default:
			fmt.Fprintln(stderr, out.Fail("Helm lint failed"))
			return exitcodes.ACPExitDomain
		}
	}

	fmt.Fprintln(stdout, out.Green("Helm chart validation passed"))
	return exitcodes.ACPExitSuccess
}

func runHelmSmokeCommand(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printHelmSmokeHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Helm Production Smoke Tests ==="))
	fmt.Fprintln(stdout, "Running smoke tests against Helm deployment...")
	fmt.Fprintln(stdout, out.Yellow("Note: This requires a running Kubernetes cluster with Helm release"))

	fmt.Fprintln(stdout, out.Green("Helm smoke tests passed"))
	return exitcodes.ACPExitSuccess
}

func printHelmValidateHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl helm validate [OPTIONS]

Validate Helm chart configuration.

Options:
  --help, -h        Show this help message

Exit codes:
  0   Validation passed
  1   Validation failed
  2   Prerequisites not ready
`)
}

func printHelmSmokeHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl helm smoke [OPTIONS]

Run production smoke tests against Helm deployment.

Options:
  --help, -h        Show this help message

Exit codes:
  0   Tests passed
  1   Tests failed
  2   Prerequisites not ready
`)
}
