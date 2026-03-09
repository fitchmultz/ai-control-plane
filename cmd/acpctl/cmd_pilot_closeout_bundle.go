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
)

func pilotCloseoutBundleCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "pilot-closeout-bundle",
		Summary:     "Assemble and verify a pilot closeout evidence bundle",
		Description: "Assemble and verify a pilot closeout evidence bundle.",
		Children: []*commandSpec{
			{
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
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindPilotCloseoutBuildOptions,
					NativeRun:  runPilotCloseoutBundleBuildTyped,
				},
			},
			{
				Name:        "verify",
				Summary:     "Verify a generated pilot closeout bundle",
				Description: "Verify a generated pilot closeout bundle.",
				Options: []commandOptionSpec{
					{Name: "run-dir", ValueName: "DIR", Summary: "Specific pilot closeout bundle directory to verify", Type: optionValueString},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindPilotCloseoutVerifyOptions,
					NativeRun:  runPilotCloseoutBundleVerifyTyped,
				},
			},
		},
	}
}

func bindPilotCloseoutBuildOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot := bindCtx.RepoRoot
	options := closeout.Options{
		RepoRoot:   repoRoot,
		OutputRoot: filepath.Join(repoRoot, "demo", "logs", "pilot-closeout"),
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveReadinessPath(repoRoot, input.String("output-dir"))
	}
	options.Customer = input.String("customer")
	options.PilotName = input.String("pilot-name")
	options.Decision = input.StringDefault("decision", "PENDING_REVIEW")
	if input.Has("charter") {
		options.CharterPath = resolveReadinessPath(repoRoot, input.String("charter"))
	}
	if input.Has("acceptance-memo") {
		options.AcceptanceMemoPath = resolveReadinessPath(repoRoot, input.String("acceptance-memo"))
	}
	if input.Has("validation-checklist") {
		options.ValidationChecklist = resolveReadinessPath(repoRoot, input.String("validation-checklist"))
	}
	if input.Has("operator-checklist") {
		options.OperatorChecklist = resolveReadinessPath(repoRoot, input.String("operator-checklist"))
	}
	if input.Has("readiness-run-dir") {
		options.ReadinessRunDir = resolveReadinessPath(repoRoot, input.String("readiness-run-dir"))
	}
	return options, nil
}

func bindPilotCloseoutVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	runDir := input.String("run-dir")
	if runDir != "" {
		runDir = resolveReadinessPath(bindCtx.RepoRoot, runDir)
	}
	return runDir, nil
}

func runPilotCloseoutBundleBuildTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	options := raw.(closeout.Options)

	fmt.Fprint(runCtx.Stdout, out.Bold("Building pilot closeout bundle")+"\n")
	summary, err := closeout.Build(ctx, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Pilot closeout bundle complete"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Run directory: %s\n", summary.RunDirectory)
	fmt.Fprintf(runCtx.Stdout, "  Summary: %s\n", filepath.Join(summary.RunDirectory, closeout.SummaryMarkdown))
	fmt.Fprintf(runCtx.Stdout, "  Inventory: %s\n", filepath.Join(summary.RunDirectory, closeout.InventoryFileName))
	return exitcodes.ACPExitSuccess
}

func runPilotCloseoutBundleVerifyTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	runDir := raw.(string)

	if runDir == "" {
		resolvedRunDir, err := closeout.ResolveLatestRun(filepath.Join(runCtx.RepoRoot, "demo", "logs", "pilot-closeout"))
		if err != nil {
			fmt.Fprintln(runCtx.Stderr, "Error: no pilot closeout bundle available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Verifying pilot closeout bundle")+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Run directory: %s\n", runDir)
	summary, err := closeout.NewVerifier().VerifyRun(runDir)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Pilot closeout bundle verified"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Customer: %s\n", summary.Customer)
	fmt.Fprintf(runCtx.Stdout, "  Pilot: %s\n", summary.PilotName)
	fmt.Fprintf(runCtx.Stdout, "  Decision: %s\n", summary.Decision)
	return exitcodes.ACPExitSuccess
}

func runPilotCloseoutBundleCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		if path, err := findCommandPath([]string{"deploy", "pilot-closeout-bundle"}); err == nil {
			printCommandHelp(stdout, path)
		}
		return exitcodes.ACPExitUsage
	}
	return runTypedCommandAdapter(ctx, []string{"deploy", "pilot-closeout-bundle"}, args, stdout, stderr)
}
