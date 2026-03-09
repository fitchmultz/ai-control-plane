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
		return runtimeInspectionMissing(c.ID(), "Credentials Valid", "Gateway")
	}

	details := component.Details
	details.HTTPStatus = component.Details.ModelsHTTPStatus
	details.Authorized = component.Details.ModelsAuthorized

	if !component.Details.MasterKeyConfigured {
		return withCheckDetails(
			newCheckResult(c.ID(), "Credentials Valid", status.HealthLevelUnhealthy, SeverityPrereq, "LITELLM_MASTER_KEY not set"),
			details,
			"Run: make install",
			"Set LITELLM_MASTER_KEY environment variable",
		)
	}

	if component.Details.Error != "" && !component.Details.ModelsReachable {
		return withCheckDetails(
			newCheckResult(c.ID(), "Credentials Valid", status.HealthLevelWarning, SeverityDomain, "Gateway unreachable; cannot validate credentials"),
			details,
			"Ensure services are running: make up",
			"Check network connectivity",
		)
	}

	if component.Details.ModelsAuthorized && component.Details.ModelsHTTPStatus == 200 {
		details.AuthStatus = "authorized"
		return withCheckDetails(
			newCheckResult(c.ID(), "Credentials Valid", status.HealthLevelHealthy, SeverityDomain, "Master key is valid"),
			details,
		)
	}

	if !component.Details.ModelsAuthorized {
		details.AuthStatus = "unauthorized"
		return withCheckDetails(
			newCheckResult(c.ID(), "Credentials Valid", status.HealthLevelUnhealthy, SeverityDomain, "Master key is invalid or placeholder"),
			details,
			"Regenerate master key: make key-gen-master",
			"Check demo/.env for correct key",
		)
	}

	return withCheckDetails(
		newCheckResult(c.ID(), "Credentials Valid", status.HealthLevelWarning, SeverityDomain, fmt.Sprintf("Unexpected response: %d", component.Details.ModelsHTTPStatus)),
		details,
		"Check gateway status: make health",
	)
}

func (c credentialsValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return noopFix(ctx, opts)
}
