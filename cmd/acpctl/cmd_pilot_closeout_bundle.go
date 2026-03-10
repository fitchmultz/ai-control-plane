// cmd_pilot_closeout_bundle.go - Pilot closeout bundle command implementation.
//
// Purpose:
//   - Provide a typed CLI surface for assembling and verifying pilot closeout evidence bundles.
//
// Responsibilities:
//   - Define the typed pilot-closeout-bundle command tree.
//   - Build local pilot closeout bundles from source documents and readiness evidence.
//   - Verify generated closeout bundle structure.
//
// Scope:
//   - Covers local bundle assembly and verification only.
//
// Usage:
//   - `acpctl deploy pilot-closeout-bundle build`
//   - `acpctl deploy pilot-closeout-bundle verify`
//
// Invariants/Assumptions:
//   - Bundles remain local-only under `demo/logs/pilot-closeout`.
//   - Source pilot documents are authored outside the generated bundle.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/closeout"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

func pilotCloseoutBundleCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "pilot-closeout-bundle",
		Summary:     "Assemble and verify a pilot closeout evidence bundle",
		Description: "Assemble and verify a pilot closeout evidence bundle.",
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "build",
				Summary:     "Assemble a local pilot closeout bundle",
				Description: "Assemble a local pilot closeout bundle.",
				Options: []commandOptionSpec{
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for bundle runs", Type: optionValueString, DefaultText: "demo/logs/pilot-closeout"},
					{Name: "customer", ValueName: "NAME", Summary: "Customer name", Type: optionValueString},
					{Name: "pilot-name", ValueName: "NAME", Summary: "Pilot name", Type: optionValueString},
					{Name: "decision", ValueName: "VALUE", Summary: "Decision label", Type: optionValueString, DefaultText: "PENDING_REVIEW"},
					{Name: "charter", ValueName: "PATH", Summary: "Pilot charter source document", Type: optionValueString},
					{Name: "acceptance-memo", ValueName: "PATH", Summary: "Pilot acceptance memo source document", Type: optionValueString},
					{Name: "validation-checklist", ValueName: "PATH", Summary: "Customer validation checklist source document", Type: optionValueString},
					{Name: "operator-checklist", ValueName: "PATH", Summary: "Optional operator handoff checklist source document", Type: optionValueString},
					{Name: "readiness-run-dir", ValueName: "DIR", Summary: "Specific readiness evidence run to include", Type: optionValueString},
				},
				Bind: bindPilotCloseoutBuildOptions,
				Run:  runPilotCloseoutBundleBuildTyped,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "verify",
				Summary:     "Verify a generated pilot closeout bundle",
				Description: "Verify a generated pilot closeout bundle.",
				Options: []commandOptionSpec{
					{Name: "run-dir", ValueName: "DIR", Summary: "Specific pilot closeout bundle directory to verify", Type: optionValueString},
				},
				Bind: bindPilotCloseoutVerifyOptions,
				Run:  runPilotCloseoutBundleVerifyTyped,
			}),
		},
	}
}

func bindPilotCloseoutBuildOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return nil, err
	}
	options := closeout.Options{
		RepoRoot:   repoRoot,
		OutputRoot: repopath.DemoLogsPath(repoRoot, "pilot-closeout"),
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveRepoInput(repoRoot, input.String("output-dir"))
	}
	options.Customer = input.String("customer")
	options.PilotName = input.String("pilot-name")
	options.Decision = input.StringDefault("decision", "PENDING_REVIEW")
	if input.Has("charter") {
		options.CharterPath = resolveRepoInput(repoRoot, input.String("charter"))
	}
	if input.Has("acceptance-memo") {
		options.AcceptanceMemoPath = resolveRepoInput(repoRoot, input.String("acceptance-memo"))
	}
	if input.Has("validation-checklist") {
		options.ValidationChecklist = resolveRepoInput(repoRoot, input.String("validation-checklist"))
	}
	if input.Has("operator-checklist") {
		options.OperatorChecklist = resolveRepoInput(repoRoot, input.String("operator-checklist"))
	}
	if input.Has("readiness-run-dir") {
		options.ReadinessRunDir = resolveRepoInput(repoRoot, input.String("readiness-run-dir"))
	}
	return options, nil
}

func bindPilotCloseoutVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return nil, err
	}
	runDir := input.String("run-dir")
	if runDir != "" {
		runDir = resolveRepoInput(repoRoot, runDir)
	}
	return runDir, nil
}

func runPilotCloseoutBundleBuildTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	options := raw.(closeout.Options)

	printCommandSection(runCtx.Stdout, out, "Building pilot closeout bundle")
	summary, err := closeout.Build(ctx, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	printCommandSuccess(runCtx.Stdout, out, "Pilot closeout bundle complete")
	printCommandDetail(runCtx.Stdout, "Run directory", summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Summary", filepath.Join(summary.RunDirectory, closeout.SummaryMarkdown))
	printCommandDetail(runCtx.Stdout, "Inventory", filepath.Join(summary.RunDirectory, closeout.InventoryFileName))
	return exitcodes.ACPExitSuccess
}

func runPilotCloseoutBundleVerifyTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	runDir := raw.(string)

	if runDir == "" {
		resolvedRunDir, err := closeout.ResolveLatestRun(repopath.DemoLogsPath(runCtx.RepoRoot, "pilot-closeout"))
		if err != nil {
			fmt.Fprintln(runCtx.Stderr, "Error: no pilot closeout bundle available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	printCommandSection(runCtx.Stdout, out, "Verifying pilot closeout bundle")
	printCommandDetail(runCtx.Stdout, "Run directory", runDir)
	summary, err := closeout.NewVerifier().VerifyRun(runDir)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	printCommandSuccess(runCtx.Stdout, out, "Pilot closeout bundle verified")
	printCommandDetail(runCtx.Stdout, "Customer", summary.Customer)
	printCommandDetail(runCtx.Stdout, "Pilot", summary.PilotName)
	printCommandDetail(runCtx.Stdout, "Decision", summary.Decision)
	return exitcodes.ACPExitSuccess
}

func runPilotCloseoutBundleCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroupPath(ctx, []string{"deploy", "pilot-closeout-bundle"}, args, stdout, stderr)
}
