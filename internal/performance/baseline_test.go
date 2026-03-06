// baseline_test.go - Tests for the lightweight performance baseline harness.
//
// Purpose:
//
//	Verify summary metrics, validation, and failure handling for the local
//	performance baseline workflow.
//
// Responsibilities:
//   - Validate option normalization
//   - Verify aggregate latency metrics and throughput math
//   - Verify mixed success/failure request handling
//
// Scope:
//   - Covers internal performance package behavior only
//
// Usage:
//   - Run via `go test ./internal/performance`
//
// Invariants/Assumptions:
//   - Tests use stubbed request functions instead of a live gateway
package performance

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunBaselineMetrics(t *testing.T) {
	latencies := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond, 400 * time.Millisecond}
	requestIndex := 0
	nowIndex := 0
	timestamps := []time.Time{
		time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 5, 0, 0, 5, 0, time.UTC),
	}
	summary, err := RunBaseline(context.Background(), BaselineOptions{
		GatewayURL:  "http://example.test",
		MasterKey:   "secret",
		Model:       "mock-gpt",
		Requests:    4,
		Concurrency: 2,
		Now: func() time.Time {
			value := timestamps[nowIndex]
			if nowIndex < len(timestamps)-1 {
				nowIndex++
			}
			return value
		},
		RequestFunc: func(_ context.Context, _ BaselineOptions) (time.Duration, error) {
			latency := latencies[requestIndex]
			requestIndex++
			return latency, nil
		},
	})
	if err != nil {
		t.Fatalf("RunBaseline() error = %v", err)
	}
	if summary.Successes != 4 || summary.Failures != 0 {
		t.Fatalf("success/failure = %d/%d, want 4/0", summary.Successes, summary.Failures)
	}
	if summary.RequestsPerSec != 0.8 {
		t.Fatalf("requests/sec = %f, want 0.8", summary.RequestsPerSec)
	}
	if summary.P50LatencyMS != 250 {
		t.Fatalf("p50 = %f, want 250", summary.P50LatencyMS)
	}
	if summary.P95LatencyMS != 385 {
		t.Fatalf("p95 = %f, want 385", summary.P95LatencyMS)
	}
}

func TestRunBaselineWarmupRequestsAreExcluded(t *testing.T) {
	requestIndex := 0
	summary, err := RunBaseline(context.Background(), BaselineOptions{
		GatewayURL:     "http://example.test",
		MasterKey:      "secret",
		Model:          "mock-gpt",
		Profile:        "interactive",
		WarmupRequests: 2,
		Requests:       3,
		Concurrency:    1,
		Now: func() time.Time {
			return time.Date(2026, 3, 5, 0, 0, 2, 0, time.UTC)
		},
		RequestFunc: func(_ context.Context, _ BaselineOptions) (time.Duration, error) {
			requestIndex++
			return 50 * time.Millisecond, nil
		},
	})
	if err != nil {
		t.Fatalf("RunBaseline() error = %v", err)
	}
	if requestIndex != 5 {
		t.Fatalf("request count = %d, want 5 total invocations", requestIndex)
	}
	if summary.WarmupRequests != 2 {
		t.Fatalf("warmup requests = %d, want 2", summary.WarmupRequests)
	}
	if summary.Profile != "interactive" {
		t.Fatalf("profile = %q, want interactive", summary.Profile)
	}
	if len(summary.Samples) != 3 {
		t.Fatalf("samples = %d, want 3 measured requests", len(summary.Samples))
	}
}

func TestRunBaselineMixedResults(t *testing.T) {
	requestIndex := 0
	summary, err := RunBaseline(context.Background(), BaselineOptions{
		GatewayURL:  "http://example.test",
		MasterKey:   "secret",
		Requests:    3,
		Concurrency: 1,
		Now: func() time.Time {
			base := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
			return base.Add(3 * time.Second)
		},
		RequestFunc: func(_ context.Context, _ BaselineOptions) (time.Duration, error) {
			requestIndex++
			if requestIndex == 2 {
				return 50 * time.Millisecond, errors.New("boom")
			}
			return 75 * time.Millisecond, nil
		},
	})
	if err != nil {
		t.Fatalf("RunBaseline() error = %v", err)
	}
	if summary.Successes != 2 || summary.Failures != 1 {
		t.Fatalf("success/failure = %d/%d, want 2/1", summary.Successes, summary.Failures)
	}
	if len(summary.Samples) != 3 {
		t.Fatalf("samples = %d, want 3", len(summary.Samples))
	}
}

func TestNormalizeOptionsValidation(t *testing.T) {
	_, err := normalizeOptions(BaselineOptions{Requests: 1, Concurrency: 1})
	if err == nil {
		t.Fatal("expected missing master key error")
	}
	_, err = normalizeOptions(BaselineOptions{MasterKey: "x", Requests: 0, Concurrency: 1})
	if err == nil {
		t.Fatal("expected invalid requests error")
	}
	_, err = normalizeOptions(BaselineOptions{MasterKey: "x", Requests: 1, Concurrency: 0})
	if err == nil {
		t.Fatal("expected invalid concurrency error")
	}
	_, err = normalizeOptions(BaselineOptions{MasterKey: "x", WarmupRequests: -1, Requests: 1, Concurrency: 1})
	if err == nil {
		t.Fatal("expected invalid warmup requests error")
	}
}
