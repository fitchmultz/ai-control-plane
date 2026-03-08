// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement credential validation diagnostics through the shared typed
//	gateway service so auth checks follow the same runtime model as status.
//
// Responsibilities:
//   - Verify the configured master key is present.
//   - Check whether the gateway authorizes that master key.
//
// Non-scope:
//   - Does not mutate gateway credentials or configuration.
//
// Invariants/Assumptions:
//   - Credential validation uses the same typed gateway probes as status.
//
// Scope:
//   - Credential diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package doctor

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type credentialsValidCheck struct{}

func (c credentialsValidCheck) ID() string { return "credentials_valid" }

func (c credentialsValidCheck) Run(ctx context.Context, opts Options) CheckResult {
	component, ok := runtimeComponent(opts, "gateway")
	if !ok {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnknown,
			Severity: SeverityRuntime,
			Message:  "Gateway runtime inspection did not produce a result",
		}
	}

	details := component.Details
	details.HTTPStatus = component.Details.ModelsHTTPStatus
	details.Authorized = component.Details.ModelsAuthorized

	if !component.Details.MasterKeyConfigured {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "LITELLM_MASTER_KEY not set",
			Details:  details,
			Suggestions: []string{
				"Run: make install",
				"Set LITELLM_MASTER_KEY environment variable",
			},
		}
	}

	if component.Details.Error != "" && !component.Details.ModelsReachable {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  "Gateway unreachable; cannot validate credentials",
			Details:  details,
			Suggestions: []string{
				"Ensure services are running: make up",
				"Check network connectivity",
			},
		}
	}

	if component.Details.ModelsAuthorized && component.Details.ModelsHTTPStatus == 200 {
		details.AuthStatus = "authorized"
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelHealthy,
			Severity: SeverityDomain,
			Message:  "Master key is valid",
			Details:  details,
		}
	}

	if !component.Details.ModelsAuthorized {
		details.AuthStatus = "unauthorized"
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "Master key is invalid or placeholder",
			Details:  details,
			Suggestions: []string{
				"Regenerate master key: make key-gen-master",
				"Check demo/.env for correct key",
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Credentials Valid",
		Level:    status.HealthLevelWarning,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("Unexpected response: %d", component.Details.ModelsHTTPStatus),
		Details:  details,
		Suggestions: []string{
			"Check gateway status: make health",
		},
	}
}

func (c credentialsValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
