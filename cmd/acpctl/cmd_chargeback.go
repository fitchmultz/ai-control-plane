// cmd_chargeback.go - Typed chargeback rendering helpers.
//
// Purpose:
//   - Provide a native acpctl surface for safe chargeback serialization flows.
//
// Responsibilities:
//   - Parse chargeback render/payload subcommands and flags.
//   - Read structured chargeback inputs from environment variables.
//   - Emit canonical JSON, CSV, and webhook payloads to stdout.
//
// Non-scope:
//   - Does not query the database directly.
//   - Does not calculate report analytics.
//
// Invariants/Assumptions:
//   - Shell orchestrators provide the required environment variables.
//   - Output is machine-readable and deterministic for equivalent inputs.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/chargeback"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type chargebackReportOptions struct {
	ReportMonth          string
	Format               string
	ArchiveDir           string
	VarianceThreshold    float64
	AnomalyThreshold     float64
	ForecastEnabled      bool
	BudgetAlertThreshold float64
	Notify               bool
	Verbose              bool
}

type chargebackRenderOptions struct {
	Format string
}

type chargebackPayloadOptions struct {
	Target string
}

func chargebackCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "chargeback",
		Summary:     "Typed chargeback rendering helpers",
		Description: "Typed chargeback serialization helpers.",
		Examples: []string{
			"acpctl chargeback report",
			"acpctl chargeback render --format json",
			"acpctl chargeback render --format csv",
			"acpctl chargeback payload --target generic",
		},
		Children: []*commandSpec{
			{
				Name:        "report",
				Summary:     "Generate canonical chargeback report artifacts",
				Description: "Generate monthly chargeback artifacts from the typed database/report workflow.",
				Examples: []string{
					"acpctl chargeback report",
					"acpctl chargeback report --format all",
					"acpctl chargeback report --month 2026-02 --no-forecast",
				},
				Options: []commandOptionSpec{
					{Name: "month", ValueName: "YYYY-MM", Summary: "Target report month", Type: optionValueString},
					{Name: "format", ValueName: "FORMAT", Summary: "Output format: markdown, json, csv, or all", Type: optionValueString, DefaultText: "markdown"},
					{Name: "archive-dir", ValueName: "DIR", Summary: "Archive directory", Type: optionValueString, DefaultText: "demo/backups/chargeback"},
					{Name: "variance-threshold", ValueName: "FLOAT", Summary: "Variance threshold percent", Type: optionValueFloat, DefaultText: "15"},
					{Name: "anomaly-threshold", ValueName: "FLOAT", Summary: "Cost-center anomaly spike threshold percent", Type: optionValueFloat, DefaultText: "200"},
					{Name: "forecast", Summary: "Enable spend forecasting", Type: optionValueBool},
					{Name: "no-forecast", Summary: "Disable spend forecasting", Type: optionValueBool},
					{Name: "budget-alert-threshold", ValueName: "FLOAT", Summary: "Budget alert percent", Type: optionValueFloat, DefaultText: "80"},
					{Name: "notify", Summary: "Send configured webhook notifications", Type: optionValueBool},
					{Name: "verbose", Summary: "Print workflow progress to stderr", Type: optionValueBool},
				},
				Sections: []commandHelpSection{
					{
						Title: "Environment",
						Lines: []string{
							"ACP_DATABASE_MODE",
							"DATABASE_URL",
							"DB_NAME",
							"DB_USER",
							"GENERIC_WEBHOOK_URL",
							"SLACK_WEBHOOK_URL",
						},
					},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindChargebackReportOptions,
					NativeRun:  runChargebackReportCommand,
				},
			},
			{
				Name:        "render",
				Summary:     "Render canonical chargeback JSON or CSV",
				Description: "Render machine-safe chargeback outputs from environment-provided inputs.",
				Examples: []string{
					"acpctl chargeback render --format json",
					"acpctl chargeback render --format csv",
				},
				Options: []commandOptionSpec{
					{Name: "format", ValueName: "FORMAT", Summary: "Output format: json or csv", Type: optionValueString, Required: true},
				},
				Sections: []commandHelpSection{
					{
						Title: "Environment",
						Lines: []string{
							"CHARGEBACK_REPORT_MONTH",
							"CHARGEBACK_COST_CENTER_JSON",
							"CHARGEBACK_MODEL_JSON",
							"CHARGEBACK_ANOMALIES_JSON",
							"CHARGEBACK_GENERATED_AT",
							"CHARGEBACK_MONTH_START",
							"CHARGEBACK_MONTH_END",
							"CHARGEBACK_TOTAL_SPEND",
							"CHARGEBACK_TOTAL_REQUESTS",
							"CHARGEBACK_TOTAL_TOKENS",
							"CHARGEBACK_VARIANCE",
							"CHARGEBACK_PREV_MONTH_SPEND",
							"CHARGEBACK_FORECAST_VALUES",
							"CHARGEBACK_DAILY_BURN",
							"CHARGEBACK_DAYS_REMAINING",
							"CHARGEBACK_EXHAUSTION_DATE",
							"CHARGEBACK_TOTAL_BUDGET",
							"CHARGEBACK_BUDGET_RISK_LEVEL",
							"CHARGEBACK_BUDGET_RISK_PERCENT",
							"CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED",
							"CHARGEBACK_SCHEMA_VERSION",
							"CHARGEBACK_VARIANCE_THRESHOLD",
							"CHARGEBACK_ANOMALY_THRESHOLD",
							"CHARGEBACK_FORECAST_ENABLED",
						},
					},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindChargebackRenderOptions,
					NativeRun:  runChargebackRenderCommand,
				},
			},
			{
				Name:        "payload",
				Summary:     "Render canonical chargeback webhook payload JSON",
				Description: "Render webhook payload JSON from environment-provided inputs.",
				Examples: []string{
					"acpctl chargeback payload --target generic",
					"acpctl chargeback payload --target slack",
				},
				Options: []commandOptionSpec{
					{Name: "target", ValueName: "TARGET", Summary: "Payload target: generic or slack", Type: optionValueString, Required: true},
				},
				Sections: []commandHelpSection{
					{
						Title: "Environment",
						Lines: []string{
							"CHARGEBACK_REPORT_MONTH",
							"CHARGEBACK_TOTAL_SPEND",
							"CHARGEBACK_VARIANCE",
							"CHARGEBACK_ANOMALIES_JSON",
							"CHARGEBACK_PAYLOAD_EVENT",
							"CHARGEBACK_PAYLOAD_TIMESTAMP",
							"CHARGEBACK_SLACK_COLOR",
							"CHARGEBACK_SLACK_EPOCH",
						},
					},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindChargebackPayloadOptions,
					NativeRun:  runChargebackPayloadCommand,
				},
			},
		},
	}
}

func bindChargebackReportOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	opts := chargebackReportOptions{
		ReportMonth:          input.String("month"),
		Format:               stringDefault(input.String("format"), "markdown"),
		ArchiveDir:           stringDefault(input.String("archive-dir"), "demo/backups/chargeback"),
		VarianceThreshold:    floatValueDefault(input, "variance-threshold", 15),
		AnomalyThreshold:     floatValueDefault(input, "anomaly-threshold", 200),
		ForecastEnabled:      !input.Bool("no-forecast"),
		BudgetAlertThreshold: floatValueDefault(input, "budget-alert-threshold", 80),
		Notify:               input.Bool("notify"),
		Verbose:              input.Bool("verbose"),
	}
	if input.Bool("forecast") {
		opts.ForecastEnabled = true
	}
	switch opts.Format {
	case "markdown", "json", "csv", "all":
	default:
		return nil, fmt.Errorf("--format must be one of: markdown, json, csv, all")
	}
	return opts, nil
}

func bindChargebackRenderOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	format := input.String("format")
	switch format {
	case "json", "csv":
		return chargebackRenderOptions{Format: format}, nil
	default:
		return nil, fmt.Errorf("--format must be one of: json, csv")
	}
}

func bindChargebackPayloadOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	target := input.String("target")
	switch target {
	case "generic", "slack":
		return chargebackPayloadOptions{Target: target}, nil
	default:
		return nil, fmt.Errorf("--target must be one of: generic, slack")
	}
}

func runChargebackReportCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(chargebackReportOptions)
	client := db.NewClient(runCtx.RepoRoot)
	defer func() { _ = client.Close() }()
	if err := client.ConfigError(); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	if opts.Verbose {
		fmt.Fprintln(runCtx.Stderr, "INFO: Generating typed chargeback report")
	}
	result, err := chargeback.GenerateReport(ctx, chargeback.NewDBStore(client), chargeback.ReportOptions{
		ReportMonth:          opts.ReportMonth,
		Format:               opts.Format,
		ArchiveDir:           opts.ArchiveDir,
		VarianceThreshold:    opts.VarianceThreshold,
		AnomalyThreshold:     opts.AnomalyThreshold,
		ForecastEnabled:      opts.ForecastEnabled,
		BudgetAlertThreshold: opts.BudgetAlertThreshold,
		Notify:               opts.Notify,
		GenericWebhookURL:    config.NewLoader().String("GENERIC_WEBHOOK_URL"),
		SlackWebhookURL:      config.NewLoader().String("SLACK_WEBHOOK_URL"),
		RepoRoot:             runCtx.RepoRoot,
		Stdout:               runCtx.Stdout,
		Stderr:               runCtx.Stderr,
	})
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if err := chargeback.WriteSelectedOutput(result, chargeback.ReportOptions{Format: opts.Format, Stdout: runCtx.Stdout}); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stderr, "INFO: Archived artifacts under %s\n", result.Outputs.Archived["md"])
	}
	if result.Data.VarianceExceeded || result.Data.HasAnomalies {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackRenderCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(chargebackRenderOptions)
	loader := config.NewLoader()
	costCenters, err := chargeback.DecodeCostCenters(loader.String("CHARGEBACK_COST_CENTER_JSON"))
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: invalid CHARGEBACK_COST_CENTER_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	switch opts.Format {
	case "csv":
		if err := chargeback.RenderCSV(runCtx.Stdout, loader.String("CHARGEBACK_REPORT_MONTH"), costCenters); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: render chargeback csv: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	case "json":
	}

	models, err := chargeback.DecodeModels(loader.String("CHARGEBACK_MODEL_JSON"))
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: invalid CHARGEBACK_MODEL_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	anomalies, err := chargeback.DecodeAnomalies(loader.String("CHARGEBACK_ANOMALIES_JSON"))
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: invalid CHARGEBACK_ANOMALIES_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	input, err := readRenderInput(loader, costCenters, models, anomalies)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if err := chargeback.RenderJSON(runCtx.Stdout, input); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: render chargeback json: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackPayloadCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(chargebackPayloadOptions)
	loader := config.NewLoader()
	switch opts.Target {
	case "generic":
		anomalies, err := chargeback.DecodeAnomalies(loader.String("CHARGEBACK_ANOMALIES_JSON"))
		if err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: invalid CHARGEBACK_ANOMALIES_JSON: %v\n", err)
			return exitcodes.ACPExitUsage
		}
		payload, err := chargeback.BuildGenericWebhookPayload(chargeback.GenericWebhookInput{
			Event:       stringDefault(loader.String("CHARGEBACK_PAYLOAD_EVENT"), "chargeback_report_generated"),
			ReportMonth: loader.String("CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  floatEnv(loader, "CHARGEBACK_TOTAL_SPEND"),
			Variance:    loader.String("CHARGEBACK_VARIANCE"),
			Anomalies:   anomalies,
			Timestamp:   stringDefault(loader.String("CHARGEBACK_PAYLOAD_TIMESTAMP"), time.Now().UTC().Format(time.RFC3339)),
		})
		if err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: build generic payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if _, err := runCtx.Stdout.Write(payload); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: write generic payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	case "slack":
		payload, err := chargeback.BuildSlackWebhookPayload(chargeback.SlackWebhookInput{
			ReportMonth: loader.String("CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  floatEnv(loader, "CHARGEBACK_TOTAL_SPEND"),
			Variance:    loader.String("CHARGEBACK_VARIANCE"),
			Color:       stringDefault(loader.String("CHARGEBACK_SLACK_COLOR"), "good"),
			Epoch:       int64EnvDefault(loader, "CHARGEBACK_SLACK_EPOCH", time.Now().Unix()),
		})
		if err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: build slack payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if _, err := runCtx.Stdout.Write(payload); err != nil {
			fmt.Fprintf(runCtx.Stderr, "Error: write slack payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	}
	return exitcodes.ACPExitUsage
}

func readRenderInput(loader *config.Loader, costCenters []chargeback.CostCenterAllocation, models []chargeback.ModelAllocation, anomalies []chargeback.Anomaly) (chargeback.ReportInput, error) {
	month1, month2, month3 := loader.ChargebackForecast()
	return chargeback.ReportInput{
		SchemaVersion:      stringDefault(loader.String("CHARGEBACK_SCHEMA_VERSION"), "1.0.0"),
		GeneratedAt:        stringDefault(loader.String("CHARGEBACK_GENERATED_AT"), time.Now().UTC().Format(time.RFC3339)),
		ReportMonth:        loader.String("CHARGEBACK_REPORT_MONTH"),
		PeriodStart:        loader.String("CHARGEBACK_MONTH_START"),
		PeriodEnd:          loader.String("CHARGEBACK_MONTH_END"),
		TotalSpend:         floatEnv(loader, "CHARGEBACK_TOTAL_SPEND"),
		TotalRequests:      int64Env(loader, "CHARGEBACK_TOTAL_REQUESTS"),
		TotalTokens:        int64Env(loader, "CHARGEBACK_TOTAL_TOKENS"),
		CostCenters:        costCenters,
		Models:             models,
		Variance:           loader.String("CHARGEBACK_VARIANCE"),
		VarianceThreshold:  floatEnv(loader, "CHARGEBACK_VARIANCE_THRESHOLD"),
		PreviousMonthSpend: floatEnv(loader, "CHARGEBACK_PREV_MONTH_SPEND"),
		Anomalies:          anomalies,
		ForecastEnabled:    boolEnvDefault(loader, "CHARGEBACK_FORECAST_ENABLED", true),
		ForecastMonth1:     month1,
		ForecastMonth2:     month2,
		ForecastMonth3:     month3,
		DailyBurn:          floatEnv(loader, "CHARGEBACK_DAILY_BURN"),
		DaysRemaining:      loader.Int64Ptr("CHARGEBACK_DAYS_REMAINING"),
		ExhaustionDate:     stringDefault(loader.String("CHARGEBACK_EXHAUSTION_DATE"), "N/A"),
		TotalBudget:        floatEnv(loader, "CHARGEBACK_TOTAL_BUDGET"),
		BudgetRisk: chargeback.BudgetRisk{
			RiskLevel:         stringDefault(loader.String("CHARGEBACK_BUDGET_RISK_LEVEL"), "unknown"),
			BudgetPercent:     loader.Float64Ptr("CHARGEBACK_BUDGET_RISK_PERCENT"),
			ThresholdExceeded: boolEnvDefault(loader, "CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED", false),
		},
		AnomalyThreshold: floatEnv(loader, "CHARGEBACK_ANOMALY_THRESHOLD"),
	}, nil
}

func floatEnv(loader *config.Loader, key string) float64 {
	return floatEnvDefault(loader, key, 0)
}

func floatEnvDefault(loader *config.Loader, key string, fallback float64) float64 {
	raw := strings.TrimSpace(loader.String(key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

func int64Env(loader *config.Loader, key string) int64 {
	return int64EnvDefault(loader, key, 0)
}

func int64EnvDefault(loader *config.Loader, key string, fallback int64) int64 {
	raw := strings.TrimSpace(loader.String(key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func boolEnvDefault(loader *config.Loader, key string, fallback bool) bool {
	raw := strings.TrimSpace(loader.String(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func stringDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func floatValueDefault(input parsedCommandInput, name string, fallback float64) float64 {
	if input.String(name) == "" {
		return fallback
	}
	value, err := input.Float(name)
	if err != nil {
		return fallback
	}
	return value
}
