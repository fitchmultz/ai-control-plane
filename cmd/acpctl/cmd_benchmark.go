// cmd_benchmark.go - Performance benchmark command implementation.
//
// Purpose:
//
//	Provide a typed CLI surface for local baseline benchmarking of the AI
//	Control Plane gateway.
//
// Responsibilities:
//   - Parse benchmark command arguments
//   - Execute the internal lightweight performance harness
//   - Emit human-readable or JSON output with stable exit codes
//
// Scope:
//   - Covers the `acpctl benchmark baseline` command family
//   - Supports local reference-host baseline measurements only
//
// Usage:
//   - Run via `acpctl benchmark baseline`
//   - Invoked by `make performance-baseline`
//
// Invariants/Assumptions:
//   - Output remains suitable for both operators and machine parsing
//   - The command does not claim customer-grade capacity proof
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/performance"
)

var benchmarkRunner = performance.RunBaseline

const benchmarkProfileConfigRelativePath = "demo/config/benchmark_thresholds.json"

type benchmarkBaselineOptions struct {
	GatewayURL     string
	MasterKey      string
	Model          string
	Profile        string
	Prompt         string
	WarmupRequests int
	Requests       int
	Concurrency    int
	MaxTokens      int
	JSON           bool
	WarmupSet      bool
	RequestsSet    bool
	ConcurrencySet bool
}

func benchmarkCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "benchmark",
		Summary:     "Lightweight local performance baseline",
		Description: "Lightweight local performance baseline.",
		Examples: []string{
			"acpctl benchmark baseline",
			"acpctl benchmark baseline --profile interactive",
			"acpctl benchmark baseline --requests 40 --concurrency 4",
			"acpctl benchmark baseline --json",
		},
		Children: []*commandSpec{
			{
				Name:        "baseline",
				Summary:     "Run the local gateway performance baseline",
				Description: "Run the local gateway performance baseline.",
				Examples: []string{
					"acpctl benchmark baseline",
					"acpctl benchmark baseline --profile interactive",
					"acpctl benchmark baseline --requests 40 --concurrency 4",
					"acpctl benchmark baseline --json",
				},
				Options: []commandOptionSpec{
					{Name: "gateway-url", ValueName: "URL", Summary: "Gateway base URL", Type: optionValueString, DefaultText: "http://127.0.0.1:4000"},
					{Name: "master-key", ValueName: "VALUE", Summary: "Gateway master key", Type: optionValueString, DefaultText: "LITELLM_MASTER_KEY env var"},
					{Name: "model", ValueName: "NAME", Summary: "Model alias to exercise", Type: optionValueString, DefaultText: "mock-gpt"},
					{Name: "profile", ValueName: "NAME", Summary: "Benchmark profile from demo/config/benchmark_thresholds.json", Type: optionValueString},
					{Name: "prompt", ValueName: "TEXT", Summary: "Prompt body to send", Type: optionValueString, DefaultText: "short fixed prompt"},
					{Name: "warmup-requests", ValueName: "N", Summary: "Warmup requests excluded from measured results", Type: optionValueInt, DefaultText: "0"},
					{Name: "requests", ValueName: "N", Summary: "Total request count", Type: optionValueInt, DefaultText: "20"},
					{Name: "concurrency", ValueName: "N", Summary: "Number of concurrent workers", Type: optionValueInt, DefaultText: "2"},
					{Name: "max-tokens", ValueName: "N", Summary: "max_tokens for each request", Type: optionValueInt, DefaultText: "32"},
					{Name: "json", Summary: "Emit machine-readable JSON to stdout", Type: optionValueBool},
				},
				Sections: []commandHelpSection{
					{
						Title: "Notes",
						Lines: []string{
							"This command produces a local reference-host baseline, not customer-grade capacity proof.",
							"Use offline-safe models such as mock-gpt or mock-claude for deterministic local runs.",
							"Profile names currently include interactive, burst, and sustained.",
						},
					},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindBenchmarkBaselineOptions,
					NativeRun:  runBenchmarkBaseline,
				},
			},
		},
	}
}

func bindBenchmarkBaselineOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	gatewayRuntime := config.NewLoader().Gateway(true)
	warmupRequests, err := input.IntDefault("warmup-requests", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid --warmup-requests: %q", input.String("warmup-requests"))
	}
	requests, err := input.IntDefault("requests", 20)
	if err != nil {
		return nil, fmt.Errorf("invalid --requests: %q", input.String("requests"))
	}
	concurrency, err := input.IntDefault("concurrency", 2)
	if err != nil {
		return nil, fmt.Errorf("invalid --concurrency: %q", input.String("concurrency"))
	}
	maxTokens, err := input.IntDefault("max-tokens", 32)
	if err != nil {
		return nil, fmt.Errorf("invalid --max-tokens: %q", input.String("max-tokens"))
	}
	opts := benchmarkBaselineOptions{
		GatewayURL:     input.StringDefault("gateway-url", "http://127.0.0.1:4000"),
		MasterKey:      input.StringDefault("master-key", gatewayRuntime.MasterKey),
		Model:          input.StringDefault("model", "mock-gpt"),
		Profile:        input.String("profile"),
		Prompt:         input.StringDefault("prompt", "Provide a short response for performance baseline verification."),
		WarmupRequests: warmupRequests,
		Requests:       requests,
		Concurrency:    concurrency,
		MaxTokens:      maxTokens,
		JSON:           input.Bool("json"),
		WarmupSet:      input.Has("warmup-requests"),
		RequestsSet:    input.Has("requests"),
		ConcurrencySet: input.Has("concurrency"),
	}
	if opts.Requests <= 0 {
		return nil, fmt.Errorf("--requests must be a positive integer")
	}
	if opts.Concurrency <= 0 {
		return nil, fmt.Errorf("--concurrency must be a positive integer")
	}
	if opts.WarmupRequests < 0 {
		return nil, fmt.Errorf("--warmup-requests must be zero or greater")
	}
	if opts.MaxTokens <= 0 {
		return nil, fmt.Errorf("--max-tokens must be a positive integer")
	}
	return opts, nil
}

