// cmd_pilot_closeout_bundle.go - Pilot closeout bundle command implementation.
//
// Purpose:
//
//	Provide a typed CLI surface for assembling and verifying pilot closeout
//	evidence bundles.
//
// Responsibilities:
//   - Parse closeout-bundle command arguments
//   - Build local pilot closeout bundles from source documents and readiness evidence
//   - Verify generated closeout bundle structure
//
// Scope:
//   - Covers local bundle assembly and verification only
//
// Usage:
//   - `acpctl deploy pilot-closeout-bundle build`
//   - `acpctl deploy pilot-closeout-bundle verify`
//
// Invariants/Assumptions:
//   - Bundles remain local-only under `demo/logs/pilot-closeout`
//   - Source pilot documents are authored outside the generated bundle
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/release"
)

func runPilotCloseoutBundleCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		printPilotCloseoutBundleHelp(stdout)
		if len(args) == 0 {
			return exitcodes.ACPExitUsage
		}
		return exitcodes.ACPExitSuccess
	}

	switch args[0] {
	case "build":
		return runPilotCloseoutBundleBuild(ctx, args[1:], stdout, stderr)
	case "verify":
		return runPilotCloseoutBundleVerify(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: unknown pilot-closeout-bundle command: %s\n", args[0])
		printPilotCloseoutBundleHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func runPilotCloseoutBundleBuild(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	options := release.PilotCloseoutOptions{
		RepoRoot:   repoRoot,
		OutputRoot: filepath.Join(repoRoot, "demo", "logs", "pilot-closeout"),
	}

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--output-dir":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --output-dir")
				return exitcodes.ACPExitUsage
			}
			options.OutputRoot = resolveReadinessPath(repoRoot, args[index])
		case "--customer":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --customer")
				return exitcodes.ACPExitUsage
			}
			options.Customer = args[index]
		case "--pilot-name":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --pilot-name")
				return exitcodes.ACPExitUsage
			}
			options.PilotName = args[index]
		case "--decision":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --decision")
				return exitcodes.ACPExitUsage
			}
			options.Decision = args[index]
		case "--charter":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --charter")
				return exitcodes.ACPExitUsage
			}
			options.CharterPath = resolveReadinessPath(repoRoot, args[index])
		case "--acceptance-memo":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --acceptance-memo")
				return exitcodes.ACPExitUsage
			}
			options.AcceptanceMemoPath = resolveReadinessPath(repoRoot, args[index])
		case "--validation-checklist":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --validation-checklist")
				return exitcodes.ACPExitUsage
			}
			options.ValidationChecklist = resolveReadinessPath(repoRoot, args[index])
		case "--operator-checklist":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --operator-checklist")
				return exitcodes.ACPExitUsage
			}
			options.OperatorChecklist = resolveReadinessPath(repoRoot, args[index])
		case "--readiness-run-dir":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --readiness-run-dir")
				return exitcodes.ACPExitUsage
			}
			options.ReadinessRunDir = resolveReadinessPath(repoRoot, args[index])
		case "--help", "-h":
			printPilotCloseoutBundleBuildHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[index])
			return exitcodes.ACPExitUsage
		}
	}

	fmt.Fprint(stdout, out.Bold("Building pilot closeout bundle")+"\n")
	summary, err := release.BuildPilotCloseoutBundle(options)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Pilot closeout bundle complete"))+"\n")
	fmt.Fprintf(stdout, "  Run directory: %s\n", summary.RunDirectory)
	fmt.Fprintf(stdout, "  Summary: %s\n", filepath.Join(summary.RunDirectory, "closeout-summary.md"))
	fmt.Fprintf(stdout, "  Inventory: %s\n", filepath.Join(summary.RunDirectory, "bundle-inventory.txt"))
	return exitcodes.ACPExitSuccess
}

func runPilotCloseoutBundleVerify(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	runDir := ""

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--run-dir":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --run-dir")
				return exitcodes.ACPExitUsage
			}
			runDir = resolveReadinessPath(repoRoot, args[index])
		case "--help", "-h":
			printPilotCloseoutBundleVerifyHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[index])
			return exitcodes.ACPExitUsage
		}
	}

	if strings.TrimSpace(runDir) == "" {
		data, err := os.ReadFile(filepath.Join(repoRoot, "demo", "logs", "pilot-closeout", "latest-run.txt"))
		if err != nil {
			fmt.Fprintln(stderr, "Error: no pilot closeout bundle available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = strings.TrimSpace(string(data))
	}

	fmt.Fprint(stdout, out.Bold("Verifying pilot closeout bundle")+"\n")
	fmt.Fprintf(stdout, "  Run directory: %s\n", runDir)
	summary, err := release.NewPilotCloseoutVerifier().VerifyPilotCloseoutBundle(runDir)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Pilot closeout bundle verified"))+"\n")
	fmt.Fprintf(stdout, "  Customer: %s\n", summary.Customer)
	fmt.Fprintf(stdout, "  Pilot: %s\n", summary.PilotName)
	fmt.Fprintf(stdout, "  Decision: %s\n", summary.Decision)
	return exitcodes.ACPExitSuccess
}

func printPilotCloseoutBundleHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy pilot-closeout-bundle <command> [OPTIONS]

Commands:
  build    Assemble a local pilot closeout bundle
  verify   Verify a generated pilot closeout bundle

Examples:
  acpctl deploy pilot-closeout-bundle build --customer "Falcon Insurance Group" --pilot-name "Claims Governance Pilot" --charter docs/examples/falcon-insurance-group/PILOT_CHARTER.md --acceptance-memo docs/examples/falcon-insurance-group/PILOT_ACCEPTANCE_MEMO.md --validation-checklist docs/examples/falcon-insurance-group/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md
  acpctl deploy pilot-closeout-bundle verify --run-dir demo/logs/pilot-closeout/pilot-closeout-20260305T170000Z
`)
}

func printPilotCloseoutBundleBuildHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy pilot-closeout-bundle build [OPTIONS]

Options:
  --output-dir DIR             Output root for bundle runs (default: demo/logs/pilot-closeout)
  --customer NAME              Customer name
  --pilot-name NAME            Pilot name
  --decision VALUE             Decision label (default: PENDING_REVIEW)
  --charter PATH               Pilot charter source document
  --acceptance-memo PATH       Pilot acceptance memo source document
  --validation-checklist PATH  Customer validation checklist source document
  --operator-checklist PATH    Optional operator handoff checklist source document
  --readiness-run-dir DIR      Specific readiness evidence run to include (default: latest)
  --help                       Show this help message
`)
}

func printPilotCloseoutBundleVerifyHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy pilot-closeout-bundle verify [OPTIONS]

Options:
  --run-dir DIR   Specific pilot closeout bundle directory to verify (default: latest generated run)
  --help          Show this help message
`)
}
