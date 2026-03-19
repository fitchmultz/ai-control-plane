// cmd_assessor_packet.go - Assessor packet command implementation.
//
// Purpose:
//   - Provide a typed CLI surface for assembling and verifying ACP-native assessor packets.
//
// Responsibilities:
//   - Define the typed assessor-packet command tree.
//   - Build local assessor packets from canonical reviewer docs and verified readiness evidence.
//   - Verify generated assessor packet structure and truth markers.
//
// Scope:
//   - Covers local packet assembly and verification only.
//
// Usage:
//   - `acpctl deploy assessor-packet build`
//   - `acpctl deploy assessor-packet verify`
//
// Invariants/Assumptions:
//   - Packets remain local-only under `demo/logs/assessor-packet`.
//   - The packet is preparation infrastructure only and does not claim external validation.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/assessor"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

func assessorPacketCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "assessor-packet",
		Summary:     "Assemble and verify an ACP-native assessor handoff packet",
		Description: "Assemble and verify an ACP-native assessor handoff packet.",
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "build",
				Summary:     "Assemble a local assessor handoff packet",
				Description: "Assemble a local assessor handoff packet.",
				Options: []commandOptionSpec{
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for assessor packet runs", Type: optionValueString, DefaultText: "demo/logs/assessor-packet"},
					{Name: "readiness-run-dir", ValueName: "DIR", Summary: "Specific readiness evidence run to include", Type: optionValueString},
				},
				Bind: bindRepoParsed(bindAssessorPacketBuildOptions),
				Run:  runAssessorPacketBuildTyped,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "verify",
				Summary:     "Verify a generated assessor packet",
				Description: "Verify a generated assessor packet.",
				Options: []commandOptionSpec{
					{Name: "run-dir", ValueName: "DIR", Summary: "Specific assessor packet directory to verify", Type: optionValueString},
				},
				Bind: bindRepoParsed(bindAssessorPacketVerifyOptions),
				Run:  runAssessorPacketVerifyTyped,
			}),
		},
	}
}

func bindAssessorPacketBuildOptions(bindCtx commandBindContext, input parsedCommandInput) (assessor.Options, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return assessor.Options{}, err
	}
	options := assessor.Options{
		RepoRoot:   repoRoot,
		OutputRoot: repopath.DemoLogsPath(repoRoot, "assessor-packet"),
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveRepoInput(repoRoot, input.NormalizedString("output-dir"))
	}
	if input.Has("readiness-run-dir") {
		options.ReadinessRunDir = resolveRepoInput(repoRoot, input.NormalizedString("readiness-run-dir"))
	}
	return options, nil
}

func bindAssessorPacketVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (string, error) {
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

func runAssessorPacketBuildTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	options := raw.(assessor.Options)
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "assessor_packet_build")))

	printCommandSection(runCtx.Stdout, out, "Building assessor packet")
	summary, err := assessor.Build(ctx, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	printCommandSuccess(runCtx.Stdout, out, "Assessor packet complete")
	printCommandDetail(runCtx.Stdout, "Run directory", summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Summary", filepath.Join(summary.RunDirectory, assessor.SummaryMarkdownName))
	printCommandDetail(runCtx.Stdout, "Inventory", filepath.Join(summary.RunDirectory, assessor.InventoryFileName))
	printCommandDetail(runCtx.Stdout, "Readiness run", summary.ReadinessRunDir)
	printCommandDetail(runCtx.Stdout, "Release bundle", summary.ReleaseBundlePacketPath)
	return exitcodes.ACPExitSuccess
}

func runAssessorPacketVerifyTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	runDir := raw.(string)
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "assessor_packet_verify")))

	if runDir == "" {
		resolvedRunDir, err := assessor.ResolveLatestRun(repopath.DemoLogsPath(runCtx.RepoRoot, "assessor-packet"))
		if err != nil {
			fmt.Fprintln(runCtx.Stderr, "Error: no assessor packet available; use --run-dir or generate one first")
			return exitcodes.ACPExitUsage
		}
		runDir = resolvedRunDir
	}

	printCommandSection(runCtx.Stdout, out, "Verifying assessor packet")
	printCommandDetail(runCtx.Stdout, "Run directory", runDir)
	summary, err := assessor.NewVerifier().VerifyRun(ctx, runDir)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	printCommandSuccess(runCtx.Stdout, out, "Assessor packet verified")
	printCommandDetail(runCtx.Stdout, "Roadmap item #22", summary.RoadmapItemStatus)
	printCommandDetail(runCtx.Stdout, "Preparation only", fmt.Sprintf("%t", summary.PreparationOnly))
	printCommandDetail(runCtx.Stdout, "Readiness run", summary.ReadinessRunDir)
	return exitcodes.ACPExitSuccess
}

func runAssessorPacketCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroupPath(ctx, []string{"deploy", "assessor-packet"}, args, stdout, stderr)
}
