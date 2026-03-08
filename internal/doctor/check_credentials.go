// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement credential validation diagnostics as a focused module.
//
// Responsibilities:
//   - Verify the configured master key is present.
//   - Check whether the gateway authorizes that master key.
//
// Scope:
//   - Credential diagnostics only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package doctor

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type credentialsValidCheck struct{}

func (c credentialsValidCheck) ID() string { return "credentials_valid" }

func (c credentialsValidCheck) Run(ctx context.Context, opts Options) CheckResult {
	masterKey := config.NewLoader().Gateway(false).MasterKey
	if masterKey == "" {
		masterKey = loadEnvFromFile(filepath.Join(opts.RepoRoot, "demo", ".env"), "LITELLM_MASTER_KEY")
	}
	if masterKey == "" {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "LITELLM_MASTER_KEY not set",
			Suggestions: []string{
				"Run: make install",
				"Set LITELLM_MASTER_KEY environment variable",
			},
		}
	}

	host, port := gatewayLocation(opts)
	client := &http.Client{Timeout: config.DefaultHTTPTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:%s/v1/models", host, port), nil)
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityRuntime,
			Message:  fmt.Sprintf("Failed to create request: %v", err),
		}
	}
	req.Header.Set("Authorization", "Bearer "+masterKey)
	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  "Gateway unreachable; cannot validate credentials",
			Suggestions: []string{
				"Ensure services are running: make up",
				"Check network connectivity",
			},
		}
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelHealthy,
			Severity: SeverityDomain,
			Message:  "Master key is valid",
			Details: map[string]any{
				"auth_status": "authorized",
			},
		}
	case http.StatusUnauthorized:
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "Master key is invalid or placeholder",
			Suggestions: []string{
				"Regenerate master key: make key-gen-master",
				"Check demo/.env for correct key",
			},
			Details: map[string]any{
				"auth_status": "unauthorized",
			},
		}
	default:
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  fmt.Sprintf("Unexpected response: %d", resp.StatusCode),
			Suggestions: []string{
				"Check gateway status: make health",
			},
		}
	}
}

func (c credentialsValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}
