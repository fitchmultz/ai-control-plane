// Package doctor provides environment preflight diagnostics.
//
// This file contains concrete check implementations.
package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// DefaultChecks returns all available diagnostic checks in execution order.
func DefaultChecks() []Check {
	return []Check{
		dockerAvailableCheck{},
		portsFreeCheck{},
		envVarsSetCheck{},
		gatewayHealthyCheck{},
		dbConnectableCheck{},
		configValidCheck{},
		credentialsValidCheck{},
	}
}

// dockerAvailableCheck validates Docker availability and accessibility.
type dockerAvailableCheck struct{}

func (c dockerAvailableCheck) ID() string {
	return "docker_available"
}

func (c dockerAvailableCheck) Run(ctx context.Context, opts Options) CheckResult {
	// Check docker binary exists
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Docker Available",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "Docker not found in PATH",
			Suggestions: []string{
				"Install Docker: https://docs.docker.com/get-docker/",
				"Verify installation: docker --version",
			},
		}
	}

	// Check docker daemon is accessible
	cmd := exec.CommandContext(ctx, dockerPath, "info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := "Docker daemon not accessible"
		suggestions := []string{
			"Ensure Docker service is running: sudo systemctl start docker",
			"Add user to docker group: sudo usermod -aG docker $USER",
			"Re-login or run: newgrp docker",
		}

		// Check if permission denied
		if strings.Contains(string(output), "permission denied") {
			msg = "Docker daemon requires permissions"
			suggestions = []string{
				"Add user to docker group: sudo usermod -aG docker $USER",
				"Re-login or run: newgrp docker",
			}
		}

		return CheckResult{
			ID:          c.ID(),
			Name:        "Docker Available",
			Level:       status.HealthLevelUnhealthy,
			Severity:    SeverityPrereq,
			Message:     msg,
			Suggestions: suggestions,
			Details: map[string]any{
				"docker_path": dockerPath,
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Docker Available",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "Docker is available and daemon is accessible",
		Details: map[string]any{
			"docker_path": dockerPath,
		},
	}
}

func (c dockerAvailableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	// Docker installation cannot be auto-remediated safely
	return false, "", nil
}

// portsFreeCheck validates that required ports are available.
type portsFreeCheck struct{}

func (c portsFreeCheck) ID() string {
	return "ports_free"
}

func (c portsFreeCheck) Run(ctx context.Context, opts Options) CheckResult {
	// Default ports if not specified
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
	// Cannot auto-remediate port conflicts safely
	return false, "", nil
}

func isPortOccupied(ctx context.Context, port int) bool {
	addr := fmt.Sprintf("%s:%d", config.DefaultGatewayHost, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true
	}
	listener.Close()
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

	hasGatewayPort := slices.Contains(occupied, gatewayPort)
	if !hasGatewayPort {
		return false
	}

	gatewayHost := opts.GatewayHost
	if strings.TrimSpace(gatewayHost) == "" {
		gatewayHost = config.DefaultGatewayHost
	}

	masterKey := strings.TrimSpace(os.Getenv("LITELLM_MASTER_KEY"))
	if masterKey == "" && strings.TrimSpace(opts.RepoRoot) != "" {
		masterKey = strings.TrimSpace(loadEnvFromFile(filepath.Join(opts.RepoRoot, "demo", ".env"), "LITELLM_MASTER_KEY"))
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

	return resp.StatusCode == http.StatusOK ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden
}

// envVarsSetCheck validates required environment variables are set.
type envVarsSetCheck struct{}

func (c envVarsSetCheck) ID() string {
	return "env_vars_set"
}

func (c envVarsSetCheck) Run(ctx context.Context, opts Options) CheckResult {
	requiredVars := []string{
		"LITELLM_MASTER_KEY",
		"LITELLM_SALT_KEY",
		"DATABASE_URL",
	}

	missing := []string{}
	found := []string{}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			// Try to load from demo/.env as fallback
			envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
			if val := loadEnvFromFile(envPath, v); val == "" {
				missing = append(missing, v)
			} else {
				found = append(found, v)
			}
		} else {
			found = append(found, v)
		}
	}

	if len(missing) > 0 {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Environment Variables Set",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  fmt.Sprintf("Missing required environment variables: %v", missing),
			Suggestions: []string{
				"Run: make install",
				"Or manually set: export LITELLM_MASTER_KEY=sk-...",
				"Copy .env.example to demo/.env and configure",
			},
			Details: map[string]any{
				"missing_vars": missing,
				"found_vars":   len(found),
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Environment Variables Set",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  fmt.Sprintf("All required environment variables set (%d found)", len(found)),
		Details: map[string]any{
			"found_vars": len(found),
		},
	}
}

func (c envVarsSetCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	// Check if .env.example exists and .env doesn't
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	examplePath := filepath.Join(opts.RepoRoot, "demo", ".env.example")

	if _, err := os.Stat(envPath); err == nil {
		return false, "", nil // .env already exists
	}

	if _, err := os.Stat(examplePath); err != nil {
		return false, "", nil // .env.example doesn't exist
	}

	// Copy example to .env
	content, err := os.ReadFile(examplePath)
	if err != nil {
		return false, "", err
	}

	if err := os.WriteFile(envPath, content, 0600); err != nil {
		return false, "", err
	}

	return true, "Created demo/.env from .env.example", nil
}

func loadEnvFromFile(path, key string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	prefix := key + "="
	for _, line := range lines {
		if after, ok := strings.CutPrefix(line, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// gatewayHealthyCheck validates the LiteLLM gateway is healthy.
type gatewayHealthyCheck struct{}

func (c gatewayHealthyCheck) ID() string {
	return "gateway_healthy"
}

func (c gatewayHealthyCheck) Run(ctx context.Context, opts Options) CheckResult {
	masterKey := os.Getenv("LITELLM_MASTER_KEY")
	if masterKey == "" {
		envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
		masterKey = loadEnvFromFile(envPath, "LITELLM_MASTER_KEY")
	}
	masterKey = strings.TrimSpace(masterKey)
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

	host := opts.GatewayHost
	if host == "" {
		host = config.DefaultGatewayHost
	}
	port := opts.GatewayPort
	if port == "" {
		port = strconv.Itoa(config.DefaultLiteLLMPort)
	}

	client := &http.Client{Timeout: config.DefaultHTTPTimeout}
	url := fmt.Sprintf("http://%s:%s/health", host, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Gateway Healthy",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityRuntime,
			Message:  fmt.Sprintf("Failed to create request: %v", err),
			Suggestions: []string{
				"Check if services are running: make ps",
				"View gateway logs: make logs",
			},
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
			Details: map[string]any{
				"host": host,
				"port": port,
			},
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
			Details: map[string]any{
				"host":          host,
				"port":          port,
				"health_status": resp.StatusCode,
			},
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
		Details: map[string]any{
			"host":          host,
			"port":          port,
			"health_status": resp.StatusCode,
		},
	}
}

func (c gatewayHealthyCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	// Cannot auto-remediate gateway issues
	return false, "", nil
}

// dbConnectableCheck validates PostgreSQL connectivity.
type dbConnectableCheck struct{}

func (c dbConnectableCheck) ID() string {
	return "db_connectable"
}

func (c dbConnectableCheck) Run(ctx context.Context, opts Options) CheckResult {
	// Check docker is available first
	if _, err := exec.LookPath("docker"); err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnknown,
			Severity: SeverityPrereq,
			Message:  "Docker not available",
			Suggestions: []string{
				"Install Docker: https://docs.docker.com/get-docker/",
			},
		}
	}

	// Resolve postgres container ID (works for compose names like demo-postgres-1)
	containerCmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=postgres", "--format", "{{.ID}}")
	containerOutput, err := containerCmd.Output()
	containerID := firstNonEmptyLine(string(containerOutput))
	if err != nil || containerID == "" {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "PostgreSQL container not running",
			Suggestions: []string{
				"Start services: make up",
				"Check container status: docker ps",
			},
		}
	}

	// Test database connectivity
	testCmd := exec.CommandContext(ctx, "docker", "exec", containerID,
		"psql", "-U", "litellm", "-d", "litellm", "-t", "-c", "SELECT 1;")
	testOutput, err := testCmd.Output()
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityDomain,
			Message:  "PostgreSQL not accepting connections",
			Suggestions: []string{
				fmt.Sprintf("Check PostgreSQL logs: docker logs %s", containerID),
				"Restart services: make restart",
			},
		}
	}

	if !strings.Contains(string(testOutput), "1") {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Database Connectable",
			Level:    status.HealthLevelWarning,
			Severity: SeverityDomain,
			Message:  "PostgreSQL responded unexpectedly",
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Database Connectable",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "PostgreSQL is accepting connections",
	}
}

