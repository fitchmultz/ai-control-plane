// cmd_doctor.go - Doctor subcommand implementation
//
// Purpose: Implement environment preflight diagnostics.
// Responsibilities:
//   - Define the typed doctor command surface.
//   - Run diagnostic checks.
//   - Output results in human or JSON format.
//
// Non-scope:
//   - Does not modify system state unless `--fix` is used.
//   - Does not replace operational monitoring.
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
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/doctor"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

type doctorOptions struct {
	JSON       bool
	Wide       bool
	Fix        bool
	Notify     bool
	SkipChecks []string
}

var newDoctorInspector = newRuntimeStatusInspector

func doctorCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "doctor",
		Summary:     "Environment preflight diagnostics",
		Description: "Environment preflight diagnostics for AI Control Plane.",
		Examples: []string{
			"acpctl doctor",
			"acpctl doctor --json",
			"acpctl doctor --fix --skip-check db_connectable",
			"acpctl doctor --notify",
			"acpctl doctor --wide",
		},
		Options: []commandOptionSpec{
			{Name: "json", Summary: "Output in JSON format", Type: optionValueBool},
			{Name: "wide", Short: "w", Summary: "Show extended details", Type: optionValueBool},
			{Name: "fix", Summary: "Attempt safe auto-remediation", Type: optionValueBool},
			{Name: "notify", Summary: "Emit budget and security findings through configured alert adapters", Type: optionValueBool},
			{Name: "skip-check", ValueName: "CHECK", Summary: "Skip a specific check", Type: optionValueString, Repeatable: true},
		},
		Sections: []commandHelpSection{
			{
				Title: "Alert environment",
				Lines: []string{
					"ACP_ALERT_GENERIC_WEBHOOK_URL",
					"ACP_ALERT_SLACK_WEBHOOK_URL",
					"GENERIC_WEBHOOK_URL",
					"SLACK_WEBHOOK_URL",
				},
			},
		},
		Bind: bindParsedValue(func(input parsedCommandInput) doctorOptions {
			return doctorOptions{
				JSON:       input.Bool("json"),
				Wide:       input.Bool("wide"),
				Fix:        input.Bool("fix"),
				Notify:     input.Bool("notify"),
				SkipChecks: input.Strings("skip-check"),
			}
		}),
		Run: runDoctor,
	})
}

func runDoctor(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(doctorOptions)
	logger := ensureWorkflowLogger(runCtx)
	skipChecks := make(map[string]struct{}, len(opts.SkipChecks))
	for _, name := range opts.SkipChecks {
		skipChecks[name] = struct{}{}
	}

	return runRuntimeReportCommand(ctx, runCtx, logger, newDoctorInspector, runtimeReportCommandConfig{
		Wide:            opts.Wide,
		Timeout:         30 * time.Second,
		TimeoutMessage:  "Doctor runtime inspection timed out",
		CanceledMessage: "Doctor runtime inspection canceled",
	}, func(out *output.Output, runtimeReport status.StatusReport) int {
		loader := config.NewLoader().WithRepoRoot(runCtx.RepoRoot)
		diagnosticOpts := doctor.Options{
			RepoRoot:      runCtx.RepoRoot,
			Gateway:       loader.Gateway(true),
			Alerts:        loader.Alerts(true),
			RequiredPorts: config.RequiredPorts(),
			SkipChecks:    skipChecks,
			Fix:           opts.Fix,
			Wide:          opts.Wide,
			RuntimeReport: &runtimeReport,
		}

		doctorCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		report := doctor.Run(doctorCtx, doctor.DefaultChecks(), diagnosticOpts)

		if code := writeStructuredCommandOutput(runCtx.Stdout, runCtx.Stderr, opts.JSON, report.WriteJSON, func(w io.Writer) error {
			return writeDoctorHuman(w, report, opts.Wide)
		}); code != exitcodes.ACPExitSuccess {
			return code
		}
		if opts.Notify {
			if err := doctor.NotifyActionableFindings(doctorCtx, diagnosticOpts.Alerts, report); err != nil {
				fmt.Fprintf(runCtx.Stderr, "Error: notify doctor findings: %v\n", err)
				return exitcodes.ACPExitRuntime
			}
		}
		return doctor.ExitCodeForReport(report)
	})
}

func writeDoctorHuman(w io.Writer, report doctor.Report, wide bool) error {
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

		if wide && !result.Details.IsZero() {
			for _, line := range result.Details.Lines() {
				fmt.Fprintf(w, "                     %s\n", line)
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
