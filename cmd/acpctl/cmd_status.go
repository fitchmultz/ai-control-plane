// cmd_status.go - Status subcommand implementation
//
// Purpose: Implement system health status collection and display.
// Responsibilities:
//   - Define the typed status command surface.
//   - Collect status from all domains.
//   - Format and display status output.
//
// Non-scope:
//   - Does not modify system state.
//   - Does not execute remediation actions.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

type statusOptions struct {
	JSON     bool
	Wide     bool
	Watch    bool
	Interval int
}

var newStatusInspector = newRuntimeStatusInspector

func statusCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "status",
		Summary:     "Aggregated system health overview",
		Description: "Display aggregated system health status across all domains.",
		Examples: []string{
			"acpctl status",
			"acpctl status --json",
			"acpctl status --wide",
			"acpctl status --watch --interval 5",
		},
		Options: []commandOptionSpec{
			{Name: "json", Summary: "Output in JSON format", Type: optionValueBool},
			{Name: "wide", Short: "w", Summary: "Show extended details", Type: optionValueBool},
			{Name: "watch", Short: "n", Summary: "Enable watch mode", Type: optionValueBool},
			{Name: "interval", ValueName: "SECONDS", Summary: "Watch interval in seconds", Type: optionValueInt, DefaultText: "2"},
		},
		Sections: []commandHelpSection{
			{
				Title: "Watch mode",
				Lines: []string{
					"Press Ctrl+C to exit watch mode.",
				},
			},
		},
		Bind: bindParsed(bindStatusOptions),
		Run:  runStatus,
	})
}

func bindStatusOptions(input parsedCommandInput) (statusOptions, error) {
	interval, err := input.IntDefault("interval", 2)
	if err != nil || interval < 1 {
		return statusOptions{}, fmt.Errorf("invalid --interval value: %q (must be a positive integer)", input.String("interval"))
	}
	return statusOptions{
		JSON:     input.Bool("json"),
		Wide:     input.Bool("wide"),
		Watch:    input.Bool("watch"),
		Interval: interval,
	}, nil
}

func runStatus(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(statusOptions)
	logger := ensureWorkflowLogger(runCtx)
	if !opts.Watch {
		return runRuntimeReportCommand(ctx, runCtx, logger, newStatusInspector, runtimeReportCommandConfig{
			Wide:            opts.Wide,
			Timeout:         30 * time.Second,
			TimeoutMessage:  "Status check timed out",
			CanceledMessage: "Status check canceled",
		}, func(out *output.Output, report status.StatusReport) int {
			return writeRuntimeReportOutput(runCtx, logger, out, "", report, opts.JSON, opts.Wide)
		})
	}
	return runStatusWatch(ctx, runCtx, logger, opts)
}

func runStatusWatch(ctx context.Context, runCtx commandRunContext, logger *slog.Logger, opts statusOptions) int {
	out := output.New()
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, newStatusInspector)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	ticker := time.NewTicker(time.Duration(opts.Interval) * time.Second)
	defer ticker.Stop()

	firstRun := true
	for {
		if !firstRun {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				writeWatchTermination(runCtx.Stdout, opts.JSON)
				return exitcodes.ACPExitSuccess
			}
		}
		firstRun = false

		if !opts.JSON && isTerminal() {
			fmt.Fprint(runCtx.Stdout, terminal.NewColors().Clear)
		}

		report, code, ok := collectRuntimeReportOrExit(ctx, runCtx, logger, out, inspector, runtimeReportCommandConfig{
			Wide:            opts.Wide,
			Timeout:         30 * time.Second,
			TimeoutMessage:  "Status check timed out",
			CanceledMessage: "Status check canceled",
		})
		if !ok {
			return code
		}

		if commandContextCanceled(ctx) {
			writeWatchTermination(runCtx.Stdout, opts.JSON)
			return exitcodes.ACPExitSuccess
		}

		if code := writeRuntimeReportOutput(runCtx, logger, out, "", report, opts.JSON, opts.Wide); code != exitcodes.ACPExitSuccess {
			return code
		}
		if !opts.JSON {
			fmt.Fprintf(runCtx.Stdout, "\nWatching (interval: %ds) - Press Ctrl+C to stop\n", opts.Interval)
		}

		_ = runCtx.Stdout.Sync()
	}
}
