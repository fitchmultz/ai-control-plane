// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Render operator-facing onboarding exports and summaries from typed workflow
//	state.
//
// Responsibilities:
//   - Build the environment export block for each supported tool and mode.
//   - Render workflow summary output with key redaction and verification rules.
//   - Keep output formatting separate from coordinator logic.
//
// Scope:
//   - Human-readable onboarding output only.
//
// Usage:
//   - Called by Run after key generation, linting, and verification.
//
// Invariants/Assumptions:
//   - The summary always redacts secrets.
//   - Full key reveal, when requested, is emitted in a separate one-time block.
package onboard

import (
	"fmt"
	"io"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func renderSummary(state runState) string {
	var builder strings.Builder
	builder.WriteString("\nConfigured:\n")
	builder.WriteString("  Tool: " + state.Options.Tool + "\n")
	builder.WriteString("  Mode: " + state.Options.Mode + "\n")
	if state.Options.Mode == "direct" {
		builder.WriteString("  Collector: " + otelGRPCEndpoint(state.Options.Host) + "\n")
	} else {
		builder.WriteString("  Gateway: " + state.Gateway.BaseURL + "\n")
	}
	if strings.TrimSpace(state.Options.Model) != "" {
		builder.WriteString("  Model: " + state.Options.Model + "\n")
	}
	if state.GeneratedAlias != "" {
		builder.WriteString("  Key alias: " + state.GeneratedAlias + "\n")
	}
	builder.WriteString("\nSettings:\n")
	builder.WriteString(renderExports(state))
	if len(state.Verification.Checks) > 0 {
		builder.WriteString("\nVerification:\n")
		builder.WriteString(renderVerificationSummary(state.Verification))
	}
	builder.WriteString("\nNext steps:\n")
	builder.WriteString(renderNextSteps(state))
	builder.WriteString("\n")
	return builder.String()
}

func renderExports(state runState) string {
	var builder strings.Builder
	if state.Options.Mode == "direct" {
		builder.WriteString(fmt.Sprintf("export OTEL_EXPORTER_OTLP_ENDPOINT=%q\n", otelGRPCEndpoint(state.Options.Host)))
		builder.WriteString("export OTEL_EXPORTER_OTLP_PROTOCOL=\"grpc\"\n")
		builder.WriteString("export OTEL_SERVICE_NAME=\"codex-cli\"\n")
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("export GATEWAY_URL=%q\n", state.Gateway.BaseURL))
	if state.Options.Tool == "claude" {
		builder.WriteString("export ANTHROPIC_BASE_URL=\"$GATEWAY_URL\"\n")
		builder.WriteString(fmt.Sprintf("export ANTHROPIC_MODEL=%q\n", state.Options.Model))
		if state.Options.Mode == "subscription" {
			builder.WriteString(fmt.Sprintf("export ANTHROPIC_CUSTOM_HEADERS=%q\n", "x-litellm-api-key: Bearer "+redactKey(state.KeyValue)))
		} else {
			builder.WriteString(fmt.Sprintf("export ANTHROPIC_API_KEY=%q\n", redactKey(state.KeyValue)))
		}
		return builder.String()
	}

	builder.WriteString("export OPENAI_BASE_URL=\"$GATEWAY_URL\"\n")
	builder.WriteString(fmt.Sprintf("export OPENAI_API_KEY=%q\n", redactKey(state.KeyValue)))
	builder.WriteString(fmt.Sprintf("export OPENAI_MODEL=%q\n", state.Options.Model))
	return builder.String()
}

func renderFullKeyReveal(state runState) string {
	if !state.Options.ShowKey || state.KeyValue == "" {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("Full key (shown once):\n")
	switch state.Options.Tool {
	case "claude":
		if state.Options.Mode == "subscription" {
			builder.WriteString(fmt.Sprintf("export ANTHROPIC_CUSTOM_HEADERS=%q\n", "x-litellm-api-key: Bearer "+state.KeyValue))
		} else {
			builder.WriteString(fmt.Sprintf("export ANTHROPIC_API_KEY=%q\n", state.KeyValue))
		}
	case "codex", "opencode", "cursor":
		if state.Options.Mode != "direct" {
			builder.WriteString(fmt.Sprintf("export OPENAI_API_KEY=%q\n", state.KeyValue))
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func renderVerificationSummary(report VerificationReport) string {
	var builder strings.Builder
	for _, check := range report.Checks {
		builder.WriteString(fmt.Sprintf("  %s %s: %s\n", verificationPrefix(check.Status), check.Name, check.Summary))
		for _, issue := range check.Issues {
			builder.WriteString(fmt.Sprintf("      Issue: %s\n", issue))
		}
		for _, fix := range check.Remediation {
			builder.WriteString(fmt.Sprintf("      Remedy: %s\n", fix))
		}
	}
	return builder.String()
}

func verificationPrefix(status VerificationStatus) string {
	switch status {
	case VerificationStatusPass:
		return "[OK]"
	case VerificationStatusSkip:
		return "[SKIP]"
	default:
		return "[FAIL]"
	}
}

func renderNextSteps(state runState) string {
	steps := make([]string, 0, 4)
	if state.Verification.HasFailures() {
		steps = append(steps, "Resolve every [FAIL] item in the verification section before treating onboarding as complete.")
	}

	switch state.Options.Tool {
	case "codex":
		if state.Options.Mode == "subscription" {
			steps = append(steps, "Run `make chatgpt-login` on the gateway host if you have not already.")
		}
		if state.Options.Mode == "direct" {
			steps = append(steps, "Launch Codex with the OTEL settings above.")
			steps = append(steps, "Confirm telemetry reaches the collector before treating visibility as working.")
			return formatSteps(steps)
		}
		if state.ToolConfig.HasIssues() {
			steps = append(steps, "Resolve the Codex config issue listed above before launching Codex.")
		} else if state.Options.WriteConfig && !state.ToolConfig.Skipped {
			if state.ToolConfig.Written {
				steps = append(steps, "Launch Codex; ACP-managed config was written and linted.")
			} else {
				steps = append(steps, "Launch Codex; ACP-managed config was already present and linted successfully.")
			}
		} else {
			steps = append(steps, "Export the variables above in the shell that launches Codex.")
		}
	case "claude":
		if state.Options.Mode == "subscription" {
			steps = append(steps, "Export the Claude variables above before launching Claude Code.")
			steps = append(steps, "In Claude Code, sign in with your Claude subscription when prompted.")
		} else {
			steps = append(steps, "Export the Claude variables above in the shell that launches Claude Code.")
		}
	case "opencode":
		steps = append(steps, "Export the OpenAI-compatible variables above before launching OpenCode.")
	case "cursor":
		steps = append(steps, "Paste the base URL, API key, and model above into Cursor's custom provider settings.")
		steps = append(steps, "Restart Cursor if it cached the old provider configuration.")
	default:
		steps = append(steps, "Apply the settings above to your tool.")
	}
	if !state.Options.Verify {
		steps = append(steps, "Optional: rerun `acpctl onboard` and answer yes to verification if you want a live connectivity check.")
	}
	return formatSteps(steps)
}

func formatSteps(steps []string) string {
	var builder strings.Builder
	for index, step := range steps {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", index+1, step))
	}
	return builder.String()
}

func redactKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func otelGRPCEndpoint(host string) string {
	return "http://" + firstNonBlank(strings.TrimSpace(host), config.DefaultGatewayHost) + ":4317"
}

func otelHealthURL(host string) string {
	return "http://" + firstNonBlank(strings.TrimSpace(host), config.DefaultGatewayHost) + ":4318/health"
}

func WriteHuman(stdout io.Writer, stderr io.Writer, result Result) {
	if strings.TrimSpace(result.Stdout) != "" {
		fprintf(stdout, "%s", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		fprintf(stderr, "%s", result.Stderr)
	}
}
