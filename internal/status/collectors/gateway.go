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
	client *gateway.Client
}

// NewGatewayCollector creates a typed gateway collector.
func NewGatewayCollector(client *gateway.Client) GatewayCollector {
	return GatewayCollector{client: client}
}

// Name returns the collector's domain name.
func (c GatewayCollector) Name() string {
	return "gateway"
}

// Collect gathers status information from the LiteLLM gateway.
func (c GatewayCollector) Collect(ctx context.Context) status.ComponentStatus {
	state := c.client.Status(ctx)
	details := status.ComponentDetails{
		BaseURL:             state.BaseURL,
		MasterKeyConfigured: state.MasterKeyConfigured,
		HTTPStatus:          state.Health.HTTPStatus,
		ModelsHTTPStatus:    state.Models.HTTPStatus,
		Reachable:           state.Health.Reachable || state.Models.Reachable,
		Authorized:          state.Health.Authorized && state.Models.Authorized,
	}

	if !state.MasterKeyConfigured {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "LITELLM_MASTER_KEY not set; authorized gateway checks skipped",
			Details: details,
			Suggestions: []string{
				"Set LITELLM_MASTER_KEY in demo/.env or your shell environment",
				"Re-run: make health",
			},
		}
	}

	if state.Health.Error != "" {
		details.Error = state.Health.Error
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("Gateway unreachable: %s", state.Health.Error),
			Details: details,
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
				"Start services: make up",
			},
		}
	}

	if !state.Health.Healthy {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("Gateway returned status %d", state.Health.HTTPStatus),
			Details: details,
			Suggestions: []string{
				"Check gateway logs: make logs",
				"Restart services: make restart",
			},
		}
	}

	if state.Models.Error != "" {
		details.Error = state.Models.Error
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Gateway responding, but models endpoint unreachable",
			Details: details,
		}
	}

	if !state.Models.Healthy {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("Models endpoint returned status %d", state.Models.HTTPStatus),
			Details: details,
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: "Gateway is responding",
		Details: details,
	}
}
