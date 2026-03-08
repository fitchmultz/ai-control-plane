// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement gateway health diagnostics through the shared typed gateway
//	service so doctor and status evaluate the same runtime model.
//
// Responsibilities:
//   - Perform an authorized gateway probe.
//   - Report actionable remediation when the gateway is unavailable.
//
// Non-scope:
//   - Does not attempt gateway remediation or mutation.
//
// Invariants/Assumptions:
//   - Gateway diagnostics use the shared typed gateway service.
//
// Scope:
//   - LiteLLM gateway diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package doctor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type gatewayHealthyCheck struct{}

func (c gatewayHealthyCheck) ID() string { return "gateway_healthy" }

func (c gatewayHealthyCheck) Run(ctx context.Context, opts Options) CheckResult {
	client := doctorGatewayClient(opts)
	state := client.Status(ctx)
	details := status.ComponentDetails{
		BaseURL:             state.BaseURL,
		HTTPStatus:          state.Health.HTTPStatus,
		ModelsHTTPStatus:    state.Models.HTTPStatus,
		MasterKeyConfigured: state.MasterKeyConfigured,
		Reachable:           state.Health.Reachable || state.Models.Reachable,
		Authorized:          state.Health.Authorized && state.Models.Authorized,
		Error:               state.Health.Error,
	}

	if !state.MasterKeyConfigured {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "LITELLM_MASTER_KEY not set; cannot run authorized gateway check",
			Details:  details,
			Suggestions: []string{
				"Set LITELLM_MASTER_KEY in demo/.env",
				"Or export it in your shell environment",
			},
		}
	}

	if state.Health.Error != "" {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  fmt.Sprintf("Gateway unreachable: %s", state.Health.Error),
			Details:  details,
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
				"Start services: make up",
			},
		}
	}

	if !state.Health.Healthy {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  fmt.Sprintf("Gateway returned status %d", state.Health.HTTPStatus),
			Details:  details,
			Suggestions: []string{
				"Check gateway logs: make logs",
				"Restart services: make restart",
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Gateway Healthy",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "Gateway is responding",
		Details:  details,
	}
}

func (c gatewayHealthyCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}

func doctorGatewayClient(opts Options) *gateway.Client {
	host, port := gatewayLocation(opts)
	clientOptions := []gateway.Option{gateway.WithHost(host), gateway.WithPort(port)}
	if masterKey := loadGatewayMasterKey(opts); masterKey != "" {
		clientOptions = append(clientOptions, gateway.WithMasterKey(masterKey))
	}
	return gateway.NewClient(clientOptions...)
}

func loadGatewayMasterKey(opts Options) string {
	return loadGatewayMasterKeyFromRepo(opts.RepoRoot)
}

func loadGatewayMasterKeyFromRepo(repoRoot string) string {
	return loadGatewayConfig(repoRoot).MasterKey
}

func gatewayLocation(opts Options) (string, int) {
	cfg := loadGatewayConfig(opts.RepoRoot)
	host := cfg.Host
	if opts.GatewayHost != "" {
		host = opts.GatewayHost
	}
	port := cfg.PortInt
	if opts.GatewayPort != "" {
		if parsed, err := strconv.Atoi(opts.GatewayPort); err == nil && parsed > 0 {
			port = parsed
		}
	}
	return host, port
}
