// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Render operator-facing onboarding exports and summaries from typed workflow
//	state.
//
// Responsibilities:
//   - Build the environment export block for each supported tool.
//   - Render workflow summary output with key redaction rules.
//   - Keep output formatting separate from coordinator logic.
//
// Scope:
//   - Human-readable onboarding output only.
//
// Usage:
//   - Called by Run after key generation and before config writes.
//
// Invariants/Assumptions:
//   - Exported keys are redacted unless ShowKey is explicitly enabled.
package onboard

import (
	"io"
	"strings"
)

func renderSummary(state runState) string {
	keyAlias := ""
	if state.KeyValue != "" {
		keyAlias = state.GeneratedAlias
	}

	var builder strings.Builder
	builder.WriteString("\n")
	builder.WriteString("Tool: " + state.Options.Tool + "\n")
	builder.WriteString("Mode: " + state.Options.Mode + "\n")
	builder.WriteString("Gateway: " + state.BaseURL + "\n")
	builder.WriteString("Model: " + state.Options.Model + "\n")
	if keyAlias != "" {
		builder.WriteString("Key alias: " + keyAlias + "\n")
	}
	builder.WriteString("\n")
	if state.Options.Mode == "subscription" && state.Options.Tool == "codex" {
		builder.WriteString("INFO: Run 'make chatgpt-login' on this gateway host before launching Codex.\n\n")
	}
	builder.WriteString(renderExports(state))
	builder.WriteString("\n")
	return builder.String()
}

func renderExports(state runState) string {
	keyValue := redactKey(state.KeyValue)
	if state.Options.ShowKey {
		keyValue = state.KeyValue
	}
	var builder strings.Builder
	if state.Options.Tool == "claude" {
		builder.WriteString(`export ANTHROPIC_BASE_URL="` + state.BaseURL + `"` + "\n")
		builder.WriteString(`export ANTHROPIC_API_KEY="` + keyValue + `"` + "\n")
		builder.WriteString(`export ANTHROPIC_MODEL="` + state.Options.Model + `"` + "\n")
		return builder.String()
	}
	if state.Options.Mode == "direct" {
		builder.WriteString(`export OTEL_EXPORTER_OTLP_ENDPOINT="http://` + state.Options.Host + `:4317"` + "\n")
		builder.WriteString(`export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"` + "\n")
		builder.WriteString(`export OTEL_SERVICE_NAME="codex-cli"` + "\n")
		return builder.String()
	}
	builder.WriteString(`export OPENAI_BASE_URL="` + state.BaseURL + `"` + "\n")
	builder.WriteString(`export OPENAI_API_KEY="` + keyValue + `"` + "\n")
	builder.WriteString(`export OPENAI_MODEL="` + state.Options.Model + `"` + "\n")
	return builder.String()
}

func redactKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func WriteHuman(stdout io.Writer, stderr io.Writer, result Result) {
	if strings.TrimSpace(result.Stdout) != "" {
		fprintf(stdout, "%s", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		fprintf(stderr, "%s", result.Stderr)
	}
}
