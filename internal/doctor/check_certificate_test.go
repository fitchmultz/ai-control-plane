// check_certificate_test.go - Focused coverage for certificate doctor checks.
//
// Purpose:
//   - Verify certificate doctor checks adapt runtime state and renewal fixes.
//
// Responsibilities:
//   - Cover missing runtime, warning passthrough, and fix wiring.
//
// Scope:
//   - Certificate diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic runtime reports and stubbed renewal functions.
package doctor

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/certlifecycle"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestCertificateHealthyCheckID(t *testing.T) {
	t.Parallel()
	if (certificateHealthyCheck{}).ID() != "certificate_healthy" {
		t.Fatalf("expected ID certificate_healthy")
	}
}

func TestCertificateHealthyCheckRunMissingRuntime(t *testing.T) {
	t.Parallel()

	result := (certificateHealthyCheck{}).Run(context.Background(), Options{})
	if result.Level != status.HealthLevelUnknown || result.Severity != SeverityRuntime {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCertificateHealthyCheckRunWarningPassthrough(t *testing.T) {
	t.Parallel()

	result := (certificateHealthyCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("certificate", status.ComponentDetails{Domain: "gateway.example.com", DaysRemaining: 10}, status.HealthLevelWarning, "Certificate expires in 10 day(s)"),
	})
	if result.Level != status.HealthLevelWarning || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Suggestions) < 4 {
		t.Fatalf("expected actionable suggestions, got %+v", result)
	}
}

func TestCertificateHealthyCheckFixRenewsWhenTLSEnabled(t *testing.T) {
	t.Parallel()

	original := renewCertificateLifecycle
	called := false
	renewCertificateLifecycle = func(context.Context, certlifecycle.Store, certlifecycle.RenewalRequest) (certlifecycle.RenewalResult, error) {
		called = true
		return certlifecycle.RenewalResult{Renewed: true}, nil
	}
	defer func() { renewCertificateLifecycle = original }()

	applied, message, err := (certificateHealthyCheck{}).Fix(context.Background(), Options{
		Gateway:  config.GatewaySettings{Host: "gateway.example.com", BaseURL: "https://gateway.example.com", TLSEnabled: true},
		RepoRoot: t.TempDir(),
	})
	if err != nil || !applied || !called {
		t.Fatalf("unexpected fix result: applied=%t called=%t message=%q err=%v", applied, called, message, err)
	}
}
