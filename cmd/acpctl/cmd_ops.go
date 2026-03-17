// cmd_ops.go - Typed operator reporting command surface.
//
// Purpose:
//   - Expose one-command operator reporting workflows.
//
// Responsibilities:
//   - Define the `ops report` command contract.
//   - Render and optionally archive the canonical runtime report.
//   - Return domain failures when runtime health is degraded.
//
// Scope:
//   - `acpctl ops report` only.
//
// Usage:
//   - Invoked through the typed command registry and `make operator-report`.
//
// Invariants/Assumptions:
//   - Output is derived from the shared runtime inspection model.
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/operatorreport"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type opsReportOptions struct {
	Format     string
	Wide       bool
	ArchiveDir string
}

func opsCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "ops",
		Summary:     "Operator reporting workflows",
		Description: "Operator reporting workflows.",
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "report",
				Summary:     "Render a canonical operator status report",
				Description: "Render a canonical operator report backed by the typed runtime status model.",
				Examples: []string{
					"acpctl ops report",
					"acpctl ops report --format json",
					"acpctl ops report --wide",
				},
				Options: []commandOptionSpec{
					{Name: "format", ValueName: "FORMAT", Summary: "Output format: markdown or json", Type: optionValueString, DefaultText: "markdown"},
					{Name: "wide", Summary: "Include extended component details", Type: optionValueBool},
					{Name: "archive-dir", ValueName: "DIR", Summary: "Archive directory", Type: optionValueString, DefaultText: "demo/backups/operator-reports"},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) opsReportOptions {
					return opsReportOptions{
						Format:     input.NormalizedString("format"),
						Wide:       input.Bool("wide"),
						ArchiveDir: input.StringDefault("archive-dir", "demo/backups/operator-reports"),
					}
				}),
				Run: runOpsReport,
			}),
		},
	}
}

func runOpsReport(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(opsReportOptions)
	logger := ensureWorkflowLogger(runCtx)

	return runRuntimeReportCommand(ctx, runCtx, logger, newRuntimeStatusInspector, runtimeReportCommandConfig{
		Wide:            opts.Wide,
		Timeout:         30 * time.Second,
		TimeoutMessage:  "Operator report timed out",
		CanceledMessage: "Operator report canceled",
	}, func(_ *output.Output, runtimeReport status.StatusReport) int {
		format, err := normalizeOpsReportFormat(opts.Format)
		if err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
			return exitcodes.ACPExitUsage
		}

		payload, ext, err := operatorreport.Render(runtimeReport, operatorreport.Request{
			Format: format,
			Wide:   opts.Wide,
		})
		if err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: operator report render failed: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if _, err := runCtx.Stdout.Write(payload); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: write operator report: %v\n", err)
			return exitcodes.ACPExitRuntime
		}

		stamp := time.Now().UTC().Format("2006-01-02T150405Z")
		if _, err := operatorreport.Archive(runCtx.RepoRoot, opts.ArchiveDir, stamp, payload, ext); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: archive operator report: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if runtimeReport.Overall != status.HealthLevelHealthy {
			return exitcodes.ACPExitDomain
		}
		return exitcodes.ACPExitSuccess
	})
}

func normalizeOpsReportFormat(raw string) (operatorreport.Format, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "markdown", "md":
		return operatorreport.FormatMarkdown, nil
	case "json":
		return operatorreport.FormatJSON, nil
	default:
		return "", fmt.Errorf("invalid operator report format %q (expected markdown or json)", raw)
	}
}
