// Package config centralizes runtime configuration loading for ACP processes.
//
// Purpose:
//   - Own typed alert-delivery configuration for operator workflows.
//
// Responsibilities:
//   - Resolve webhook destinations for doctor notification delivery.
//   - Preserve internal/config ownership for env and repo-local `.env` access.
//
// Scope:
//   - Alert destination loading only.
//
// Usage:
//   - Call `Loader.Alerts()` from typed workflows that emit notifications.
//
// Invariants/Assumptions:
//   - Process environment takes precedence over repo-local fallback values.
package config

import "github.com/mitchfultz/ai-control-plane/internal/textutil"

// AlertSettings captures configured notification destinations for doctor flows.
type AlertSettings struct {
	GenericWebhookURL string
	SlackWebhookURL   string
}

// Alerts resolves alert-delivery settings, optionally allowing repo-local fallback.
func (l *Loader) Alerts(includeRepoFallback bool) AlertSettings {
	resolve := l.String
	if includeRepoFallback {
		resolve = l.RepoAwareString
	}

	return AlertSettings{
		GenericWebhookURL: textutil.FirstNonBlank(
			resolve("ACP_ALERT_GENERIC_WEBHOOK_URL"),
			resolve("GENERIC_WEBHOOK_URL"),
		),
		SlackWebhookURL: textutil.FirstNonBlank(
			resolve("ACP_ALERT_SLACK_WEBHOOK_URL"),
			resolve("SLACK_WEBHOOK_URL"),
		),
	}
}
