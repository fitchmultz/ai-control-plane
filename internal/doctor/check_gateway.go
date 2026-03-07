// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement gateway health diagnostics as a focused module.
//
// Responsibilities:
//   - Perform an authorized gateway health probe.
//   - Report actionable remediation when the gateway is unavailable.
//
// Scope:
//   - LiteLLM gateway diagnostics only.
package doctor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type gatewayHealthyCheck struct{}

func (c gatewayHealthyCheck) ID() string { return "gateway_healthy" }

func (c gatewayHealthyCheck) Run(ctx context.Context, opts Options) CheckResult {
	masterKey := strings.TrimSpace(loadGatewayMasterKey(opts))
	if masterKey == "" {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "LITELLM_MASTER_KEY not set; cannot run authorized gateway check",
			Suggestions: []string{
				"Set LITELLM_MASTER_KEY in demo/.env",
				"Or export it in your shell environment",
			},
		}
	}

	host, port := gatewayLocation(opts)
	client := &http.Client{Timeout: config.DefaultHTTPTimeout}
	url := fmt.Sprintf("http://%s:%s/health", host, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return CheckResult{
			ID:          c.ID(),
			Name:        "Gateway Healthy",
			Level:       status.HealthLevelUnhealthy,
			Severity:    SeverityRuntime,
			Message:     fmt.Sprintf("Failed to create request: %v", err),
			Suggestions: []string{"Check if services are running: make ps", "View gateway logs: make logs"},
		}
	}
	req.Header.Set("Authorization", "Bearer "+masterKey)
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  fmt.Sprintf("Gateway unreachable: %v", err),
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
				"Start services: make up",
			},
			Details: map[string]any{"host": host, "port": port},
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelHealthy,
			Severity: SeverityDomain,
			Message:  "Gateway is responding",
			Details:  map[string]any{"host": host, "port": port, "health_status": resp.StatusCode},
		}
	}
	return CheckResult{
		ID:       c.ID(),
		Name:     "Gateway Healthy",
		Level:    status.HealthLevelUnhealthy,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("Gateway returned status %d", resp.StatusCode),
		Suggestions: []string{
			"Check gateway logs: make logs",
			"Restart services: make restart",
		},
		Details: map[string]any{"host": host, "port": port, "health_status": resp.StatusCode},
	}
}

func (c gatewayHealthyCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}

func loadGatewayMasterKey(opts Options) string {
	if value := strings.TrimSpace(os.Getenv("LITELLM_MASTER_KEY")); value != "" {
		return value
	}
	return loadEnvFromFile(filepath.Join(opts.RepoRoot, "demo", ".env"), "LITELLM_MASTER_KEY")
}

func gatewayLocation(opts Options) (string, string) {
	host := opts.GatewayHost
	if host == "" {
		host = config.DefaultGatewayHost
	}
	port := opts.GatewayPort
	if port == "" {
		port = strconv.Itoa(config.DefaultLiteLLMPort)
	}
	return host, port
}
