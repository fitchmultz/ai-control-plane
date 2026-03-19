// cmd_evidence.go - Vendor evidence ingest command surface.
//
// Purpose:
//   - Own the typed vendor evidence ingest workflow for local host-first use.
//
// Responsibilities:
//   - Define the `evidence ingest` command tree.
//   - Read vendor export payloads from file or stdin.
//   - Delegate normalization and schema validation to internal/ingest.
//
// Scope:
//   - Evidence ingest command bindings and operator output only.
//
// Usage:
//   - `acpctl evidence ingest --format compliance-api --file export.json`
//   - `cat export.json | acpctl evidence ingest --format compliance-api`
//
// Invariants/Assumptions:
//   - Supported ingest is local/file/stdin based, not a long-running service.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/ingest"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

type evidenceIngestOptions struct {
	RepoRoot   string
	OutputRoot string
	InputPath  string
	SourceName string
	Format     ingest.Format
}

func evidenceCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "evidence",
		Summary:     "Vendor evidence ingest workflows",
		Description: "Vendor evidence ingest workflows.",
		Examples: []string{
			"acpctl evidence ingest --format compliance-api --file examples/vendor-evidence/compliance_export.sample.json",
			"cat export.json | acpctl evidence ingest --format compliance-api",
		},
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "ingest",
				Summary:     "Normalize supported vendor evidence into ACP schema artifacts",
				Description: "Normalize supported vendor evidence into ACP schema artifacts.",
				Options: []commandOptionSpec{
					{Name: "format", ValueName: "FORMAT", Summary: "Input format (currently: compliance-api)", Type: optionValueString, DefaultText: "compliance-api"},
					{Name: "file", ValueName: "PATH", Summary: "Read input from a JSON file instead of stdin", Type: optionValueString},
					{Name: "output-dir", ValueName: "DIR", Summary: "Output root for ingest runs", Type: optionValueString, DefaultText: "demo/logs/evidence/vendor-ingest"},
					{Name: "source-name", ValueName: "NAME", Summary: "Logical source.service.name label for normalized output", Type: optionValueString, DefaultText: ingest.DefaultSourceService},
				},
				Bind: bindRepoParsed(bindEvidenceIngestOptions),
				Run:  runEvidenceIngestTyped,
			}),
		},
	}
}

func bindEvidenceIngestOptions(bindCtx commandBindContext, input parsedCommandInput) (evidenceIngestOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return evidenceIngestOptions{}, err
	}
	format, err := normalizeEvidenceFormat(input.String("format"))
	if err != nil {
		return evidenceIngestOptions{}, err
	}
	options := evidenceIngestOptions{
		RepoRoot:   repoRoot,
		OutputRoot: repopath.DemoLogsPath(repoRoot, "evidence", ingest.DefaultOutputSubdir),
		SourceName: ingest.DefaultSourceService,
		Format:     format,
	}
	if input.Has("file") {
		options.InputPath = resolveRepoInput(repoRoot, input.NormalizedString("file"))
	}
	if input.Has("output-dir") {
		options.OutputRoot = resolveRepoInput(repoRoot, input.NormalizedString("output-dir"))
	}
	if input.Has("source-name") {
		options.SourceName = input.NormalizedString("source-name")
	}
	return options, nil
}

func runEvidenceIngestTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	opts := raw.(evidenceIngestOptions)
	payload, inputLabel, inputCode, err := loadEvidencePayload(opts.InputPath)
	if err != nil {
		return failCommand(runCtx.Stderr, out, inputCode, err, "evidence ingest input error")
	}

	printCommandSection(runCtx.Stdout, out, "Ingesting vendor evidence")
	result, err := ingest.Ingest(ctx, ingest.Options{
		RepoRoot:     opts.RepoRoot,
		OutputRoot:   opts.OutputRoot,
		InputPath:    inputLabel,
		SourceName:   opts.SourceName,
		Format:       opts.Format,
		InputPayload: payload,
	})
	if err != nil {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "evidence ingest failed")
	}

	printCommandSuccess(runCtx.Stdout, out, "Vendor evidence ingest complete")
	printCommandDetail(runCtx.Stdout, "Run directory", result.Summary.RunDirectory)
	printCommandDetail(runCtx.Stdout, "Format", result.Summary.Format)
	printCommandDetail(runCtx.Stdout, "Source type", result.Summary.SourceType)
	printCommandDetail(runCtx.Stdout, "Source name", result.Summary.SourceName)
	printCommandDetail(runCtx.Stdout, "Input", inputLabel)
	printCommandDetail(runCtx.Stdout, "Records", result.Summary.RecordCount)
	printCommandDetail(runCtx.Stdout, "Normalized", result.Summary.NormalizedPath)
	printCommandDetail(runCtx.Stdout, "Summary", result.Summary.RunDirectory+string(os.PathSeparator)+ingest.SummaryMarkdownName)
	printCommandDetail(runCtx.Stdout, "Issues", result.Summary.ValidationIssueCount)
	if len(result.Issues) > 0 {
		return failValidation(runCtx.Stderr, out, result.Issues, "Vendor evidence validation failed")
	}
	return exitcodes.ACPExitSuccess
}

func loadEvidencePayload(inputPath string) ([]byte, string, int, error) {
	if strings.TrimSpace(inputPath) != "" {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("read input file: %w", err)
		}
		return data, inputPath, exitcodes.ACPExitSuccess, nil
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("inspect stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return nil, "", exitcodes.ACPExitUsage, fmt.Errorf("provide --file or pipe JSON payload on stdin")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, "", exitcodes.ACPExitUsage, fmt.Errorf("stdin was empty")
	}
	return data, "stdin", exitcodes.ACPExitSuccess, nil
}

func normalizeEvidenceFormat(raw string) (ingest.Format, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(ingest.FormatComplianceAPI):
		return ingest.FormatComplianceAPI, nil
	default:
		return "", fmt.Errorf("unsupported evidence format %q", raw)
	}
}

func runEvidenceCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroupPath(ctx, []string{"evidence"}, args, stdout, stderr)
}
