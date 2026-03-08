// cmd_readiness_evidence.go - Readiness evidence command implementation.
//
// Purpose:
//
//	Generate and verify timestamped readiness evidence packs for enterprise review.
//
// Responsibilities:
//   - Parse readiness-evidence command arguments.
//   - Execute internal/readiness workflows.
//   - Print operator-facing summaries with stable exit codes.
//
// Scope:
//   - Covers local proof-pack generation and verification only.
//
// Usage:
//   - `acpctl deploy readiness-evidence run`
//   - `acpctl deploy readiness-evidence verify --run-dir <dir>`
//
// Invariants/Assumptions:
//   - Generated evidence remains local-only under demo/logs/evidence.
//   - Exit codes follow the repository-wide contract.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
)

func runReadinessEvidenceCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		printReadinessEvidenceHelp(stdout)
		if len(args) == 0 {
			return exitcodes.ACPExitUsage
		}
		return exitcodes.ACPExitSuccess
	}

	switch args[0] {
	case "run":
		return runReadinessEvidenceRun(ctx, args[1:], stdout, stderr)
	case "verify":
		return runReadinessEvidenceVerify(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: unknown readiness-evidence command: %s\n", args[0])
		printReadinessEvidenceHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func runReadinessEvidenceRun(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	makeBin := config.NewLoader().Tooling().MakeBinary
	options := readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    filepath.Join(repoRoot, "demo", "logs", "evidence"),
		MakeBin:       makeBin,
		BundleVersion: bundle.GetDefaultVersion(repoRoot),
	}

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--output-dir":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --output-dir")
				return exitcodes.ACPExitUsage
			}
			options.OutputRoot = resolveReadinessPath(repoRoot, args[index+1])
			index++
		case "--bundle-version":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --bundle-version")
				return exitcodes.ACPExitUsage
			}
			options.BundleVersion = args[index+1]
			index++
		case "--include-production":
			options.IncludeProduction = true
		case "--secrets-env-file":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --secrets-env-file")
				return exitcodes.ACPExitUsage
			}
			options.SecretsEnvFile = resolveReadinessPath(repoRoot, args[index+1])
			index++
		case "--verbose":
			options.Verbose = true
		case "--help", "-h":
			printReadinessEvidenceRunHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[index])
			return exitcodes.ACPExitUsage
		}
	}

	fmt.Fprint(stdout, out.Bold("Generating readiness evidence")+"\n")
	summary, err := readiness.RunContext(ctx, options, stdout, stderr)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Readiness evidence complete"))+"\n")
	fmt.Fprintf(stdout, "  Run directory: %s\n", summary.RunDirectory)
	fmt.Fprintf(stdout, "  Summary: %s\n", filepath.Join(summary.RunDirectory, readiness.SummaryMarkdownName))
	fmt.Fprintf(stdout, "  Tracker: %s\n", filepath.Join(summary.RunDirectory, readiness.TrackerMarkdownName))
	fmt.Fprintf(stdout, "  Decision: %s\n", filepath.Join(summary.RunDirectory, readiness.DecisionMarkdownName))
	fmt.Fprintf(stdout, "  Overall status: %s\n", summary.OverallStatus)
	if summary.OverallStatus != "PASS" {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runReadinessEvidenceVerify(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)
	runDir := ""

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--run-dir":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --run-dir")
				return exitcodes.ACPExitUsage
			}
			runDir = resolveReadinessPath(repoRoot, args[index+1])
			index++
		case "--help", "-h":
			printReadinessEvidenceVerifyHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[index])
			return exitcodes.ACPExitUsage
		}
	}

	if runDir == "" {
		resolvedRunDir, err := readiness.ResolveLatestRun(filepath.Join(repoRoot, "demo", "logs", "evidence"))
		if err != nil {
			fmt.Fprintln(stderr, "Error: no readiness run available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	fmt.Fprint(stdout, out.Bold("Verifying readiness evidence")+"\n")
	fmt.Fprintf(stdout, "  Run directory: %s\n", runDir)

	summary, err := readiness.NewVerifier().VerifyRun(runDir)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Readiness evidence verified"))+"\n")
	fmt.Fprintf(stdout, "  Run ID: %s\n", summary.RunID)
	fmt.Fprintf(stdout, "  Overall status: %s\n", summary.OverallStatus)
	fmt.Fprintf(stdout, "  Inventory: %s\n", filepath.Join(runDir, readiness.InventoryFileName))
	return exitcodes.ACPExitSuccess
}

func resolveReadinessPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func printReadinessEvidenceHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy readiness-evidence <command> [OPTIONS]

Commands:
  run      Generate a new readiness evidence run
  verify   Verify a generated readiness evidence directory

Examples:
  acpctl deploy readiness-evidence run
  acpctl deploy readiness-evidence run --include-production --secrets-env-file /etc/ai-control-plane/secrets.env
  acpctl deploy readiness-evidence verify --run-dir demo/logs/evidence/readiness-20260305T230000Z

Exit codes:
  0   Success
  1   Domain failure (one or more required gates failed, or verification found drift)
  2   Prerequisite failure
  3   Runtime/internal error
  64  Usage error
`)
}

func printReadinessEvidenceRunHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy readiness-evidence run [OPTIONS]

Options:
  --output-dir DIR          Output root for readiness runs (default: demo/logs/evidence)
  --bundle-version VALUE    Bundle version for release-bundle commands (default: git short SHA)
  --include-production      Attempt the production-like gate if secrets are available
  --secrets-env-file PATH   Secrets file for production gate (default: /etc/ai-control-plane/secrets.env)
  --verbose                 Reserved for future verbose rendering
  --help                    Show this help message
`)
}

func printReadinessEvidenceVerifyHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy readiness-evidence verify [OPTIONS]

Options:
  --run-dir DIR   Specific readiness run directory to verify (default: latest generated run)
  --help          Show this help message
`)
}
