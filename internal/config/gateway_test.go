// gateway_test.go - Coverage for gateway and tooling config helpers.
//
// Purpose:
//   - Verify gateway, tooling, and chargeback helper configuration stays deterministic.
//
// Responsibilities:
//   - Cover explicit URL parsing and host/port fallback logic.
//   - Cover tooling, forecast, timestamp, and helper normalization functions.
//
// Scope:
//   - Gateway/process convenience config only.
//
// Usage:
//   - Run via `go test ./internal/config`.
//
// Invariants/Assumptions:
//   - Tests use explicit loaders and avoid host-specific state where possible.
package config

import (
	"slices"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

func TestDatabaseModeHelpers(t *testing.T) {
	if DatabaseModeExternal.String() != "external" {
		t.Fatalf("String() = %q", DatabaseModeExternal.String())
	}
	if !DatabaseModeEmbedded.IsEmbedded() {
		t.Fatal("expected embedded mode to report embedded")
	}
	if !DatabaseModeExternal.IsExternal() {
		t.Fatal("expected external mode to report external")
	}
}

func TestRequiredPorts(t *testing.T) {
	if !slices.Equal(RequiredPorts(), []int{DefaultLiteLLMPort, DefaultPostgresPort}) {
		t.Fatalf("RequiredPorts() = %v", RequiredPorts())
	}
}

func TestResolveGatewaySettingsUsesExplicitURL(t *testing.T) {
	settings := ResolveGatewaySettings(GatewayResolveInput{
		URL:       "https://gateway.example.com",
		MasterKey: " master-key ",
	})
	if settings.Scheme != "https" || settings.Host != "gateway.example.com" || settings.PortInt != 443 {
		t.Fatalf("unexpected gateway settings: %+v", settings)
	}
	if settings.BaseURL != "https://gateway.example.com" || !settings.TLSEnabled || settings.MasterKey != "master-key" {
		t.Fatalf("unexpected gateway settings: %+v", settings)
	}
}

func TestResolveGatewaySettingsFallsBackWhenExplicitURLIsInvalid(t *testing.T) {
	settings := ResolveGatewaySettings(GatewayResolveInput{
		URL:  "not-a-url",
		Host: "gateway.internal",
		Port: "not-a-port",
		TLS:  "true",
	})
	if settings.Scheme != "https" || settings.PortInt != DefaultLiteLLMPort || settings.Port != "4000" {
		t.Fatalf("unexpected gateway fallback settings: %+v", settings)
	}
	if settings.BaseURL != "https://gateway.internal:4000" {
		t.Fatalf("unexpected gateway fallback settings: %+v", settings)
	}
}

func TestLoaderGatewayFallsBackToHostPortAndRepoAwareKey(t *testing.T) {
	loader := NewTestLoader(map[string]string{
		"GATEWAY_HOST": "gateway.internal",
		"LITELLM_PORT": "not-a-port",
	}, "/repo", map[string]string{
		"LITELLM_MASTER_KEY": "repo-key",
		"ACP_GATEWAY_TLS":    "true",
	})

	settings := loader.Gateway(true)
	if settings.Scheme != "https" || settings.PortInt != DefaultLiteLLMPort || settings.Port != "4000" {
		t.Fatalf("unexpected gateway fallback settings: %+v", settings)
	}
	if settings.BaseURL != "https://gateway.internal:4000" || settings.MasterKey != "repo-key" {
		t.Fatalf("unexpected gateway fallback settings: %+v", settings)
	}
}

func TestLoaderGatewayUsesRepoFallbackForHostAndPort(t *testing.T) {
	loader := NewTestLoader(nil, "/repo", map[string]string{
		"GATEWAY_HOST":       "repo-gateway.internal",
		"LITELLM_PORT":       "8443",
		"ACP_GATEWAY_TLS":    "true",
		"LITELLM_MASTER_KEY": "repo-key",
	})

	settings := loader.Gateway(true)
	if settings.Host != "repo-gateway.internal" {
		t.Fatalf("Host = %q", settings.Host)
	}
	if settings.Port != "8443" || settings.PortInt != 8443 {
		t.Fatalf("unexpected port settings: %+v", settings)
	}
	if settings.BaseURL != "https://repo-gateway.internal:8443" {
		t.Fatalf("BaseURL = %q", settings.BaseURL)
	}
	if settings.MasterKey != "repo-key" {
		t.Fatalf("MasterKey = %q", settings.MasterKey)
	}
}

func TestLoaderGatewayIgnoresLegacyGatewayScheme(t *testing.T) {
	loader := NewTestLoader(map[string]string{
		"GATEWAY_SCHEME": "https",
	}, "/repo", nil)

	settings := loader.Gateway(false)
	if settings.Scheme != "http" {
		t.Fatalf("expected legacy GATEWAY_SCHEME to be ignored, got %+v", settings)
	}
}

func TestToolingAndChargebackHelpers(t *testing.T) {
	now := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	loader := NewTestLoader(map[string]string{
		"ACPCTL_MAKE_BIN":              "gmake",
		"ACP_COMPOSE_PROJECT":          "acp-demo",
		"ACP_SLOT":                     "standby",
		"ACP_USER_ROLE":                "operator",
		"HOME":                         "/home/tester",
		"BACKUP_DIR":                   "/tmp/backups",
		"NO_COLOR":                     "1",
		"CI_FULL":                      "true",
		"SOURCE_DATE_EPOCH":            "123",
		"CHARGEBACK_FORECAST_VALUES":   "1.5, N/A,2.5",
		"CHARGEBACK_PAYLOAD_TIMESTAMP": "2026-03-08T00:00:00Z",
	}, "/repo", nil)

	tooling := loader.Tooling()
	if tooling.MakeBinary != "gmake" || tooling.ComposeProject != "acp-demo" || tooling.Slot != "standby" || tooling.Role != "operator" || tooling.HomeDir != "/home/tester" || tooling.BackupDir != "/tmp/backups" || !tooling.NoColor || tooling.CIFull != "true" || tooling.SourceDateEpoch != "123" {
		t.Fatalf("unexpected tooling settings: %+v", tooling)
	}

	first, second, third := loader.ChargebackForecast()
	if first == nil || *first != 1.5 || second != nil || third == nil || *third != 2.5 {
		t.Fatalf("unexpected forecast values: %v %v %v", first, second, third)
	}
	if timestamp := loader.ChargebackTimestamp(now); timestamp != "2026-03-08T00:00:00Z" {
		t.Fatalf("ChargebackTimestamp() = %q", timestamp)
	}

	fallbackLoader := NewTestLoader(nil, "/repo", nil)
	a, b, c := fallbackLoader.ChargebackForecast()
	if a != nil || b != nil || c != nil {
		t.Fatalf("expected nil forecast defaults, got %v %v %v", a, b, c)
	}
	if timestamp := fallbackLoader.ChargebackTimestamp(now); timestamp != now.Format(time.RFC3339) {
		t.Fatalf("fallback ChargebackTimestamp() = %q", timestamp)
	}
}

func TestGatewayHelperFunctions(t *testing.T) {
	if scheme := normalizeGatewayScheme(" tls "); scheme != "https" {
		t.Fatalf("normalizeGatewayScheme() = %q", scheme)
	}
	if scheme := normalizeGatewayScheme("no"); scheme != "http" {
		t.Fatalf("normalizeGatewayScheme() = %q", scheme)
	}

	parsed, ok := parseGatewayURL("https://gateway.example.com:8443")
	if !ok || parsed.Host != "gateway.example.com" || parsed.PortInt != 8443 || parsed.BaseURL != "https://gateway.example.com:8443" {
		t.Fatalf("parseGatewayURL() = %+v ok=%t", parsed, ok)
	}
	if _, ok := parseGatewayURL("https://gateway.example.com:not-a-port"); ok {
		t.Fatal("expected invalid port URL to fail")
	}
	if _, ok := parseGatewayURL("not-a-url"); ok {
		t.Fatal("expected malformed URL to fail")
	}

	if baseURL := buildGatewayBaseURL("https", "gateway.example.com", "8443"); baseURL != "https://gateway.example.com:8443" {
		t.Fatalf("buildGatewayBaseURL() = %q", baseURL)
	}
	if baseURL := buildGatewayBaseURL("http", "gateway.example.com", ""); baseURL != "http://gateway.example.com" {
		t.Fatalf("buildGatewayBaseURL() = %q", baseURL)
	}

	if defaultPortForScheme("https") != 443 || defaultPortForScheme("http") != DefaultLiteLLMPort {
		t.Fatalf("unexpected default ports: https=%d http=%d", defaultPortForScheme("https"), defaultPortForScheme("http"))
	}
	if value := textutil.FirstNonBlank("", " alpha ", "beta"); value != "alpha" {
		t.Fatalf("FirstNonBlank() = %q", value)
	}
}
