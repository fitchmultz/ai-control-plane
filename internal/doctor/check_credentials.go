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
	client := doctorGatewayClient(opts)
	state := client.Status(ctx)
	details := status.ComponentDetails{
		BaseURL:             state.BaseURL,
		HTTPStatus:          state.Models.HTTPStatus,
		MasterKeyConfigured: state.MasterKeyConfigured,
		Authorized:          state.Models.Authorized,
		Error:               state.Models.Error,
	}

	if !state.MasterKeyConfigured {
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

	if state.Models.Error != "" {
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

	if state.Models.Healthy {
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

	if !state.Models.Authorized {
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
		Message:  fmt.Sprintf("Unexpected response: %d", state.Models.HTTPStatus),
		Details:  details,
		Suggestions: []string{
			"Check gateway status: make health",
		},
	}
}

func (c credentialsValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
