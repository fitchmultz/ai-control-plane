// cmd_chargeback.go - Chargeback command specification.
//
// Purpose:
//   - Declare the acpctl chargeback command tree and help text.
//
// Responsibilities:
//   - Describe chargeback report, render, and payload subcommands.
//   - Bind each subcommand to a thin CLI adapter.
//
// Non-scope:
//   - Does not decode environment variables or execute workflows directly.
//   - Does not own database composition or rendering logic.
//
// Invariants/Assumptions:
//   - Chargeback business logic lives under internal/chargeback.
//   - Backend handlers keep this file declarative.
//
// Scope:
//   - Chargeback command-spec wiring only.
//
// Usage:
//   - Used through the root command registry.
package main

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
