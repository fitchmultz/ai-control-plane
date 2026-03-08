// gateway_test validates GatewayCollector HTTP-based status checks.
//
// Purpose:
//
//	Ensure gateway health collection correctly interprets HTTP responses
//	from /health and /v1/models endpoints under various conditions.
//
// Responsibilities:
//   - Verify authorized health check behavior for HTTP 200 responses.
//   - Verify models endpoint validation.
//   - Verify error handling for unreachable and error-status gateways.
//   - Verify timeout and context cancellation handling.
//
// Non-scope:
//   - Does not test against real running LiteLLM services.
//
// Invariants/Assumptions:
//   - HTTP client respects the 5-second timeout.
//   - Authorization header must be present for gateway checks.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package collectors

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const testMasterKey = "test-master-key"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func gatewayCollectorForURL(serverURL string) GatewayCollector {
	host, port, err := net.SplitHostPort(strings.TrimPrefix(serverURL, "http://"))
	if err != nil {
		panic(err)
	}
	return GatewayCollector{
		Host:      host,
		Port:      port,
		MasterKey: testMasterKey,
	}
}

func TestGatewayCollector_Name(t *testing.T) {
	t.Parallel()

	c := GatewayCollector{}
	if c.Name() != "gateway" {
		t.Fatalf("expected name 'gateway', got %q", c.Name())
	}
}

func TestGatewayCollector_MissingMasterKey(t *testing.T) {
	// Not parallel: mutates process environment.
	oldKey := os.Getenv("LITELLM_MASTER_KEY")
	defer os.Setenv("LITELLM_MASTER_KEY", oldKey)
	os.Unsetenv("LITELLM_MASTER_KEY")

	c := GatewayCollector{
		Host:      "127.0.0.1",
		Port:      "4000",
		MasterKey: "",
	}

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning when master key is missing, got %s", result.Level)
	}

	if result.Message != "LITELLM_MASTER_KEY not set; authorized gateway checks skipped" {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestGatewayCollector_HealthEndpoint_Healthy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		healthStatus  int
		modelsStatus  int
		expectedLevel status.HealthLevel
		expectedMsg   string
	}{
		{
			name:          "both endpoints return 200",
			healthStatus:  http.StatusOK,
			modelsStatus:  http.StatusOK,
			expectedLevel: status.HealthLevelHealthy,
			expectedMsg:   "Gateway is responding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer "+testMasterKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				switch r.URL.Path {
				case "/health":
					w.WriteHeader(tt.healthStatus)
				case "/v1/models":
					w.WriteHeader(tt.modelsStatus)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			c := gatewayCollectorForURL(server.URL)

			ctx := context.Background()
			result := c.Collect(ctx)

			if result.Level != tt.expectedLevel {
				t.Fatalf("expected level %s, got %s", tt.expectedLevel, result.Level)
			}

			if result.Name != "gateway" {
				t.Fatalf("expected name 'gateway', got %q", result.Name)
			}

			if result.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, result.Message)
			}

			// Verify details contain status codes
			if result.Level == status.HealthLevelHealthy {
				details, ok := result.Details.(map[string]any)
				if !ok {
					t.Fatal("expected details to be map[string]any")
				}
				if _, ok := details["health_status"]; !ok {
					t.Fatal("expected health_status in details")
				}
			}
		})
	}
}

