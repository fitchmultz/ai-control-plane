// cmd_smoke.go - Runtime smoke gate command implementation.
//
// Purpose:
//   - Run truthful runtime smoke validation for operator-facing production gates.
//
// Responsibilities:
//   - Reuse the canonical runtime inspection stack for smoke checks.
//   - Enforce gateway auth, gateway/model reachability, and database readiness.
//   - Return ACP exit codes that reflect real runtime state.
//
// Scope:
//   - File-local smoke command parsing, execution, and output.
//
// Usage:
//   - Invoked via `acpctl smoke` and make targets that delegate to it.
//
// Invariants/Assumptions:
//   - Smoke is a real gate and must not silently pass on warnings or bad inputs.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/runtimeinspect"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const smokeCommandTimeout = 30 * time.Second

type smokeInspector interface {
	Collect(ctx context.Context, opts status.Options) status.StatusReport
	Close() error
}

var newSmokeInspector = func(repoRoot string) smokeInspector {
	return runtimeinspect.NewInspector(repoRoot)
}

type smokeOptions struct {
	Verbose bool
}

func runSmokeTestCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	options, err := parseSmokeArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		printSmokeTestHelp(stderr)
		return exitcodes.ACPExitUsage
	}
	if options == nil {
		printSmokeTestHelp(stdout)
		return exitcodes.ACPExitSuccess
	}

	out := output.New()
	if !prereq.CommandExists("docker") {
		fmt.Fprintln(stderr, out.Fail("Docker not found"))
		fmt.Fprintln(stderr, "Install Docker from https://docs.docker.com/get-docker/")
		return exitcodes.ACPExitPrereq
	}

	repoRoot := detectRepoRootWithContext(ctx)
	if repoRoot == "" {
		fmt.Fprintln(stderr, out.Fail("Failed to detect repository root"))
		return exitcodes.ACPExitRuntime
	}

	inspector := newSmokeInspector(repoRoot)
	defer inspector.Close()

	smokeCtx, cancel := context.WithTimeout(ctx, smokeCommandTimeout)
	defer cancel()

	report := inspector.Collect(smokeCtx, status.Options{
		RepoRoot: repoRoot,
		Wide:     options.Verbose,
	})

	fmt.Fprintln(stdout, out.Bold("=== Runtime Smoke Checks ==="))
	if err := report.WriteHuman(stdout, options.Verbose); err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to render smoke output: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if errors.Is(smokeCtx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(stderr, out.Fail("Smoke check timed out"))
		return exitcodes.ACPExitRuntime
	}
	if errors.Is(smokeCtx.Err(), context.Canceled) {
		fmt.Fprintln(stderr, out.Fail("Smoke check canceled"))
		return exitcodes.ACPExitRuntime
	}

	readiness := runtimeinspect.EvaluateReadiness(report, runtimeinspect.DefaultReadinessComponents)
	if !readiness.Ready {
		fmt.Fprintln(stderr, out.Fail("Runtime smoke failed: required components are not ready"))
		for _, name := range readiness.Missing {
			component, ok := readiness.Pending[name]
			switch {
			case ok && strings.TrimSpace(component.Message) != "":
				fmt.Fprintf(stderr, "  - %s: %s\n", name, component.Message)
			default:
				fmt.Fprintf(stderr, "  - %s: not ready\n", name)
			}
		}
		return exitcodes.ACPExitDomain
	}

	switch report.Overall {
	case status.HealthLevelHealthy:
		fmt.Fprintln(stdout, out.Green("Runtime smoke checks passed"))
		return exitcodes.ACPExitSuccess
	case status.HealthLevelWarning, status.HealthLevelUnhealthy:
		fmt.Fprintln(stderr, out.Fail("Runtime smoke checks failed"))
		return exitcodes.ACPExitDomain
	default:
		fmt.Fprintln(stderr, out.Fail("Runtime smoke returned unknown status"))
		return exitcodes.ACPExitRuntime
	}
}

func parseSmokeArgs(args []string) (*smokeOptions, error) {
	options := &smokeOptions{}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return nil, nil
		case "--verbose", "-v":
			options.Verbose = true
		default:
			return nil, fmt.Errorf("unknown option: %s", arg)
		}
	}
	return options, nil
}

func printSmokeTestHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl smoke [OPTIONS]

Run truthful runtime smoke checks against the active ACP deployment.

Checks:
  - Docker prerequisite availability
  - Gateway health endpoint reachability
  - Authorized /v1/models access using LITELLM_MASTER_KEY
  - Database readiness
  - Canonical runtime readiness for required components

Options:
  --verbose, -v     Enable detailed output
  --help, -h        Show this help message

Environment variables:
  GATEWAY_HOST        Gateway host (default: 127.0.0.1)
  LITELLM_PORT        Gateway port (default: 4000)
  LITELLM_MASTER_KEY  Master key for authorized gateway checks (required)
  ACP_DATABASE_MODE   Database mode: embedded|external (default: embedded)

Examples:
  acpctl smoke
  acpctl smoke --verbose

Exit codes:
  0   Tests passed
  1   Tests failed
  2   Prerequisites not ready
  3   Runtime/internal error
  64  Usage error
`)
}
