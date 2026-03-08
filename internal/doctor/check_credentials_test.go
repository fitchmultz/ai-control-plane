// check_credentials_test.go - Focused coverage for credential-validation checks.
//
// Purpose:
//   - Verify doctor credential checks classify gateway auth states correctly.
//
// Responsibilities:
//   - Cover missing key, unreachable gateway, unauthorized key, authorized key, and unexpected status.
//
// Scope:
//   - Credential diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic gateway runtime state only.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestCredentialsValidCheckID(t *testing.T) {
	t.Parallel()
	if (credentialsValidCheck{}).ID() != "credentials_valid" {
		t.Fatalf("expected ID credentials_valid")
	}
}

func TestCredentialsValidCheckRunMissingKey(t *testing.T) {
	result := (credentialsValidCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: false,
		}, status.HealthLevelWarning, "missing"),
	})

	if result.Level != status.HealthLevelUnhealthy || result.Severity != SeverityPrereq {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCredentialsValidCheckRunGatewayUnreachable(t *testing.T) {
	result := (credentialsValidCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: true,
			ModelsReachable:     false,
			Error:               "dial tcp",
		}, status.HealthLevelWarning, "gateway unreachable"),
	})

	if result.Level != status.HealthLevelWarning || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCredentialsValidCheckRunUnauthorized(t *testing.T) {
	result := (credentialsValidCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: true,
			ModelsReachable:     true,
			ModelsAuthorized:    false,
			ModelsHTTPStatus:    401,
		}, status.HealthLevelUnhealthy, "unauthorized"),
	})

	if result.Level != status.HealthLevelUnhealthy || result.Details.AuthStatus != "unauthorized" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCredentialsValidCheckRunAuthorized(t *testing.T) {
	result := (credentialsValidCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: true,
			ModelsReachable:     true,
			ModelsAuthorized:    true,
			ModelsHTTPStatus:    200,
		}, status.HealthLevelHealthy, "authorized"),
	})

	if result.Level != status.HealthLevelHealthy || result.Details.AuthStatus != "authorized" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCredentialsValidCheckRunUnexpectedStatus(t *testing.T) {
	result := (credentialsValidCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: true,
			ModelsReachable:     true,
			ModelsAuthorized:    true,
			ModelsHTTPStatus:    418,
		}, status.HealthLevelWarning, "teapot"),
	})

	if result.Level != status.HealthLevelWarning || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}
