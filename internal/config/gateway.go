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
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

// GatewaySettings describes the effective gateway endpoint and auth context.
type GatewaySettings struct {
	Scheme     string
	Host       string
	Port       string
	PortInt    int
	BaseURL    string
	TLSEnabled bool
	MasterKey  string
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
	resolve := l.String
	if includeRepoFallback {
		resolve = l.RepoAwareString
	}

	explicitURL := textutil.FirstNonBlank(resolve("ACP_GATEWAY_URL"), resolve("GATEWAY_URL"))
	if explicitURL != "" {
		if settings, ok := parseGatewayURL(explicitURL); ok {
			settings.MasterKey = resolve("LITELLM_MASTER_KEY")
			return settings
		}
	}

	host := textutil.FirstNonBlank(resolve("GATEWAY_HOST"), DefaultGatewayHost)
	portString := textutil.FirstNonBlank(resolve("LITELLM_PORT"), strconv.Itoa(DefaultLiteLLMPort))
	port, err := strconv.Atoi(portString)
	if err != nil || port <= 0 {
		port = DefaultLiteLLMPort
		portString = strconv.Itoa(DefaultLiteLLMPort)
	}
	scheme := normalizeGatewayScheme(resolve("ACP_GATEWAY_SCHEME"), resolve("GATEWAY_SCHEME"), resolve("ACP_GATEWAY_TLS"))
	masterKey := resolve("LITELLM_MASTER_KEY")
	return GatewaySettings{
		Scheme:     scheme,
		Host:       host,
		Port:       portString,
		PortInt:    port,
		BaseURL:    buildGatewayBaseURL(scheme, host, portString),
		TLSEnabled: scheme == "https",
		MasterKey:  masterKey,
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
	if trimmed == "" || textutil.EqualFoldTrimmed(trimmed, "n/a") {
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

func normalizeGatewayScheme(values ...string) string {
	for _, value := range values {
		switch textutil.LowerTrim(value) {
		case "https", "tls", "true", "1", "yes", "on":
			return "https"
		case "http", "false", "0", "no", "off":
			return "http"
		}
	}
	return "http"
}

func parseGatewayURL(raw string) (GatewaySettings, bool) {
	parsed, err := url.Parse(textutil.Trim(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return GatewaySettings{}, false
	}
	portString := parsed.Port()
	port := defaultPortForScheme(parsed.Scheme)
	if portString != "" {
		parsedPort, convErr := strconv.Atoi(portString)
		if convErr != nil || parsedPort <= 0 {
			return GatewaySettings{}, false
		}
		port = parsedPort
	} else {
		portString = strconv.Itoa(port)
	}
	baseURL := strings.TrimSuffix((&url.URL{
		Scheme: parsed.Scheme,
		Host:   parsed.Host,
	}).String(), "/")
	return GatewaySettings{
		Scheme:     parsed.Scheme,
		Host:       parsed.Hostname(),
		Port:       portString,
		PortInt:    port,
		BaseURL:    baseURL,
		TLSEnabled: textutil.EqualFoldTrimmed(parsed.Scheme, "https"),
	}, true
}

func buildGatewayBaseURL(scheme string, host string, port string) string {
	trimmedScheme := normalizeGatewayScheme(scheme)
	trimmedHost := textutil.Trim(host)
	trimmedPort := textutil.Trim(port)
	if trimmedPort == "" {
		return fmt.Sprintf("%s://%s", trimmedScheme, trimmedHost)
	}
	return fmt.Sprintf("%s://%s:%s", trimmedScheme, trimmedHost, trimmedPort)
}

func defaultPortForScheme(scheme string) int {
	if textutil.EqualFoldTrimmed(scheme, "https") {
		return 443
	}
	return DefaultLiteLLMPort
}
