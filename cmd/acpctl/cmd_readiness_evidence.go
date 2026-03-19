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
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/readiness"
)

func readinessEvidenceCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "readiness-evidence",
		Summary:     "Generate and verify dated readiness evidence",
		Description: "Generate and verify timestamped readiness evidence packs for enterprise review.",
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "run",
				Summary:     "Generate a new readiness evidence run",
				Description: "Generate a new readiness evidence run.",
				Options: []commandOptionSpec{
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for readiness runs", Type: optionValueString, DefaultText: "demo/logs/evidence"},
					{Name: "bundle-version", ValueName: "VALUE", Summary: "Bundle version for release-bundle commands", Type: optionValueString, DefaultText: "VERSION file"},
					{Name: "include-production", Summary: "Attempt the production-like gate if secrets are available", Type: optionValueBool},
					{Name: "secrets-env-file", ValueName: "PATH", Summary: "Secrets file for production gate", Type: optionValueString, DefaultText: "/etc/ai-control-plane/secrets.env"},
					{Name: "verbose", Summary: "Reserved for future verbose rendering", Type: optionValueBool},
				},
				Bind: bindRepoParsed(bindReadinessEvidenceRunOptions),
				Run:  runReadinessEvidenceRunTyped,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "verify",
				Summary:     "Verify a generated readiness evidence directory",
				Description: "Verify a generated readiness evidence directory.",
				Options: []commandOptionSpec{
					{Name: "run-dir", ValueName: "DIR", Summary: "Specific readiness run directory to verify", Type: optionValueString},
				},
				Bind: bindRepoParsed(bindReadinessEvidenceVerifyOptions),
				Run:  runReadinessEvidenceVerifyTyped,
			}),
		},
	}
}

func bindReadinessEvidenceRunOptions(bindCtx commandBindContext, input parsedCommandInput) (readiness.Options, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return readiness.Options{}, err
	}
	makeBin := config.NewLoader().Tooling().MakeBinary
	options := readiness.Options{
		RepoRoot:      repoRoot,
		OutputRoot:    repopath.DemoLogsPath(repoRoot, "evidence"),
		MakeBin:       makeBin,
		BundleVersion: bundle.GetDefaultVersion(repoRoot),
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveRepoInput(repoRoot, input.NormalizedString("output-dir"))
	}
	if input.Has("bundle-version") {
		options.BundleVersion = input.NormalizedString("bundle-version")
	}
	if input.Has("secrets-env-file") {
		options.SecretsEnvFile = resolveRepoInput(repoRoot, input.NormalizedString("secrets-env-file"))
	}
	options.IncludeProduction = input.Bool("include-production")
	options.Verbose = input.Bool("verbose")
	return options, nil
}

func bindReadinessEvidenceVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (string, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return "", err
	}
	runDir := input.NormalizedString("run-dir")
	if runDir != "" {
		runDir = resolveRepoInput(repoRoot, runDir)
	}
	return runDir, nil
}

func runReadinessEvidenceRunTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	options := raw.(readiness.Options)
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "readiness_evidence_run")))

	printCommandSection(runCtx.Stdout, out, "Generating readiness evidence")
	summary, err := readiness.RunContext(ctx, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	printCommandSuccess(runCtx.Stdout, out, "Readiness evidence complete")
	printCommandDetail(runCtx.Stdout, "Run directory", summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Summary", filepath.Join(summary.RunDirectory, readiness.SummaryMarkdownName))
	printCommandDetail(runCtx.Stdout, "Tracker", filepath.Join(summary.RunDirectory, readiness.TrackerMarkdownName))
	printCommandDetail(runCtx.Stdout, "Decision", filepath.Join(summary.RunDirectory, readiness.DecisionMarkdownName))
	printCommandDetail(runCtx.Stdout, "Overall status", summary.OverallStatus)
	if summary.OverallStatus != "PASS" {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runReadinessEvidenceVerifyTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	runDir := raw.(string)
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "readiness_evidence_verify")))

	if runDir == "" {
		resolvedRunDir, err := readiness.ResolveLatestRun(repopath.DemoLogsPath(runCtx.RepoRoot, "evidence"))
		if err != nil {
			fmt.Fprintln(runCtx.Stderr, "Error: no readiness run available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	printCommandSection(runCtx.Stdout, out, "Verifying readiness evidence")
	printCommandDetail(runCtx.Stdout, "Run directory", runDir)

	summary, err := readiness.NewVerifier().VerifyRun(ctx, runDir)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	printCommandSuccess(runCtx.Stdout, out, "Readiness evidence verified")
	printCommandDetail(runCtx.Stdout, "Run ID", summary.RunID)
	printCommandDetail(runCtx.Stdout, "Overall status", summary.OverallStatus)
	printCommandDetail(runCtx.Stdout, "Inventory", filepath.Join(runDir, readiness.InventoryFileName))
	return exitcodes.ACPExitSuccess
}

func runReadinessEvidenceCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"deploy", "readiness-evidence"}, args, stdout, stderr)
}
