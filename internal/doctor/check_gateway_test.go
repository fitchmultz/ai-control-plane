// check_gateway_test.go - Focused coverage for gateway health doctor checks.
//
// Purpose:
//   - Verify gateway doctor checks adapt runtime state into prerequisite/domain results.
//
// Responsibilities:
//   - Cover missing runtime, missing master key, and authorized passthrough cases.
//
// Scope:
//   - Gateway diagnostics only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests use synthetic runtime reports only.
package doctor

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func TestGatewayHealthyCheckID(t *testing.T) {
	t.Parallel()
	if (gatewayHealthyCheck{}).ID() != "gateway_healthy" {
		t.Fatalf("expected ID gateway_healthy")
	}
}

func TestGatewayHealthyCheckRunMissingRuntime(t *testing.T) {
	t.Parallel()

	result := (gatewayHealthyCheck{}).Run(context.Background(), Options{})
	if result.Level != status.HealthLevelUnknown || result.Severity != SeverityRuntime {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestGatewayHealthyCheckRunMissingMasterKey(t *testing.T) {
	result := (gatewayHealthyCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: false,
		}, status.HealthLevelWarning, "LITELLM_MASTER_KEY not set; authorized gateway checks skipped"),
	})

	if result.Level != status.HealthLevelUnhealthy || result.Severity != SeverityPrereq {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !strings.Contains(result.Message, "LITELLM_MASTER_KEY not set") {
		t.Fatalf("unexpected message %q", result.Message)
	}
}

func TestGatewayHealthyCheckRunAuthorized(t *testing.T) {
	result := (gatewayHealthyCheck{}).Run(context.Background(), Options{
		RuntimeReport: runtimeReportFor("gateway", status.ComponentDetails{
			MasterKeyConfigured: true,
			HealthReachable:     true,
			ModelsReachable:     true,
			ModelsAuthorized:    true,
			ModelsHTTPStatus:    200,
		}, status.HealthLevelHealthy, "Gateway is responding"),
	})

	if result.Level != status.HealthLevelHealthy || result.Severity != SeverityDomain {
		t.Fatalf("unexpected result: %+v", result)
	}
}