func runBenchmarkBaseline(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(benchmarkBaselineOptions)
	logger := workflowLogger(runCtx, "benchmark_baseline", "model", opts.Model, "profile", opts.Profile, "json", opts.JSON)
	workflowStart(logger, "gateway_url", opts.GatewayURL, "requests", opts.Requests, "concurrency", opts.Concurrency)
	runnerOpts := performance.BaselineOptions{
		GatewayURL:     opts.GatewayURL,
		MasterKey:      opts.MasterKey,
		Model:          opts.Model,
		Prompt:         opts.Prompt,
		WarmupRequests: opts.WarmupRequests,
		Requests:       opts.Requests,
		Concurrency:    opts.Concurrency,
		MaxTokens:      opts.MaxTokens,
		HTTPTimeout:    30 * time.Second,
	}
	if opts.Profile != "" {
		logger.Info("workflow.profile_resolve", "profile", opts.Profile)
		catalog, err := performance.LoadProfileCatalog(filepath.Join(runCtx.RepoRoot, benchmarkProfileConfigRelativePath))
		if err != nil {
			workflowFailure(logger, err)
			fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		profile, err := catalog.ResolveProfile(opts.Profile)
		if err != nil {
			workflowFailure(logger, err)
			fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
			return exitcodes.ACPExitUsage
		}
		runnerOpts.Profile = opts.Profile
		if !opts.WarmupSet {
			runnerOpts.WarmupRequests = profile.Workload.WarmupRequests
		}
		if !opts.RequestsSet {
			runnerOpts.Requests = profile.Workload.MeasuredRequests
		}
		if !opts.ConcurrencySet {
			runnerOpts.Concurrency = profile.Workload.Concurrency
		}
	}

	summary, err := benchmarkRunner(ctx, runnerOpts)
	if err != nil {
		workflowFailure(logger, err)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Fprintln(runCtx.Stderr, "Error: benchmark timed out")
		case errors.Is(err, context.Canceled):
			fmt.Fprintln(runCtx.Stderr, "Error: benchmark canceled")
		default:
			fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		}
		return exitcodes.ACPExitRuntime
	}
	if opts.JSON {
		payload, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			workflowFailure(logger, err)
			fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		fmt.Fprintln(runCtx.Stdout, string(payload))
	} else {
		printBenchmarkSummary(runCtx.Stdout, summary)
	}
	workflowComplete(logger, "failures", summary.Failures, "successes", summary.Successes, "duration", summary.Duration)
	if summary.Failures > 0 {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func printBenchmarkSummary(out *os.File, summary *performance.Summary) {
	terminal := output.New()
	fmt.Fprintln(out, terminal.Bold("Performance baseline summary"))
	fmt.Fprintf(out, "  Gateway: %s\n", summary.GatewayURL)
	fmt.Fprintf(out, "  Model: %s\n", summary.Model)
	if summary.Profile != "" {
		fmt.Fprintf(out, "  Profile: %s\n", summary.Profile)
	}
	fmt.Fprintf(out, "  Warmup requests: %d\n", summary.WarmupRequests)
	fmt.Fprintf(out, "  Requests: %d\n", summary.Requests)
	fmt.Fprintf(out, "  Concurrency: %d\n", summary.Concurrency)
	fmt.Fprintf(out, "  Successes: %d\n", summary.Successes)
	fmt.Fprintf(out, "  Failures: %d\n", summary.Failures)
	fmt.Fprintf(out, "  Duration: %s\n", summary.Duration)
	fmt.Fprintf(out, "  Throughput: %.2f req/s\n", summary.RequestsPerSec)
	fmt.Fprintf(out, "  Average latency: %.2f ms\n", summary.AverageLatencyMS)
	fmt.Fprintf(out, "  p50 latency: %.2f ms\n", summary.P50LatencyMS)
	fmt.Fprintf(out, "  p95 latency: %.2f ms\n", summary.P95LatencyMS)
	fmt.Fprintf(out, "  Min/max latency: %.2f ms / %.2f ms\n", summary.MinLatencyMS, summary.MaxLatencyMS)
}

func runBenchmarkCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	invocation, err := parseInvocation(append([]string{"benchmark"}, args...))
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if len(args) == 0 {
		printCommandHelp(stdout, invocation.Path)
		return exitcodes.ACPExitUsage
	}
	return executeInvocation(ctx, invocation, stdout, stderr)
}
