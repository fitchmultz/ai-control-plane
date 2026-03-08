// collectors_test.go - Tests for typed status collectors.
//
// Purpose:
//
//	Exercise the collectors against the new shared gateway/database service
//	contracts without depending on the removed Docker-shell helpers.
//
// Responsibilities:
//   - Verify gateway collector behavior against a test HTTP server.
//   - Verify database-backed collectors handle configuration/query failures.
//
// Non-scope:
//   - Does not require a live PostgreSQL or Docker runtime.
//
// Invariants/Assumptions:
//   - Collectors consume the shared typed services introduced by the cutover.
//
// Scope:
//   - Collector smoke tests only.
//
// Usage:
//   - Used through `go test` for collectors package coverage.
package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func gatewayClientForServer(t *testing.T, server *httptest.Server, opts ...gateway.Option) *gateway.Client {
	t.Helper()
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}
	base := []gateway.Option{gateway.WithHost(parsed.Hostname()), gateway.WithPort(port)}
	base = append(base, opts...)
	return gateway.NewClient(base...)
}

func TestGatewayCollectorHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	collector := NewGatewayCollector(gatewayClientForServer(t, server, gateway.WithMasterKey("sk-test")))
	result := collector.Collect(context.Background())

	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy gateway, got %s", result.Level)
	}
	if result.Details.HTTPStatus != http.StatusOK {
		t.Fatalf("expected health status 200, got %d", result.Details.HTTPStatus)
	}
}

func TestDatabaseCollectorAmbiguousConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")
	t.Setenv("ACP_DATABASE_MODE", "")

	connector := db.NewConnector("")
	collector := NewDatabaseCollector(db.NewRuntimeService(connector))
	result := collector.Collect(context.Background())
	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy database status, got %s", result.Level)
	}
}

func TestKeysCollectorQueryFailureBecomesWarning(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")
	t.Setenv("ACP_DATABASE_MODE", "external")

	connector := db.NewConnector("")
	collector := NewKeysCollector(db.NewReadonlyService(connector))
	result := collector.Collect(context.Background())
	if result.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning keys status, got %s", result.Level)
	}
}

func TestBudgetCollectorQueryFailureBecomesWarning(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db?sslmode=disable&connect_timeout=1")
	t.Setenv("ACP_DATABASE_MODE", "external")

	connector := db.NewConnector("")
	collector := NewBudgetCollector(db.NewReadonlyService(connector))
	result := collector.Collect(context.Background())
	if result.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning budget status, got %s", result.Level)
	}
}

func TestDetectionsCollectorMissingConfig(t *testing.T) {
	connector := db.NewConnector("")
	collector := NewDetectionsCollector(t.TempDir(), db.NewReadonlyService(connector))
	result := collector.Collect(context.Background())
	if result.Level != status.HealthLevelUnknown {
		t.Fatalf("expected unknown detections status, got %s", result.Level)
	}
}
