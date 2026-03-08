// cmd_doctor.go - Doctor subcommand implementation
//
// Purpose: Implement environment preflight diagnostics
// Responsibilities:
//   - Parse doctor flags
//   - Run diagnostic checks
//   - Output results in human or JSON format
//
// Non-scope:
//   - Does not modify system state (unless --fix is used)
//   - Does not replace operational monitoring
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

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/doctor"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

func printDoctorHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl doctor [OPTIONS]

Environment preflight diagnostics for AI Control Plane.

Checks:
  docker_available    Docker binary and daemon accessible
  ports_free          Required ports (4000, 5432) are available
  env_vars_set        Required environment variables configured
  gateway_healthy     LiteLLM gateway responding
  db_connectable      PostgreSQL accepting connections
  config_valid        Deployment configuration valid
  credentials_valid   Master key valid and usable

Options:
  --json              Output in JSON format
  --wide, -w          Show extended details
  --fix               Attempt safe auto-remediation
  --skip-check CHECK  Skip specific check (repeatable)
  --help, -h          Show this help message

Examples:
  acpctl doctor
  acpctl doctor --json
  acpctl doctor --fix --skip-check db_connectable
  acpctl doctor --wide

Exit codes:
  0   All checks passed
  1   One or more checks failed (domain)
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runDoctorCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	var jsonOutput, wide, fix bool
	skipChecks := make(map[string]struct{})

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			printDoctorHelp(stdout)
			return exitcodes.ACPExitSuccess
		case arg == "--json":
			jsonOutput = true
		case arg == "--wide" || arg == "-w":
			wide = true
		case arg == "--fix":
			fix = true
		case arg == "--skip-check":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --skip-check requires a CHECK argument")
				return exitcodes.ACPExitUsage
			}
			skipChecks[args[i+1]] = struct{}{}
			i++
		default:
			fmt.Fprintf(stderr, "Error: Unknown option: %s\n", arg)
			return exitcodes.ACPExitUsage
		}
	}

	repoRoot := detectRepoRootWithContext(ctx)
	if repoRoot == "" {
		fmt.Fprintln(stderr, "Error: failed to detect repository root")
		return exitcodes.ACPExitRuntime
	}
	gatewayRuntime := config.NewLoader().Gateway(true)

	opts := doctor.Options{
		RepoRoot:      repoRoot,
		GatewayHost:   gatewayRuntime.Host,
		GatewayPort:   gatewayRuntime.Port,
		RequiredPorts: config.RequiredPorts(),
		SkipChecks:    skipChecks,
		Fix:           fix,
		Wide:          wide,
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	report := doctor.Run(ctx, doctor.DefaultChecks(), opts)

	if jsonOutput {
		if err := report.WriteJSON(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write JSON output: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
	} else {
		if err := writeDoctorHuman(stdout, report, wide); err != nil {
			fmt.Fprintf(stderr, "Error: failed to write output: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
	}

	return doctor.ExitCodeForReport(report)
}

func writeDoctorHuman(w *os.File, report doctor.Report, wide bool) error {
	colors := terminal.NewColors()
	sf := terminal.NewStatusFormatter()

	formatStatus := func(level status.HealthLevel) string {
		switch level {
		case status.HealthLevelHealthy:
			return sf.OK()
		case status.HealthLevelWarning:
			return sf.Warn()
		case status.HealthLevelUnhealthy:
			return sf.Fail()
		default:
			return "[UNK]"
		}
	}

	fmt.Fprintln(w, colors.Bold+"=== AI Control Plane - Doctor Diagnostics ==="+colors.Reset)
	fmt.Fprintln(w, "")

	for _, result := range report.Results {
		paddedName := fmt.Sprintf("%-20s", result.Name)
		fmt.Fprintf(w, "%s %s %s\n", paddedName, formatStatus(result.Level), result.Message)

		if len(result.Suggestions) > 0 && (result.Level == status.HealthLevelUnhealthy || result.Level == status.HealthLevelWarning) {
			for _, suggestion := range result.Suggestions {
				fmt.Fprintf(w, "                     %s\n", suggestion)
			}
		}

		if result.FixApplied {
			fmt.Fprintf(w, "                     %s %s\n", sf.OK(), result.FixMessage)
		}

		if wide && result.Details != nil && len(result.Details) > 0 {
			for k, v := range result.Details {
				fmt.Fprintf(w, "                     %s: %v\n", k, v)
			}
		}
	}

	fmt.Fprintln(w, "")
	var overallStr string
	switch report.Overall {
	case status.HealthLevelHealthy:
		overallStr = sf.Healthy()
	case status.HealthLevelWarning:
		overallStr = sf.Warning()
	case status.HealthLevelUnhealthy:
		overallStr = sf.Unhealthy()
	default:
		overallStr = "UNKNOWN"
	}
	fmt.Fprintf(w, "Overall: %s (%s)\n", overallStr, report.Duration)

	return nil
}
