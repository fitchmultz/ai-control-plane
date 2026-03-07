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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/chargeback"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func runChargebackCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printChargebackHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	switch args[0] {
	case "help", "--help", "-h":
		printChargebackHelp(stdout)
		return exitcodes.ACPExitSuccess
	case "render":
		return runChargebackRenderCommand(ctx, args[1:], stdout, stderr)
	case "payload":
		return runChargebackPayloadCommand(ctx, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Error: Unknown chargeback subcommand: %s\n", args[0])
		printChargebackHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func printChargebackHelp(out *os.File) {
	command := mustLookupNativeCommand("chargeback")

	fmt.Fprint(out, `Usage: acpctl chargeback <subcommand> [options]

Typed chargeback serialization helpers.

Subcommands:
`)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(out, "  %-12s %s\n", subcommand.Name, subcommand.Description)
	}
	fmt.Fprint(out, `

Examples:
  acpctl chargeback render --format json
  acpctl chargeback render --format csv
  acpctl chargeback payload --target generic

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printChargebackRenderHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl chargeback render --format <json|csv>

Render machine-safe chargeback outputs from environment-provided inputs.

Environment:
  CHARGEBACK_REPORT_MONTH
  CHARGEBACK_COST_CENTER_JSON
  CHARGEBACK_MODEL_JSON
  CHARGEBACK_ANOMALIES_JSON
  CHARGEBACK_GENERATED_AT
  CHARGEBACK_MONTH_START
  CHARGEBACK_MONTH_END
  CHARGEBACK_TOTAL_SPEND
  CHARGEBACK_TOTAL_REQUESTS
  CHARGEBACK_TOTAL_TOKENS
  CHARGEBACK_VARIANCE
  CHARGEBACK_PREV_MONTH_SPEND
  CHARGEBACK_FORECAST_VALUES
  CHARGEBACK_DAILY_BURN
  CHARGEBACK_DAYS_REMAINING
  CHARGEBACK_EXHAUSTION_DATE
  CHARGEBACK_TOTAL_BUDGET
  CHARGEBACK_BUDGET_RISK_LEVEL
  CHARGEBACK_BUDGET_RISK_PERCENT
  CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED
  CHARGEBACK_SCHEMA_VERSION
  CHARGEBACK_VARIANCE_THRESHOLD
  CHARGEBACK_ANOMALY_THRESHOLD
  CHARGEBACK_FORECAST_ENABLED

Examples:
  acpctl chargeback render --format json
  acpctl chargeback render --format csv

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printChargebackPayloadHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl chargeback payload --target <generic|slack>

Render webhook payload JSON from environment-provided inputs.

Environment:
  CHARGEBACK_REPORT_MONTH
  CHARGEBACK_TOTAL_SPEND
  CHARGEBACK_VARIANCE
  CHARGEBACK_ANOMALIES_JSON
  CHARGEBACK_PAYLOAD_EVENT
  CHARGEBACK_PAYLOAD_TIMESTAMP
  CHARGEBACK_SLACK_COLOR
  CHARGEBACK_SLACK_EPOCH

Examples:
  acpctl chargeback payload --target generic
  acpctl chargeback payload --target slack

Exit codes:
  0   Success
  1   Domain non-success
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func runChargebackRenderCommand(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 1 && isHelpToken(args[0]) {
		printChargebackRenderHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	flags := flag.NewFlagSet("chargeback render", flag.ContinueOnError)
	flags.SetOutput(stderr)
	format := flags.String("format", "", "Output format: json or csv")
	if err := flags.Parse(args); err != nil {
		return exitcodes.ACPExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Error: unexpected argument(s): %s\n", strings.Join(flags.Args(), " "))
		printChargebackRenderHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	costCenters, err := chargeback.DecodeCostCenters(os.Getenv("CHARGEBACK_COST_CENTER_JSON"))
	if err != nil {
		fmt.Fprintf(stderr, "Error: invalid CHARGEBACK_COST_CENTER_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	switch *format {
	case "csv":
		if err := chargeback.RenderCSV(stdout, os.Getenv("CHARGEBACK_REPORT_MONTH"), costCenters); err != nil {
			fmt.Fprintf(stderr, "Error: render chargeback csv: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	case "json":
	default:
		fmt.Fprintln(stderr, "Error: --format must be one of: json, csv")
		printChargebackRenderHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	models, err := chargeback.DecodeModels(os.Getenv("CHARGEBACK_MODEL_JSON"))
	if err != nil {
		fmt.Fprintf(stderr, "Error: invalid CHARGEBACK_MODEL_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	anomalies, err := chargeback.DecodeAnomalies(os.Getenv("CHARGEBACK_ANOMALIES_JSON"))
	if err != nil {
		fmt.Fprintf(stderr, "Error: invalid CHARGEBACK_ANOMALIES_JSON: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	input, err := readRenderInput(costCenters, models, anomalies)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if err := chargeback.RenderJSON(stdout, input); err != nil {
		fmt.Fprintf(stderr, "Error: render chargeback json: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackPayloadCommand(_ context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 1 && isHelpToken(args[0]) {
		printChargebackPayloadHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	flags := flag.NewFlagSet("chargeback payload", flag.ContinueOnError)
	flags.SetOutput(stderr)
	target := flags.String("target", "", "Payload target: generic or slack")
	if err := flags.Parse(args); err != nil {
		return exitcodes.ACPExitUsage
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Error: unexpected argument(s): %s\n", strings.Join(flags.Args(), " "))
		printChargebackPayloadHelp(stderr)
		return exitcodes.ACPExitUsage
	}

	switch *target {
	case "generic":
		anomalies, err := chargeback.DecodeAnomalies(os.Getenv("CHARGEBACK_ANOMALIES_JSON"))
		if err != nil {
			fmt.Fprintf(stderr, "Error: invalid CHARGEBACK_ANOMALIES_JSON: %v\n", err)
			return exitcodes.ACPExitUsage
		}
		payload, err := chargeback.BuildGenericWebhookPayload(chargeback.GenericWebhookInput{
			Event:       stringDefault(os.Getenv("CHARGEBACK_PAYLOAD_EVENT"), "chargeback_report_generated"),
			ReportMonth: os.Getenv("CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  floatEnv("CHARGEBACK_TOTAL_SPEND"),
			Variance:    os.Getenv("CHARGEBACK_VARIANCE"),
			Anomalies:   anomalies,
			Timestamp:   stringDefault(os.Getenv("CHARGEBACK_PAYLOAD_TIMESTAMP"), time.Now().UTC().Format(time.RFC3339)),
		})
		if err != nil {
			fmt.Fprintf(stderr, "Error: build generic payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if _, err := stdout.Write(payload); err != nil {
			fmt.Fprintf(stderr, "Error: write generic payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	case "slack":
		payload, err := chargeback.BuildSlackWebhookPayload(chargeback.SlackWebhookInput{
			ReportMonth: os.Getenv("CHARGEBACK_REPORT_MONTH"),
			TotalSpend:  floatEnv("CHARGEBACK_TOTAL_SPEND"),
			Variance:    os.Getenv("CHARGEBACK_VARIANCE"),
			Color:       stringDefault(os.Getenv("CHARGEBACK_SLACK_COLOR"), "good"),
			Epoch:       int64EnvDefault("CHARGEBACK_SLACK_EPOCH", time.Now().Unix()),
		})
		if err != nil {
			fmt.Fprintf(stderr, "Error: build slack payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		if _, err := stdout.Write(payload); err != nil {
			fmt.Fprintf(stderr, "Error: write slack payload: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		return exitcodes.ACPExitSuccess
	default:
		fmt.Fprintln(stderr, "Error: --target must be one of: generic, slack")
		printChargebackPayloadHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func readRenderInput(costCenters []chargeback.CostCenterAllocation, models []chargeback.ModelAllocation, anomalies []chargeback.Anomaly) (chargeback.ReportInput, error) {
	month1, month2, month3 := forecastValues(os.Getenv("CHARGEBACK_FORECAST_VALUES"))
	return chargeback.ReportInput{
		SchemaVersion:      stringDefault(os.Getenv("CHARGEBACK_SCHEMA_VERSION"), "1.0.0"),
		GeneratedAt:        stringDefault(os.Getenv("CHARGEBACK_GENERATED_AT"), time.Now().UTC().Format(time.RFC3339)),
		ReportMonth:        os.Getenv("CHARGEBACK_REPORT_MONTH"),
		PeriodStart:        os.Getenv("CHARGEBACK_MONTH_START"),
		PeriodEnd:          os.Getenv("CHARGEBACK_MONTH_END"),
		TotalSpend:         floatEnv("CHARGEBACK_TOTAL_SPEND"),
		TotalRequests:      int64Env("CHARGEBACK_TOTAL_REQUESTS"),
		TotalTokens:        int64Env("CHARGEBACK_TOTAL_TOKENS"),
		CostCenters:        costCenters,
		Models:             models,
		Variance:           os.Getenv("CHARGEBACK_VARIANCE"),
		VarianceThreshold:  floatEnv("CHARGEBACK_VARIANCE_THRESHOLD"),
		PreviousMonthSpend: floatEnv("CHARGEBACK_PREV_MONTH_SPEND"),
		Anomalies:          anomalies,
		ForecastEnabled:    boolEnvDefault("CHARGEBACK_FORECAST_ENABLED", true),
		ForecastMonth1:     month1,
		ForecastMonth2:     month2,
		ForecastMonth3:     month3,
		DailyBurn:          floatEnv("CHARGEBACK_DAILY_BURN"),
		DaysRemaining:      nullableInt64Env("CHARGEBACK_DAYS_REMAINING"),
		ExhaustionDate:     stringDefault(os.Getenv("CHARGEBACK_EXHAUSTION_DATE"), "N/A"),
		TotalBudget:        floatEnv("CHARGEBACK_TOTAL_BUDGET"),
		BudgetRisk: chargeback.BudgetRisk{
			RiskLevel:         stringDefault(os.Getenv("CHARGEBACK_BUDGET_RISK_LEVEL"), "unknown"),
			BudgetPercent:     nullableFloatEnv("CHARGEBACK_BUDGET_RISK_PERCENT"),
			ThresholdExceeded: boolEnvDefault("CHARGEBACK_BUDGET_RISK_THRESHOLD_EXCEEDED", false),
		},
		AnomalyThreshold: floatEnv("CHARGEBACK_ANOMALY_THRESHOLD"),
	}, nil
}

func forecastValues(raw string) (*float64, *float64, *float64) {
	parts := strings.Split(stringDefault(raw, "N/A,N/A,N/A"), ",")
	for len(parts) < 3 {
		parts = append(parts, "N/A")
	}
	return nullableFloat(parts[0]), nullableFloat(parts[1]), nullableFloat(parts[2])
}

func floatEnv(key string) float64 {
	return floatEnvDefault(key, 0)
}

func floatEnvDefault(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

func int64Env(key string) int64 {
	return int64EnvDefault(key, 0)
}

func int64EnvDefault(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" || strings.EqualFold(raw, "N/A") {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func boolEnvDefault(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
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

func nullableFloatEnv(key string) *float64 {
	return nullableFloat(os.Getenv(key))
}

func nullableFloat(raw string) *float64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "N/A") || strings.EqualFold(trimmed, "null") {
		return nil
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &value
}

func nullableInt64Env(key string) *int64 {
	trimmed := strings.TrimSpace(os.Getenv(key))
	if trimmed == "" || strings.EqualFold(trimmed, "N/A") || strings.EqualFold(trimmed, "null") {
		return nil
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil
	}
	return &value
}

func stringDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
