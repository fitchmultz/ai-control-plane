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
	"io"
	"os"
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
	return &commandSpec{
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
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindParsed(bindStatusOptions),
			NativeRun:  runStatus,
		},
	}
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
	out := output.New()
	logger := ensureWorkflowLogger(runCtx)
	inspector, code := openRuntimeStatusInspector(runCtx, logger, out, newStatusInspector)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer inspector.Close()

	statusOpts := status.Options{
		RepoRoot: runCtx.RepoRoot,
		Wide:     opts.Wide,
	}
	if !opts.Watch {
		return runStatusOnce(ctx, runCtx.Stdout, runCtx.Stderr, inspector, statusOpts, opts.JSON, opts.Wide)
	}
	return runStatusWatch(ctx, runCtx.Stdout, runCtx.Stderr, inspector, statusOpts, opts.JSON, opts.Wide, opts.Interval)
}

func runStatusCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"status"}, args, stdout, stderr)
}

func runStatusOnce(ctx context.Context, stdout *os.File, stderr *os.File, inspector runtimeStatusInspector, opts status.Options, jsonOutput bool, wide bool) int {
	report, _, cancel := collectRuntimeStatusReport(ctx, inspector, opts.RepoRoot, opts.Wide, 30*time.Second)
	defer cancel()

	if code := writeStructuredCommandOutput(stdout, stderr, jsonOutput, report.WriteJSON, func(w io.Writer) error {
		return report.WriteHuman(w, wide)
	}); code != exitcodes.ACPExitSuccess {
		return code
	}
	return exitCodeForHealthLevel(report.Overall)
}

func runStatusWatch(ctx context.Context, stdout *os.File, stderr *os.File, inspector runtimeStatusInspector, opts status.Options, jsonOutput bool, wide bool, interval int) int {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	firstRun := true
	for {
		if !firstRun {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				if !jsonOutput {
					fmt.Fprintln(stdout)
					fmt.Fprintln(stdout, "Watch mode stopped.")
				}
				return exitcodes.ACPExitSuccess
			}
		}
		firstRun = false

		if !jsonOutput && isTerminal() {
			colors := terminal.NewColors()
			fmt.Fprint(stdout, colors.Clear)
		}

		report, _, cancel := collectRuntimeStatusReport(ctx, inspector, opts.RepoRoot, opts.Wide, 30*time.Second)
		cancel()

		if ctx.Err() != nil {
			if !jsonOutput {
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "Watch mode stopped.")
			}
			return exitcodes.ACPExitSuccess
		}

		if code := writeStructuredCommandOutput(stdout, stderr, jsonOutput, report.WriteJSON, func(w io.Writer) error {
			return report.WriteHuman(w, wide)
		}); code != exitcodes.ACPExitSuccess {
			return code
		}
		if !jsonOutput {
			fmt.Fprintf(stdout, "\nWatching (interval: %ds) - Press Ctrl+C to stop\n", interval)
		}

		_ = stdout.Sync()
	}
}