func (c dbConnectableCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	// Cannot auto-remediate database issues
	return false, "", nil
}

func firstNonEmptyLine(raw string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// configValidCheck validates deployment configuration.
type configValidCheck struct{}

func (c configValidCheck) ID() string {
	return "config_valid"
}

func (c configValidCheck) Run(ctx context.Context, opts Options) CheckResult {
	requiredFiles := []string{
		"demo/docker-compose.yml",
		"demo/config/litellm.yaml",
	}

	missingFiles := make([]string, 0, len(requiredFiles))
	for _, relPath := range requiredFiles {
		if _, err := os.Stat(filepath.Join(opts.RepoRoot, relPath)); err != nil {
			missingFiles = append(missingFiles, relPath)
		}
	}
	if len(missingFiles) > 0 {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Config Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityPrereq,
			Message:  "Required deployment configuration files are missing",
			Suggestions: []string{
				"Ensure repository is complete",
				"Run: make install",
			},
			Details: map[string]any{
				"missing_files": missingFiles,
			},
		}
	}

	if _, err := os.Stat(filepath.Join(opts.RepoRoot, "demo", ".env")); err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Config Valid",
			Level:    status.HealthLevelWarning,
			Severity: SeverityPrereq,
			Message:  "Environment file demo/.env is missing",
			Suggestions: []string{
				"Run: make install-env",
				"Populate required environment variables in demo/.env",
			},
		}
	}

	return CheckResult{
		ID:       c.ID(),
		Name:     "Config Valid",
		Level:    status.HealthLevelHealthy,
		Severity: SeverityDomain,
		Message:  "Deployment configuration files are present",
	}
}

