// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Build structured onboarding verification and linting results while
//	preserving caller cancellation.
//
// Responsibilities:
//   - Validate the emitted tool env/config contract.
//   - Validate ACP-managed tool config writes.
//   - Probe network reachability for the selected onboarding mode.
//   - Verify generated gateway keys can authorize model discovery when applicable.
//
// Scope:
//   - Read-only verification logic only.
//
// Usage:
//   - Called by Run after key generation and any ACP-managed config writes.
//
// Invariants/Assumptions:
//   - HTTP requests honor the caller-provided context.
//   - Local lint always runs even when network verification is disabled.
package onboard

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

func verifyOnboarding(ctx context.Context, state runState) VerificationReport {
	checks := []VerificationCheck{
		validateEmittedConfigContract(state),
		validateToolConfigWrites(state.ToolConfig),
	}

	if state.Options.Verify {
		checks = append(checks,
			verifyGatewayReachability(ctx, state),
			verifyAuthorizedModelPath(ctx, state),
		)
	} else {
		checks = append(checks,
			skippedCheck("gateway reachability", "network verification disabled by operator"),
			skippedCheck("authorized model path", "network verification disabled by operator"),
		)
	}

	issues := validation.NewIssues()
	for _, check := range checks {
		issues.Extend(check.Issues)
	}
	return VerificationReport{Checks: checks, Issues: issues.Sorted()}
}

func validateEmittedConfigContract(state runState) VerificationCheck {
	issues := validation.NewIssues()
	rendered := renderExports(state)

	if strings.TrimSpace(rendered) == "" {
		issues.Add("rendered exports are empty; rerun `acpctl onboard`")
	}

	if state.Options.Mode == "direct" {
		expectedEndpoint := fmt.Sprintf("export OTEL_EXPORTER_OTLP_ENDPOINT=%q", otelGRPCEndpoint(state.Options.Host))
		if !strings.Contains(rendered, expectedEndpoint) {
			issues.Addf("rendered exports are missing %s", expectedEndpoint)
		}
		if !strings.Contains(rendered, `export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"`) {
			issues.Add(`rendered exports are missing export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"`)
		}
		if !strings.Contains(rendered, `export OTEL_SERVICE_NAME="codex-cli"`) {
			issues.Add(`rendered exports are missing export OTEL_SERVICE_NAME="codex-cli"`)
		}
		if strings.Contains(rendered, "export GATEWAY_URL=") {
			issues.Add("direct mode must not emit GATEWAY_URL")
		}
		if strings.Contains(rendered, "export OPENAI_API_KEY=") || strings.Contains(rendered, "export ANTHROPIC_API_KEY=") {
			issues.Add("direct mode must not emit routed API-key exports")
		}
		if err := validateAbsoluteURL(otelGRPCEndpoint(state.Options.Host)); err != nil {
			issues.Addf("generated OTEL endpoint %q is invalid: %v", otelGRPCEndpoint(state.Options.Host), err)
		}
		if err := validateAbsoluteURL(otelHealthURL(state.Options.Host)); err != nil {
			issues.Addf("generated OTEL health URL %q is invalid: %v", otelHealthURL(state.Options.Host), err)
		}
		if issues.Len() == 0 {
			return passCheck("env/config contract", "generated OTEL exports are valid for direct mode")
		}
		return failCheck(
			"env/config contract",
			issues.Sorted(),
			"Rerun `acpctl onboard` and provide a valid collector host.",
		)
	}

	expectedGatewayURL := fmt.Sprintf("export GATEWAY_URL=%q", state.Gateway.BaseURL)
	if !strings.Contains(rendered, expectedGatewayURL) {
		issues.Addf("rendered exports are missing %s", expectedGatewayURL)
	}
	if strings.Contains(rendered, "export OTEL_EXPORTER_OTLP_ENDPOINT=") {
		issues.Add("gateway-routed modes must not emit OTEL direct-mode exports")
	}
	if err := validateAbsoluteURL(state.Gateway.BaseURL); err != nil {
		issues.Addf("generated GATEWAY_URL %q is invalid: %v", state.Gateway.BaseURL, err)
	}
	if strings.TrimSpace(state.KeyValue) == "" {
		issues.Add("generated API key is empty; rerun `acpctl onboard` to create a fresh key")
	}
	if strings.TrimSpace(state.Options.Model) == "" {
		issues.Add("model alias is empty; rerun `acpctl onboard` and choose a model")
	}

	switch state.Options.Tool {
	case "claude":
		if !strings.Contains(rendered, `export ANTHROPIC_BASE_URL="$GATEWAY_URL"`) {
			issues.Add(`rendered exports are missing export ANTHROPIC_BASE_URL="$GATEWAY_URL"`)
		}
		expectedModel := fmt.Sprintf("export ANTHROPIC_MODEL=%q", state.Options.Model)
		if !strings.Contains(rendered, expectedModel) {
			issues.Addf("rendered exports are missing %s", expectedModel)
		}
		if state.Options.Mode == "subscription" {
			if !strings.Contains(rendered, "export ANTHROPIC_CUSTOM_HEADERS=") {
				issues.Add("rendered exports are missing ANTHROPIC_CUSTOM_HEADERS for Claude subscription mode")
			}
		} else if !strings.Contains(rendered, "export ANTHROPIC_API_KEY=") {
			issues.Add("rendered exports are missing ANTHROPIC_API_KEY")
		}
	default:
		if !strings.Contains(rendered, `export OPENAI_BASE_URL="$GATEWAY_URL"`) {
			issues.Add(`rendered exports are missing export OPENAI_BASE_URL="$GATEWAY_URL"`)
		}
		if !strings.Contains(rendered, "export OPENAI_API_KEY=") {
			issues.Add("rendered exports are missing OPENAI_API_KEY")
		}
		expectedModel := fmt.Sprintf("export OPENAI_MODEL=%q", state.Options.Model)
		if !strings.Contains(rendered, expectedModel) {
			issues.Addf("rendered exports are missing %s", expectedModel)
		}
	}

	if issues.Len() == 0 {
		return passCheck("env/config contract", "rendered exports form a valid onboarding contract")
	}
	return failCheck(
		"env/config contract",
		issues.Sorted(),
		"Rerun `acpctl onboard` and choose a valid host, port, and model.",
		"If you manually edited the emitted settings, discard them and use the values from this run.",
	)
}

