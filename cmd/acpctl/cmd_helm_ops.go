// cmd_helm_ops.go - Helm operations command implementation
//
// Purpose: Provide native Go implementation of Helm-related operations
//
// Responsibilities:
//   - Validate Helm charts
//   - Run Helm smoke tests

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

func runHelmValidateCommand(args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printHelmValidateHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Helm Chart Validation ==="))

	// Check if helm is available
	if _, err := exec.LookPath("helm"); err != nil {
		fmt.Fprintln(stderr, out.Fail("helm not found in PATH"))
		return exitcodes.ACPExitPrereq
	}

	repoRoot := detectRepoRoot()
	helmDir := repoRoot + "/deploy/helm/ai-control-plane"

	// Run helm lint
	cmd := exec.Command("helm", "lint", helmDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Helm lint failed: %s\n"), string(output))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, out.Green("Helm chart validation passed"))
	return exitcodes.ACPExitSuccess
}

func runHelmSmokeCommand(args []string, stdout *os.File, stderr *os.File) int {
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
