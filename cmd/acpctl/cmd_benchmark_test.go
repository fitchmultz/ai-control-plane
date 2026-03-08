// cmd_benchmark_test.go - Tests for the benchmark CLI surface.
//
// Purpose:
//
//	Verify the benchmark command parses flags correctly and returns stable
//	exit codes and output for local performance baselines.
//
// Responsibilities:
//   - Verify benchmark help and usage behavior
//   - Verify JSON output for successful baseline runs
//   - Verify domain failures when benchmark requests fail
//
// Scope:
//   - Covers command-layer behavior only
//   - Does not exercise a live gateway
//
// Usage:
//   - Run via `go test ./cmd/acpctl`
//
// Invariants/Assumptions:
//   - Tests stub the internal benchmark runner for deterministic output
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/performance"
)

func TestRunBenchmarkCommand_NoArgsReturnsUsage(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := runBenchmarkCommand(context.Background(), nil, stdout, stderr)

	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit code, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stdout), "Usage: acpctl benchmark <subcommand>") {
		t.Fatalf("expected benchmark help output, got %s", readFile(t, stdout))
	}
}

func TestRunBenchmarkBaseline_JSONSuccess(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	original := benchmarkRunner
	benchmarkRunner = func(_ context.Context, opts performance.BaselineOptions) (*performance.Summary, error) {
		if opts.Requests != 5 {
			t.Fatalf("requests = %d, want 5", opts.Requests)
		}
		if opts.Concurrency != 3 {
			t.Fatalf("concurrency = %d, want 3", opts.Concurrency)
		}
		return &performance.Summary{
			GatewayURL:       "http://127.0.0.1:4000",
			Model:            "mock-gpt",
			WarmupRequests:   0,
			Requests:         5,
			Concurrency:      3,
			Successes:        5,
			Failures:         0,
			StartedAtUTC:     time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
			FinishedAtUTC:    time.Date(2026, 3, 5, 0, 0, 2, 0, time.UTC),
			Duration:         "2s",
			RequestsPerSec:   2.5,
			AverageLatencyMS: 180,
			P50LatencyMS:     175,
			P95LatencyMS:     240,
			MinLatencyMS:     150,
			MaxLatencyMS:     250,
		}, nil
	}
	t.Cleanup(func() { benchmarkRunner = original })

	exitCode := runBenchmarkCommand(context.Background(), []string{"baseline", "--requests", "5", "--concurrency", "3", "--json"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	output := readFile(t, stdout)
	if !strings.Contains(output, "\"requests\": 5") || !strings.Contains(output, "\"p95_latency_ms\": 240") {
		t.Fatalf("expected JSON summary output, got %s", output)
	}
}

func TestRunBenchmarkBaseline_FailuresReturnDomainExit(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	original := benchmarkRunner
	benchmarkRunner = func(_ context.Context, _ performance.BaselineOptions) (*performance.Summary, error) {
		return &performance.Summary{
			GatewayURL:     "http://127.0.0.1:4000",
			Model:          "mock-gpt",
			WarmupRequests: 0,
			Requests:       3,
			Concurrency:    1,
			Successes:      2,
			Failures:       1,
			StartedAtUTC:   time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
			FinishedAtUTC:  time.Date(2026, 3, 5, 0, 0, 1, 0, time.UTC),
			Duration:       "1s",
		}, nil
	}
	t.Cleanup(func() { benchmarkRunner = original })

	exitCode := runBenchmarkCommand(context.Background(), []string{"baseline"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain exit code, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stdout), "Failures: 1") {
		t.Fatalf("expected failure count in summary, got %s", readFile(t, stdout))
	}
}

func TestRunBenchmarkBaseline_ProfileAppliesDefaults(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, benchmarkProfileConfigRelativePath), `{
  "schema_version": "1.0.0",
  "defaults": {"warmup_requests": 5, "measured_requests": 40, "concurrency": 2},
  "profiles": {
    "interactive": {
      "description": "Interactive",
      "workload": {"warmup_requests": 5, "measured_requests": 30, "concurrency": 1}
    }
  }
}`)
	stdout, stderr := newTestFiles(t)
	original := benchmarkRunner
	benchmarkRunner = func(_ context.Context, opts performance.BaselineOptions) (*performance.Summary, error) {
		if opts.Profile != "interactive" {
			t.Fatalf("profile = %q, want interactive", opts.Profile)
		}
		if opts.WarmupRequests != 5 || opts.Requests != 30 || opts.Concurrency != 1 {
			t.Fatalf("unexpected profile workload: warmup=%d requests=%d concurrency=%d", opts.WarmupRequests, opts.Requests, opts.Concurrency)
		}
		return &performance.Summary{
			GatewayURL:       opts.GatewayURL,
			Model:            opts.Model,
			Profile:          opts.Profile,
			WarmupRequests:   opts.WarmupRequests,
			Requests:         opts.Requests,
			Concurrency:      opts.Concurrency,
			Successes:        opts.Requests,
			Failures:         0,
			StartedAtUTC:     time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
			FinishedAtUTC:    time.Date(2026, 3, 5, 0, 0, 2, 0, time.UTC),
			Duration:         "2s",
			RequestsPerSec:   15,
			AverageLatencyMS: 200,
			P50LatencyMS:     180,
			P95LatencyMS:     300,
			MinLatencyMS:     150,
			MaxLatencyMS:     400,
		}, nil
	}
	t.Cleanup(func() { benchmarkRunner = original })

	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runBenchmarkCommand(context.Background(), []string{"baseline", "--profile", "interactive"}, stdout, stderr)
	})
	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "Profile: interactive") {
		t.Fatalf("expected profile in summary output, got %s", readFile(t, stdout))
	}
}

func TestRunBenchmarkBaseline_CanceledReturnsRuntime(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	original := benchmarkRunner
	benchmarkRunner = func(_ context.Context, _ performance.BaselineOptions) (*performance.Summary, error) {
		return nil, context.Canceled
	}
	t.Cleanup(func() { benchmarkRunner = original })

	exitCode := runBenchmarkCommand(context.Background(), []string{"baseline"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitRuntime {
		t.Fatalf("expected runtime exit code, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stderr), "benchmark canceled") {
		t.Fatalf("expected cancel message, got %s", readFile(t, stderr))
	}
}