func verifyGatewayReachability(ctx context.Context, state runState) VerificationCheck {
	issues := validation.NewIssues()
	remediation := []string{
		"Confirm the runtime is healthy with `make health`.",
	}

	if state.Options.Mode == "direct" {
		collectorHealth := otelHealthURL(state.Options.Host)
		code, err := probeStatus(ctx, collectorHealth, "", state.Options.HTTPClient)
		if err != nil {
			issues.Addf("OTEL collector health endpoint %s is unreachable: %v", collectorHealth, err)
		} else if code != http.StatusOK {
			issues.Addf("OTEL collector health endpoint %s returned HTTP %d", collectorHealth, code)
		}
		if issues.Len() == 0 {
			return passCheck("gateway reachability", fmt.Sprintf("OTEL collector reachable at %s", collectorHealth))
		}
		return failCheck(
			"gateway reachability",
			issues.Sorted(),
			append(remediation,
				"Start or repair the local collector with `make up-production` when direct telemetry is required.",
			)...,
		)
	}

	healthURL := state.Gateway.BaseURL + "/health"
	healthCode, err := probeStatus(ctx, healthURL, "", state.Options.HTTPClient)
	if err != nil {
		issues.Addf("gateway %s is unreachable: %v", state.Gateway.BaseURL, err)
	} else if healthCode != http.StatusOK && healthCode != http.StatusUnauthorized && healthCode != http.StatusForbidden {
		issues.Addf("gateway /health returned HTTP %d at %s", healthCode, healthURL)
	}
	if issues.Len() == 0 {
		return passCheck("gateway reachability", fmt.Sprintf("gateway reachable at %s", healthURL))
	}
	return failCheck(
		"gateway reachability",
		issues.Sorted(),
		append(remediation,
			"Start or repair the local stack with `make up` if this should be a local gateway.",
			"If you are targeting a remote gateway, confirm GATEWAY_URL points at the routable ACP endpoint from this machine.",
		)...,
	)
}

func verifyAuthorizedModelPath(ctx context.Context, state runState) VerificationCheck {
	if state.Options.Mode == "direct" {
		return skippedCheck("authorized model path", "direct mode does not generate a gateway key or call /v1/models")
	}

	client := gateway.NewClient(
		gateway.WithBaseURL(state.Gateway.BaseURL),
		gateway.WithMasterKey(state.KeyValue),
		gateway.WithHTTPClient(state.Options.HTTPClient),
	)
	ok, code, err := client.Models(ctx)
	if err == nil && ok && code == http.StatusOK {
		return passCheck("authorized model path", "generated key successfully authorized GET /v1/models")
	}

	issues := validation.NewIssues()
	remediation := []string{
		"Confirm the gateway is healthy with `make health`.",
		"Rerun `acpctl onboard` to mint a fresh key and retry verification.",
	}
	if err != nil {
		issues.Addf("authorized GET /v1/models failed against %s: %v", state.Gateway.BaseURL, err)
	} else {
		issues.Addf("authorized GET /v1/models returned HTTP %d against %s", code, state.Gateway.BaseURL)
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			remediation = append(remediation, "Inspect gateway auth or policy configuration and confirm generated keys are allowed to access `/v1/models`.")
		}
	}

	return failCheck("authorized model path", issues.Sorted(), remediation...)
}

func validateToolConfigWrites(result ToolConfigResult) VerificationCheck {
	if result.Skipped {
		return skippedCheck("tool config writes", result.Summary)
	}
	if !result.HasIssues() {
		return passCheck("tool config writes", result.Summary)
	}
	return failCheck(
		"tool config writes",
		result.Issues,
		"Resolve the tool-config issue above, then rerun `acpctl onboard`.",
	)
}

func passCheck(name string, summary string) VerificationCheck {
	return VerificationCheck{Name: name, Status: VerificationStatusPass, Summary: summary}
}

func skippedCheck(name string, summary string) VerificationCheck {
	return VerificationCheck{Name: name, Status: VerificationStatusSkip, Summary: summary}
}

func failCheck(name string, issues []string, remediation ...string) VerificationCheck {
	summary := "verification failed"
	if len(issues) > 0 {
		summary = issues[0]
	}
	return VerificationCheck{
		Name:        name,
		Status:      VerificationStatusFail,
		Summary:     summary,
		Issues:      append([]string(nil), issues...),
		Remediation: append([]string(nil), remediation...),
	}
}

func validateAbsoluteURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must include scheme and host")
	}
	return nil
}

func verifySubscriptionPrereq(ctx context.Context, state runState) error {
	if state.Options.Mode != "subscription" || state.Options.Tool != "codex" {
		return nil
	}
	healthCode, err := probeStatus(ctx, state.Gateway.BaseURL+"/health", "", state.Options.HTTPClient)
	if err != nil || (healthCode != http.StatusOK && healthCode != http.StatusUnauthorized && healthCode != http.StatusForbidden) {
		return fmt.Errorf("gateway health is not ready for subscription mode (HTTP %d). Complete ChatGPT device login first: make chatgpt-login", healthCode)
	}
	return nil
}

func probeStatus(ctx context.Context, url string, key string, client *http.Client) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(key) != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
