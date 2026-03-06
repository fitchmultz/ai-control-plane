// cmd_status.go - Status subcommand implementation
//
// Purpose: Implement system health status collection and display
// Responsibilities:
//   - Parse status flags (json, wide, watch)
//   - Collect status from all domains
//   - Format and display status output
//
// Non-scope:
//   - Does not modify system state
//   - Does not execute remediation actions

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/collectors"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

func printStatusHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl status [OPTIONS]

Display aggregated system health status across all domains.

Options:
  --json       Output in JSON format
  --wide, -w   Show extended details
  --watch, -n  Watch mode - continuous monitoring (interval in seconds, default: 2)
  --help, -h   Show this help message

Examples:
  acpctl status              # Show human-readable status summary
  acpctl status --json       # Output JSON for programmatic use
  acpctl status --wide       # Show detailed information
  acpctl status --watch      # Continuous monitoring (2 second interval)
  acpctl status --watch=5    # Continuous monitoring (5 second interval)

Exit codes:
  0   All systems healthy
  1   One or more systems unhealthy (domain non-success)
  2   Prerequisites not ready (docker not installed)
  3   Runtime/internal error
  64  Usage error

Watch Mode:
  Press Ctrl+C to exit watch mode.
`)
}

func runStatusCommand(args []string, stdout *os.File, stderr *os.File) int {
	var jsonOutput, wide, watchMode bool
	var watchInterval int

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			printStatusHelp(stdout)
			return exitcodes.ACPExitSuccess
		case arg == "--json":
			jsonOutput = true
		case arg == "--wide" || arg == "-w":
			wide = true
		case arg == "--watch" || arg == "-n":
			watchMode = true
			watchInterval = 2
		case strings.HasPrefix(arg, "--watch="):
			watchMode = true
			intervalStr := strings.TrimPrefix(arg, "--watch=")
			interval, err := strconv.Atoi(intervalStr)
			if err != nil || interval < 1 {
				fmt.Fprintf(stderr, "Error: Invalid watch interval: %s\n", intervalStr)
				return exitcodes.ACPExitUsage
			}
			watchInterval = interval
		case strings.HasPrefix(arg, "-n="):
			watchMode = true
			intervalStr := strings.TrimPrefix(arg, "-n=")
			interval, err := strconv.Atoi(intervalStr)
			if err != nil || interval < 1 {
				fmt.Fprintf(stderr, "Error: Invalid watch interval: %s\n", intervalStr)
				return exitcodes.ACPExitUsage
			}
			watchInterval = interval
		default:
			fmt.Fprintf(stderr, "Error: Unknown option: %s\n", arg)
			return exitcodes.ACPExitUsage
		}
	}

	repoRoot := detectRepoRoot()
	if repoRoot == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}

	gatewayHost := os.Getenv("GATEWAY_HOST")
	if gatewayHost == "" {
		gatewayHost = config.DefaultGatewayHost
	}
	litellmPort := os.Getenv("LITELLM_PORT")
	if litellmPort == "" {
		litellmPort = strconv.Itoa(config.DefaultLiteLLMPort)
	}

	opts := status.Options{
		RepoRoot:    repoRoot,
		GatewayHost: gatewayHost,
		LITELLMPort: litellmPort,
		Wide:        wide,
	}

	collectorsList := []status.Collector{
		collectors.GatewayCollector{Host: gatewayHost, Port: litellmPort},
		collectors.NewDatabaseCollector(repoRoot),
		collectors.NewKeysCollector(repoRoot),
		collectors.NewBudgetCollector(repoRoot),
		collectors.NewDetectionsCollector(repoRoot),
	}

	if !watchMode {
		return runStatusOnce(stdout, stderr, collectorsList, opts, jsonOutput, wide)
	}

	return runStatusWatch(stdout, stderr, collectorsList, opts, jsonOutput, wide, watchInterval)
}

func runStatusOnce(stdout *os.File, stderr *os.File, collectorsList []status.Collector, opts status.Options, jsonOutput bool, wide bool) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := status.CollectAll(ctx, collectorsList, opts)

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

func runStatusWatch(stdout *os.File, stderr *os.File, collectorsList []status.Collector, opts status.Options, jsonOutput bool, wide bool, interval int) int {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Create a context that cancels on interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

		// Create timeout context that also respects signal cancellation
		collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		report := status.CollectAll(collectCtx, collectorsList, opts)
		cancel()

		// Check if context was cancelled during collection
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