func (c configValidCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	// Cannot auto-remediate configuration issues
	return false, "", nil
}

// credentialsValidCheck validates that the master key is valid.
type credentialsValidCheck struct{}

func (c credentialsValidCheck) ID() string {
	return "credentials_valid"
}

func (c credentialsValidCheck) Run(ctx context.Context, opts Options) CheckResult {
	// Get master key from environment or demo/.env
	masterKey := os.Getenv("LITELLM_MASTER_KEY")
	if masterKey == "" {
		envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
		masterKey = loadEnvFromFile(envPath, "LITELLM_MASTER_KEY")
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

	// Check if gateway is reachable first
	host := opts.GatewayHost
	if host == "" {
		host = config.DefaultGatewayHost
	}
	port := opts.GatewayPort
	if port == "" {
		port = strconv.Itoa(config.DefaultLiteLLMPort)
	}

	client := &http.Client{Timeout: config.DefaultHTTPTimeout}
	url := fmt.Sprintf("http://%s:%s/v1/models", host, port)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return CheckResult{
			ID:       c.ID(),
			Name:     "Credentials Valid",
			Level:    status.HealthLevelUnhealthy,
			Severity: SeverityRuntime,
			Message:  fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	// Never log the actual key
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
	// Cannot auto-remediate credential issues
	return false, "", nil
}

func sanitizeOutput(output string) string {
	// Remove any potential secrets from output
	lines := strings.Split(output, "\n")
	var result []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "key") ||
			strings.Contains(lower, "secret") ||
			strings.Contains(lower, "token") ||
			strings.Contains(lower, "password") ||
			strings.Contains(lower, "database_url") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}