func TestGatewayCollector_HealthEndpoint_ErrorStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		healthStatus int
		expectedMsg  string
	}{
		{
			name:         "health returns 500",
			healthStatus: http.StatusInternalServerError,
			expectedMsg:  "Gateway returned status 500",
		},
		{
			name:         "health returns 503",
			healthStatus: http.StatusServiceUnavailable,
			expectedMsg:  "Gateway returned status 503",
		},
		{
			name:         "health returns 404",
			healthStatus: http.StatusNotFound,
			expectedMsg:  "Gateway returned status 404",
		},
		{
			name:         "health returns 401",
			healthStatus: http.StatusUnauthorized,
			expectedMsg:  "Gateway returned status 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.healthStatus)
			}))
			defer server.Close()

			c := GatewayCollector{
				Host:      server.Listener.Addr().(*net.TCPAddr).IP.String(),
				Port:      fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port),
				MasterKey: testMasterKey,
			}

			ctx := context.Background()
			result := c.Collect(ctx)

			if result.Level != status.HealthLevelUnhealthy {
				t.Fatalf("expected unhealthy, got %s", result.Level)
			}

			if result.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, result.Message)
			}

			if len(result.Suggestions) == 0 {
				t.Fatal("expected suggestions for unhealthy status")
			}
		})
	}
}

func TestGatewayCollector_HealthEndpoint_WarningModelsStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		modelsStatus int
		expectedMsg  string
	}{
		{
			name:         "models returns 500",
			modelsStatus: http.StatusInternalServerError,
			expectedMsg:  "Models endpoint returned status 500",
		},
		{
			name:         "models returns 404",
			modelsStatus: http.StatusNotFound,
			expectedMsg:  "Models endpoint returned status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer "+testMasterKey {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				switch r.URL.Path {
				case "/health":
					w.WriteHeader(http.StatusOK)
				case "/v1/models":
					w.WriteHeader(tt.modelsStatus)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			c := GatewayCollector{
				Host:      server.Listener.Addr().(*net.TCPAddr).IP.String(),
				Port:      fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port),
				MasterKey: testMasterKey,
			}

			ctx := context.Background()
			result := c.Collect(ctx)

			if result.Level != status.HealthLevelWarning {
				t.Fatalf("expected warning, got %s", result.Level)
			}

			if result.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, result.Message)
			}
		})
	}
}

func TestGatewayCollector_Unreachable(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	c := GatewayCollector{
		Host:      addr.IP.String(),
		Port:      fmt.Sprintf("%d", addr.Port),
		MasterKey: testMasterKey,
	}

	ctx := context.Background()
	result := c.Collect(ctx)

	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy for unreachable, got %s", result.Level)
	}

	if result.Name != "gateway" {
		t.Fatalf("expected name 'gateway', got %q", result.Name)
	}

	if !strings.Contains(result.Message, "Gateway unreachable") {
		t.Fatalf("expected 'Gateway unreachable' in message, got %q", result.Message)
	}

	if len(result.Suggestions) == 0 {
		t.Fatal("expected suggestions for unreachable gateway")
	}
}

func TestGatewayCollector_ContextCancellation(t *testing.T) {
	t.Parallel()

	c := GatewayCollector{
		Host:      "127.0.0.1",
		Port:      "4000",
		MasterKey: testMasterKey,
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := c.Collect(ctx)

	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy for cancelled context, got %s", result.Level)
	}

	if !strings.Contains(result.Message, "Gateway unreachable") {
		t.Fatalf("expected 'Gateway unreachable' in message, got %q", result.Message)
	}
	if !strings.Contains(result.Message, context.Canceled.Error()) {
		t.Fatalf("expected cancellation detail in message, got %q", result.Message)
	}
}

func TestGatewayCollector_Timeout(t *testing.T) {
	t.Parallel()

	c := GatewayCollector{
		Host:      "127.0.0.1",
		Port:      "4000",
		MasterKey: testMasterKey,
		client: &http.Client{
			Timeout: 20 * time.Millisecond,
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			}),
		},
	}

	result := c.Collect(context.Background())

	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy for timeout, got %s", result.Level)
	}
	if !strings.Contains(result.Message, "Gateway unreachable") {
		t.Fatalf("expected gateway unreachable message, got %q", result.Message)
	}
	if !strings.Contains(result.Message, context.DeadlineExceeded.Error()) {
		t.Fatalf("expected timeout detail in message, got %q", result.Message)
	}
}
