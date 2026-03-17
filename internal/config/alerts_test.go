// alerts_test.go - Coverage for alert configuration helpers.
//
// Purpose:
//   - Verify alert destination resolution stays deterministic.
//
// Responsibilities:
//   - Cover process-env precedence and repo-local fallback aliases.
//   - Keep doctor notification config under internal/config ownership.
//
// Scope:
//   - Alert config loading only.
//
// Usage:
//   - Run via `go test ./internal/config`.
//
// Invariants/Assumptions:
//   - Tests avoid host-specific environment state.
package config

import "testing"

func TestLoaderAlertsUsesProcessEnvThenRepoFallback(t *testing.T) {
	loader := NewTestLoader(map[string]string{
		"ACP_ALERT_GENERIC_WEBHOOK_URL": "https://process.example/generic",
	}, "/repo", map[string]string{
		"GENERIC_WEBHOOK_URL": "https://repo.example/generic",
		"SLACK_WEBHOOK_URL":   "https://repo.example/slack",
	})

	settings := loader.Alerts(true)
	if settings.GenericWebhookURL != "https://process.example/generic" {
		t.Fatalf("GenericWebhookURL = %q", settings.GenericWebhookURL)
	}
	if settings.SlackWebhookURL != "https://repo.example/slack" {
		t.Fatalf("SlackWebhookURL = %q", settings.SlackWebhookURL)
	}
}

func TestLoaderAlertsCanIgnoreRepoFallback(t *testing.T) {
	loader := NewTestLoader(nil, "/repo", map[string]string{
		"GENERIC_WEBHOOK_URL": "https://repo.example/generic",
		"SLACK_WEBHOOK_URL":   "https://repo.example/slack",
	})

	settings := loader.Alerts(false)
	if settings.GenericWebhookURL != "" || settings.SlackWebhookURL != "" {
		t.Fatalf("expected empty process-only alert settings, got %+v", settings)
	}
}
