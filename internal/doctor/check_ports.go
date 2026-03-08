// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//
//	Implement port occupancy diagnostics as a focused module.
//
// Responsibilities:
//   - Check whether required ports are free.
//   - Distinguish expected ACP port usage from unexpected conflicts.
//
// Scope:
//   - TCP port diagnostics only.
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
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

type portsFreeCheck struct{}

func (c portsFreeCheck) ID() string { return "ports_free" }

func (c portsFreeCheck) Run(ctx context.Context, opts Options) CheckResult {
	ports := opts.RequiredPorts
	if len(ports) == 0 {
		ports = config.RequiredPorts()
	}

	occupied := []int{}
	for _, port := range ports {
		if isPortOccupied(ctx, port) {
			occupied = append(occupied, port)
		}
	}
	if len(occupied) > 0 {
		if occupiedPortsBelongToRunningACP(ctx, occupied, opts) {
			return CheckResult{
				ID:       c.ID(),
				Name:     "Ports Free",
				Level:    status.HealthLevelHealthy,
				Severity: SeverityDomain,
				Message:  fmt.Sprintf("Required ports are bound by running AI Control Plane services: %v", occupied),
				Details: map[string]any{
					"required_ports": ports,
					"occupied_ports": occupied,
				},
			}
		}
		return CheckResult{
			ID:       c.ID(),
			Name:     "Ports Free",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  fmt.Sprintf("Ports already in use: %v (expected when services are already running)", occupied),
			Suggestions: []string{
				fmt.Sprintf("Identify processes: ss -tlnp | grep -E ':(%s)'", joinPorts(occupied)),
				fmt.Sprintf("Or: lsof -i :%d", occupied[0]),
				"If these ports belong to AI Control Plane services, this is expected after startup",
				"Otherwise stop conflicting services or choose different ports",
			},
			Details: map[string]any{
				"required_ports": ports,
				"occupied_ports": occupied,
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Ports Free",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("All required ports available: %v", ports),
		Details: map[string]any{
			"checked_ports": ports,
		},
	}
}

func (c portsFreeCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	return false, "", nil
}

func isPortOccupied(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("%s:%d", config.DefaultGatewayHost, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	_ = listener.Close()
	return false
}

func joinPorts(ports []int) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = fmt.Sprintf("%d", p)
	}
	return strings.Join(parts, "|")
}

func occupiedPortsBelongToRunningACP(ctx context.Context, occupied []int, opts Options) bool {
	gatewayPort := config.DefaultLiteLLMPort
	if rawPort := strings.TrimSpace(opts.GatewayPort); rawPort != "" {
		if parsedPort, err := strconv.Atoi(rawPort); err == nil && parsedPort > 0 {
			gatewayPort = parsedPort
		}
	}
	if !slices.Contains(occupied, gatewayPort) {
		return false
	}

	gatewayHost := opts.GatewayHost
	if strings.TrimSpace(gatewayHost) == "" {
		gatewayHost = config.DefaultGatewayHost
	}

	masterKey := config.NewLoader().Gateway(false).MasterKey
	if masterKey == "" && strings.TrimSpace(opts.RepoRoot) != "" {
		masterKey = loadEnvFromFile(filepath.Join(opts.RepoRoot, "demo", ".env"), "LITELLM_MASTER_KEY")
	}

	healthURL := fmt.Sprintf("http://%s:%d/health", gatewayHost, gatewayPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false
	}
	if masterKey != "" {
		req.Header.Set("Authorization", "Bearer "+masterKey)
	}
	client := &http.Client{Timeout: config.DefaultHealthCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden
}
