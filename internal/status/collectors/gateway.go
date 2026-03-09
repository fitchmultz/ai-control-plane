// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose a gateway status collector that consumes the shared typed gateway
//	service instead of implementing bespoke HTTP probe logic.
//
// Responsibilities:
//   - Convert typed gateway probe results into status.ComponentStatus.
//   - Preserve operator-facing suggestions for common gateway failures.
//
// Non-scope:
//   - Does not perform bespoke HTTP request construction outside the gateway service.
//
// Invariants/Assumptions:
//   - Gateway probe results come from the shared typed gateway service.
//
// Scope:
//   - Gateway status collection only.
//
// Usage:
//   - Construct with NewGatewayCollector(client) and call Collect(ctx).
package collectors

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// GatewayCollector checks LiteLLM gateway health.
type GatewayCollector struct {
	reader gateway.StatusReader
}

// NewGatewayCollector creates a typed gateway collector.
func NewGatewayCollector(reader gateway.StatusReader) GatewayCollector {
	return GatewayCollector{reader: reader}
}

// Name returns the collector's domain name.
func (c GatewayCollector) Name() string {
	return "gateway"
}

// Collect gathers status information from the LiteLLM gateway.
func (c GatewayCollector) Collect(ctx context.Context) status.ComponentStatus {
	state := c.reader.Status(ctx)
	details := status.ComponentDetails{
		Scheme:              state.Scheme,
		BaseURL:             state.BaseURL,
		TLSEnabled:          state.TLSEnabled,
		MasterKeyConfigured: state.MasterKeyConfigured,
		HTTPStatus:          state.Health.HTTPStatus,
		ModelsHTTPStatus:    state.Models.HTTPStatus,
		HealthReachable:     state.Health.Reachable,
		ModelsReachable:     state.Models.Reachable,
		HealthAuthorized:    state.Health.Authorized,
		ModelsAuthorized:    state.Models.Authorized,
		Reachable:           state.Health.Reachable || state.Models.Reachable,
		Authorized:          state.Health.Authorized && state.Models.Authorized,
	}

	if !state.MasterKeyConfigured {
		return componentStatus(c.Name(), status.HealthLevelWarning, "LITELLM_MASTER_KEY not set; authorized gateway checks skipped", details,
			"Set LITELLM_MASTER_KEY in demo/.env or your shell environment",
			"Re-run: make health",
		)
	}

	if state.Health.Error != "" {
		return componentStatus(c.Name(), status.HealthLevelUnhealthy, fmt.Sprintf("Gateway unreachable: %s", state.Health.Error), withDetailError(details, fmt.Errorf("%s", state.Health.Error)),
			"Check if services are running: make ps",
			"View gateway logs: make logs",
			"Start services: make up",
		)
	}

	if !state.Health.Healthy {
		return componentStatus(c.Name(), status.HealthLevelUnhealthy, fmt.Sprintf("Gateway returned status %d", state.Health.HTTPStatus), details,
			"Check gateway logs: make logs",
			"Restart services: make restart",
		)
	}

	if state.Models.Error != "" {
		return componentStatus(c.Name(), status.HealthLevelWarning, "Gateway responding, but models endpoint unreachable", withDetailError(details, fmt.Errorf("%s", state.Models.Error)))
	}

	if !state.Models.Healthy {
		return componentStatus(c.Name(), status.HealthLevelWarning, fmt.Sprintf("Models endpoint returned status %d", state.Models.HTTPStatus), details)
	}

	return componentStatus(c.Name(), status.HealthLevelHealthy, "Gateway is responding", details)
}
