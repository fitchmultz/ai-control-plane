// Package config centralizes runtime configuration loading for ACP processes.
//
// Purpose:
//   - Expose typed gateway and process-level runtime configuration.
//
// Responsibilities:
//   - Normalize gateway host/port resolution.
//   - Resolve authenticated operator flows that allow repo-local fallback.
//   - Centralize CI/tooling overrides and UX toggles.
//
// Scope:
//   - Gateway, tooling, and process convenience config only.
//
// Usage:
//   - Call `Loader.Gateway()` or `Loader.Tooling()` from composition layers.
//
// Invariants/Assumptions:
//   - Gateway defaults stay aligned with `internal/config/defaults.go`.
//   - Repo-local fallback is opt-in and limited to operator workflows.
package config

import (
	"strconv"
	"strings"
	"time"
)

// GatewaySettings describes the effective gateway endpoint and auth context.
type GatewaySettings struct {
	Host      string
	Port      string
	PortInt   int
	MasterKey string
}

// ToolingSettings describes process-level command/tooling overrides.
type ToolingSettings struct {
	MakeBinary      string
	ComposeProject  string
	Slot            string
	Role            string
	HomeDir         string
	BackupDir       string
	NoColor         bool
	CIFull          string
	SourceDateEpoch string
}

// Gateway returns gateway settings, optionally allowing repo-local key fallback.
func (l *Loader) Gateway(includeRepoFallback bool) GatewaySettings {
	host := l.StringDefault("GATEWAY_HOST", DefaultGatewayHost)
	portString := l.StringDefault("LITELLM_PORT", strconv.Itoa(DefaultLiteLLMPort))
	port, err := strconv.Atoi(portString)
	if err != nil || port <= 0 {
		port = DefaultLiteLLMPort
		portString = strconv.Itoa(DefaultLiteLLMPort)
	}
	masterKey := l.String("LITELLM_MASTER_KEY")
	if includeRepoFallback && masterKey == "" {
		masterKey = l.RepoAwareString("LITELLM_MASTER_KEY")
	}
	return GatewaySettings{
		Host:      host,
		Port:      portString,
		PortInt:   port,
		MasterKey: masterKey,
	}
}

// Tooling returns process-level ACP command configuration.
func (l *Loader) Tooling() ToolingSettings {
	return ToolingSettings{
		MakeBinary:      l.StringDefault("ACPCTL_MAKE_BIN", "make"),
		ComposeProject:  l.String("ACP_COMPOSE_PROJECT"),
		Slot:            l.String("ACP_SLOT"),
		Role:            l.String("ACP_USER_ROLE"),
		HomeDir:         l.String("HOME"),
		BackupDir:       l.String("BACKUP_DIR"),
		NoColor:         l.String("NO_COLOR") != "",
		CIFull:          l.String("CI_FULL"),
		SourceDateEpoch: l.String("SOURCE_DATE_EPOCH"),
	}
}

// ChargebackForecast returns the three nullable forecast values.
func (l *Loader) ChargebackForecast() (*float64, *float64, *float64) {
	raw := l.String("CHARGEBACK_FORECAST_VALUES")
	if raw == "" {
		raw = "N/A,N/A,N/A"
	}
	parts := splitExactlyThree(raw)
	return nullableFloat(parts[0]), nullableFloat(parts[1]), nullableFloat(parts[2])
}

func splitExactlyThree(raw string) []string {
	parts := strings.Split(raw, ",")
	for len(parts) < 3 {
		parts = append(parts, "N/A")
	}
	return parts[:3]
}

func nullableFloat(raw string) *float64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "n/a") {
		return nil
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil
	}
	return &value
}

// ChargebackTimestamp returns the configured timestamp or a deterministic fallback.
func (l *Loader) ChargebackTimestamp(now time.Time) string {
	return l.StringDefault("CHARGEBACK_PAYLOAD_TIMESTAMP", now.UTC().Format(time.RFC3339))
}
