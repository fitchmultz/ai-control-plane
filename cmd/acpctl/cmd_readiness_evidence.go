// cmd_readiness_evidence.go - Readiness evidence command implementation.
//
// Purpose:
//   - Generate and verify timestamped readiness evidence packs for enterprise review.
//
// Responsibilities:
//   - Define the typed readiness-evidence command tree.
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
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
)

func readinessEvidenceCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "readiness-evidence",
		Summary:     "Generate and verify dated readiness evidence",
		Description: "Generate and verify timestamped readiness evidence packs for enterprise review.",
		Children: []*commandSpec{
			{
				Name:        "run",
				Summary:     "Generate a new readiness evidence run",
				Description: "Generate a new readiness evidence run.",
				Options: []commandOptionSpec{
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for readiness runs", Type: optionValueString, DefaultText: "demo/logs/evidence"},
					{Name: "bundle-version", ValueName: "VALUE", Summary: "Bundle version for release-bundle commands", Type: optionValueString, DefaultText: "git short SHA"},
					{Name: "include-production", Summary: "Attempt the production-like gate if secrets are available", Type: optionValueBool},
					{Name: "secrets-env-file", ValueName: "PATH", Summary: "Secrets file for production gate", Type: optionValueString, DefaultText: "/etc/ai-control-plane/secrets.env"},
					{Name: "verbose", Summary: "Reserved for future verbose rendering", Type: optionValueBool},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindReadinessEvidenceRunOptions,
					NativeRun:  runReadinessEvidenceRunTyped,
				},
			},
			{
				Name:        "verify",
				Summary:     "Verify a generated readiness evidence directory",
				Description: "Verify a generated readiness evidence directory.",
				Options: []commandOptionSpec{
					{Name: "run-dir", ValueName: "DIR", Summary: "Specific readiness run directory to verify", Type: optionValueString},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindReadinessEvidenceVerifyOptions,
					NativeRun:  runReadinessEvidenceVerifyTyped,
				},
			},
		},
	}
}

func bindReadinessEvidenceRunOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot := bindCtx.RepoRoot
	makeBin := config.NewLoader().Tooling().MakeBinary
	options := readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    filepath.Join(repoRoot, "demo", "logs", "evidence"),
		MakeBin:       makeBin,
		BundleVersion: bundle.GetDefaultVersion(repoRoot),
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveReadinessPath(repoRoot, input.String("output-dir"))
	}
	if input.Has("bundle-version") {
		options.BundleVersion = input.String("bundle-version")
	}
	if input.Has("secrets-env-file") {
		options.SecretsEnvFile = resolveReadinessPath(repoRoot, input.String("secrets-env-file"))
	}
	options.IncludeProduction = input.Bool("include-production")
	options.Verbose = input.Bool("verbose")
	return options, nil
}

func bindReadinessEvidenceVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	runDir := input.String("run-dir")
	if runDir != "" {
		runDir = resolveReadinessPath(bindCtx.RepoRoot, runDir)
	}
	return runDir, nil
}

func runReadinessEvidenceRunTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	options := raw.(readiness.Options)
	ctx = logging.WithLogger(ctx, runCtx.Logger.With(slog.String("workflow", "readiness_evidence")))

	fmt.Fprint(runCtx.Stdout, out.Bold("Generating readiness evidence")+"\n")
	summary, err := readiness.RunContext(ctx, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Readiness evidence complete"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Run directory: %s\n", summary.RunDirectory)
	fmt.Fprintf(runCtx.Stdout, "  Summary: %s\n", filepath.Join(summary.RunDirectory, readiness.SummaryMarkdownName))
	fmt.Fprintf(runCtx.Stdout, "  Tracker: %s\n", filepath.Join(summary.RunDirectory, readiness.TrackerMarkdownName))
	fmt.Fprintf(runCtx.Stdout, "  Decision: %s\n", filepath.Join(summary.RunDirectory, readiness.DecisionMarkdownName))
	fmt.Fprintf(runCtx.Stdout, "  Overall status: %s\n", summary.OverallStatus)
	if summary.OverallStatus != "PASS" {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runReadinessEvidenceVerifyTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	runDir := raw.(string)

	if runDir == "" {
		resolvedRunDir, err := readiness.ResolveLatestRun(filepath.Join(runCtx.RepoRoot, "demo", "logs", "evidence"))
		if err != nil {
			fmt.Fprintln(runCtx.Stderr, "Error: no readiness run available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Verifying readiness evidence")+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Run directory: %s\n", runDir)

	summary, err := readiness.NewVerifier().VerifyRun(runDir)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Readiness evidence verified"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Run ID: %s\n", summary.RunID)
	fmt.Fprintf(runCtx.Stdout, "  Overall status: %s\n", summary.OverallStatus)
	fmt.Fprintf(runCtx.Stdout, "  Inventory: %s\n", filepath.Join(runDir, readiness.InventoryFileName))
	return exitcodes.ACPExitSuccess
}

func resolveReadinessPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func runReadinessEvidenceCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"deploy", "readiness-evidence"}, args, stdout, stderr)
}
