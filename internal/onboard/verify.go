// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Perform gateway and collector verification for onboarding flows while
//	preserving caller cancellation.
//
// Responsibilities:
//   - Probe subscription and API-key routes.
//   - Validate direct-mode OTEL collector health.
//   - Keep HTTP verification isolated from workflow coordination.
//
// Scope:
//   - Read-only verification logic only.
//
// Usage:
//   - Called by Run when --verify is enabled.
//
// Invariants/Assumptions:
//   - HTTP requests honor the caller-provided context.
//   - Authorized checks include bearer auth only when a generated key exists.
package onboard

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func verifyOnboarding(ctx context.Context, state runState) error {
	if !state.Options.Verify {
		return nil
	}
	if state.Options.Mode == "direct" {
		code, err := probeStatus(ctx, fmt.Sprintf("http://%s:4318/health", state.Options.Host), "", state.Options.HTTPClient)
		if err != nil || code != http.StatusOK {
			return fmt.Errorf("OTEL collector check returned HTTP %d at http://%s:4318/health", code, state.Options.Host)
		}
		fprintf(state.Options.Stdout, "INFO: OTEL collector health check passed\n")
		return nil
	}

	healthCode, healthErr := probeStatus(ctx, state.BaseURL+"/health", "", state.Options.HTTPClient)
	if healthErr != nil || (healthCode != http.StatusOK && healthCode != http.StatusUnauthorized) {
		return fmt.Errorf("gateway /health returned HTTP %d", healthCode)
	}
	modelCode, modelErr := probeStatus(ctx, state.BaseURL+"/v1/models", state.KeyValue, state.Options.HTTPClient)
	if modelErr != nil || modelCode != http.StatusOK {
		return fmt.Errorf("authorized /v1/models check returned HTTP %d", modelCode)
	}
	fprintf(state.Options.Stdout, "INFO: Gateway health and authorized model checks passed\n")
	return nil
}

func verifySubscriptionPrereq(ctx context.Context, state runState) error {
	if state.Options.Mode != "subscription" || state.Options.Tool != "codex" {
		return nil
	}
	healthCode, err := probeStatus(ctx, state.BaseURL+"/health", "", state.Options.HTTPClient)
	if err != nil || (healthCode != http.StatusOK && healthCode != http.StatusUnauthorized) {
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
