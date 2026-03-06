// Package collectors provides domain-specific status collectors.
package collectors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// GatewayCollector checks LiteLLM gateway health.
type GatewayCollector struct {
	Host      string
	Port      string
	MasterKey string
}

// Name returns the collector's domain name.
func (c GatewayCollector) Name() string {
	return "gateway"
}

// Collect gathers status information from the LiteLLM gateway.
func (c GatewayCollector) Collect(ctx context.Context) status.ComponentStatus {
	client := &http.Client{Timeout: config.DefaultHTTPTimeout}
	masterKey := strings.TrimSpace(c.MasterKey)
	if masterKey == "" {
		masterKey = strings.TrimSpace(os.Getenv("LITELLM_MASTER_KEY"))
	}
	if masterKey == "" {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "LITELLM_MASTER_KEY not set; authorized gateway checks skipped",
			Suggestions: []string{
				"Set LITELLM_MASTER_KEY in demo/.env or your shell environment",
				"Re-run: make health",
			},
		}
	}

	// Build health URL
	healthURL := fmt.Sprintf("http://%s:%s/health", c.Host, c.Port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
			},
		}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", masterKey))

	resp, err := client.Do(req)
	if err != nil {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelUnhealthy,
			Message: fmt.Sprintf("Gateway unreachable: %v", err),
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
				"Start services: make up",
			},
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Also check /v1/models endpoint
		modelsURL := fmt.Sprintf("http://%s:%s/v1/models", c.Host, c.Port)
		modelsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
		if err != nil {
			return status.ComponentStatus{
				Name:    c.Name(),
				Level:   status.HealthLevelWarning,
				Message: "Gateway /health OK, but /v1/models check failed",
				Details: map[string]any{
					"health_status": resp.StatusCode,
				},
			}
		}
		modelsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", masterKey))

		modelsResp, err := client.Do(modelsReq)
		if err != nil {
			return status.ComponentStatus{
				Name:    c.Name(),
				Level:   status.HealthLevelWarning,
				Message: "Gateway responding, but models endpoint unreachable",
				Details: map[string]any{
					"health_status": resp.StatusCode,
				},
			}
		}
		defer modelsResp.Body.Close()

		if modelsResp.StatusCode == http.StatusOK {
			return status.ComponentStatus{
				Name:    c.Name(),
				Level:   status.HealthLevelHealthy,
				Message: "Gateway is responding",
				Details: map[string]any{
					"health_status": resp.StatusCode,
					"models_status": modelsResp.StatusCode,
				},
			}
		}

		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("Models endpoint returned status %d", modelsResp.StatusCode),
			Details: map[string]any{
				"health_status": resp.StatusCode,
				"models_status": modelsResp.StatusCode,
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelUnhealthy,
		Message: fmt.Sprintf("Gateway returned status %d", resp.StatusCode),
		Details: map[string]any{
			"health_status": resp.StatusCode,
		},
		Suggestions: []string{
			"Check gateway logs: make logs",
			"Restart services: make restart",
		},
	}
}
