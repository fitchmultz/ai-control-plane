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
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

type statusOptions struct {
	JSON     bool
	Wide     bool
	Watch    bool
	Interval int
}

type statusInspector interface {
	Collect(context.Context, status.Options) status.StatusReport
	Close() error
}

var newStatusInspector = func(repoRoot string) statusInspector {
	return runtimeinspect.NewInspector(repoRoot)
}

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
			NativeBind: bindStatusOptions,
			NativeRun:  runStatus,
		},
	}
}

func bindStatusOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	interval, err := input.IntDefault("interval", 2)
	if err != nil || interval < 1 {
		return nil, fmt.Errorf("invalid --interval value: %q (must be a positive integer)", input.String("interval"))
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
	if runCtx.RepoRoot == "" {
		fmt.Fprintln(runCtx.Stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}
	inspector := newStatusInspector(runCtx.RepoRoot)
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
	return runTypedCommandAdapter(ctx, []string{"status"}, args, stdout, stderr)
}

func runStatusOnce(ctx context.Context, stdout *os.File, stderr *os.File, inspector statusInspector, opts status.Options, jsonOutput bool, wide bool) int {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	report := inspector.Collect(ctx, opts)

	if jsonOutput {
		if err := report.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write JSON output: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
	} else {
		if err := report.WriteHuman(stdout, wide); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write output: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
	}

	switch report.Overall {
	case status.HealthLevelHealthy:
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		return exitcodes.ACPExitDomain
	default:
		return exitcodes.ACPExitRuntime
	}
}

func runStatusWatch(ctx context.Context, stdout *os.File, stderr *os.File, inspector statusInspector, opts status.Options, jsonOutput bool, wide bool, interval int) int {
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

		collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report := inspector.Collect(collectCtx, opts)
		cancel()

		if ctx.Err() != nil {
			if !jsonOutput {
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "Watch mode stopped.")
			}
			return exitcodes.ACPExitSuccess
		}

		if jsonOutput {
			if err := report.WriteJSON(stdout); err != nil {
				fmt.Fprintf(stderr, "Error: failed to write JSON output: %v\n", err)
				return exitcodes.ACPExitRuntime
			}
		} else {
			if err := report.WriteHuman(stdout, wide); err != nil {
				fmt.Fprintf(stderr, "Error: failed to write output: %v\n", err)
				return exitcodes.ACPExitRuntime
			}
			fmt.Fprintf(stdout, "\nWatching (interval: %ds) - Press Ctrl+C to stop\n", interval)
		}

		_ = stdout.Sync()
	}
}
