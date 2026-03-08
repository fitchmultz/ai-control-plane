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
	"strconv"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/performance"
)

var benchmarkRunner = performance.RunBaseline

const benchmarkProfileConfigRelativePath = "demo/config/benchmark_thresholds.json"

func runBenchmarkCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		printBenchmarkHelp(stdout)
		if len(args) == 0 {
			return exitcodes.ACPExitUsage
		}
		return exitcodes.ACPExitSuccess
	}
	if args[0] != "baseline" {
		fmt.Fprintf(stderr, "Error: unknown benchmark command: %s\n", args[0])
		printBenchmarkHelp(stderr)
		return exitcodes.ACPExitUsage
	}
	return runBenchmarkBaseline(ctx, args[1:], stdout, stderr)
}

func runBenchmarkBaseline(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	gatewayRuntime := config.NewLoader().Gateway(true)
	opts := performance.BaselineOptions{
		GatewayURL:     "http://127.0.0.1:4000",
		MasterKey:      gatewayRuntime.MasterKey,
		Model:          "mock-gpt",
		Prompt:         "Provide a short response for performance baseline verification.",
		WarmupRequests: 0,
		Requests:       20,
		Concurrency:    2,
		MaxTokens:      32,
		HTTPTimeout:    30 * time.Second,
	}
	jsonOutput := false
	profileName := ""
	requestsSet := false
	concurrencySet := false
	warmupSet := false
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--gateway-url":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --gateway-url")
				return exitcodes.ACPExitUsage
			}
			opts.GatewayURL = args[index]
		case "--master-key":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --master-key")
				return exitcodes.ACPExitUsage
			}
			opts.MasterKey = args[index]
		case "--model":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --model")
				return exitcodes.ACPExitUsage
			}
			opts.Model = args[index]
		case "--profile":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --profile")
				return exitcodes.ACPExitUsage
			}
			profileName = args[index]
		case "--prompt":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --prompt")
				return exitcodes.ACPExitUsage
			}
			opts.Prompt = args[index]
		case "--requests":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --requests")
				return exitcodes.ACPExitUsage
			}
			value, err := strconv.Atoi(args[index])
			if err != nil {
				fmt.Fprintf(stderr, "Error: invalid --requests value: %v\n", err)
				return exitcodes.ACPExitUsage
			}
			opts.Requests = value
			requestsSet = true
		case "--concurrency":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --concurrency")
				return exitcodes.ACPExitUsage
			}
			value, err := strconv.Atoi(args[index])
			if err != nil {
				fmt.Fprintf(stderr, "Error: invalid --concurrency value: %v\n", err)
				return exitcodes.ACPExitUsage
			}
			opts.Concurrency = value
			concurrencySet = true
		case "--warmup-requests":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --warmup-requests")
				return exitcodes.ACPExitUsage
			}
			value, err := strconv.Atoi(args[index])
			if err != nil {
				fmt.Fprintf(stderr, "Error: invalid --warmup-requests value: %v\n", err)
				return exitcodes.ACPExitUsage
			}
			opts.WarmupRequests = value
			warmupSet = true
		case "--max-tokens":
			index++
			if index >= len(args) {
				fmt.Fprintln(stderr, "Error: missing value for --max-tokens")
				return exitcodes.ACPExitUsage
			}
			value, err := strconv.Atoi(args[index])
			if err != nil {
				fmt.Fprintf(stderr, "Error: invalid --max-tokens value: %v\n", err)
				return exitcodes.ACPExitUsage
			}
			opts.MaxTokens = value
		case "--json":
			jsonOutput = true
		case "--help", "-h":
			printBenchmarkBaselineHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", args[index])
			return exitcodes.ACPExitUsage
		}
	}

	if profileName != "" {
		catalog, err := performance.LoadProfileCatalog(filepath.Join(detectRepoRootWithContext(ctx), benchmarkProfileConfigRelativePath))
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		profile, err := catalog.ResolveProfile(profileName)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return exitcodes.ACPExitUsage
		}
		opts.Profile = profileName
		if !warmupSet {
			opts.WarmupRequests = profile.Workload.WarmupRequests
		}
		if !requestsSet {
			opts.Requests = profile.Workload.MeasuredRequests
		}
		if !concurrencySet {
			opts.Concurrency = profile.Workload.Concurrency
		}
	}

	summary, err := benchmarkRunner(ctx, opts)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			fmt.Fprintln(stderr, "Error: benchmark timed out")
		case errors.Is(err, context.Canceled):
			fmt.Fprintln(stderr, "Error: benchmark canceled")
		default:
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return exitcodes.ACPExitRuntime
	}
	if jsonOutput {
		payload, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return exitcodes.ACPExitRuntime
		}
		fmt.Fprintln(stdout, string(payload))
	} else {
		printBenchmarkSummary(stdout, summary)
	}
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

func printBenchmarkHelp(out *os.File) {
	command := mustLookupNativeCommand("benchmark")

	fmt.Fprint(out, `Usage: acpctl benchmark <command> [OPTIONS]

Commands:
`)
	for _, subcommand := range command.Subcommands {
		fmt.Fprintf(out, "  %-10s %s\n", subcommand.Name, subcommand.Description)
	}
	fmt.Fprint(out, `

Examples:
  acpctl benchmark baseline
  acpctl benchmark baseline --profile interactive
  acpctl benchmark baseline --requests 40 --concurrency 4
  acpctl benchmark baseline --json

Exit codes:
  0   Success
  1   Domain non-success (one or more benchmark requests failed)
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}

func printBenchmarkBaselineHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl benchmark baseline [OPTIONS]

Options:
  --gateway-url URL      Gateway base URL (default: http://127.0.0.1:4000)
  --master-key VALUE     Gateway master key (default: LITELLM_MASTER_KEY env var)
  --model NAME           Model alias to exercise (default: mock-gpt)
  --profile NAME         Benchmark profile from demo/config/benchmark_thresholds.json
  --prompt TEXT          Prompt body to send (default: short fixed prompt)
  --warmup-requests N    Warmup requests excluded from measured results (default: 0)
  --requests N           Total request count (default: 20)
  --concurrency N        Number of concurrent workers (default: 2)
  --max-tokens N         max_tokens for each request (default: 32)
  --json                 Emit machine-readable JSON to stdout
  --help                 Show this help message

Notes:
  - This command produces a local reference-host baseline, not customer-grade capacity proof.
  - Use offline-safe models such as mock-gpt or mock-claude for deterministic local runs.
  - Profile names currently include interactive, burst, and sustained.
`)
}
