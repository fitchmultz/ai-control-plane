// cmd_health.go - Health check command implementation
//
// Purpose: Provide native Go implementation of health checks
//
// Responsibilities:
//   - Check Docker container status
//   - Verify LiteLLM gateway endpoints
//   - Check PostgreSQL connectivity
//   - Check OTEL collector status
//
// Non-scope:
//   - Does not start services
//   - Does not fix issues

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/health"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

const healthCommandTimeout = 30 * time.Second

func runHealthCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	// Parse arguments
	verbose := false
	for _, arg := range args {
		switch arg {
		case "--verbose", "-v":
			verbose = true
		case "--help", "-h":
			printHealthHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	out := output.New()

	// Check prerequisites
	if !prereq.CommandExists("docker") {
		fmt.Fprintln(stderr, out.Fail("Docker not found"))
		fmt.Fprintln(stderr, "Install Docker from https://docs.docker.com/get-docker/")
		return exitcodes.ACPExitPrereq
	}

	// Detect Docker Compose
	compose, err := docker.NewCompose(docker.DefaultProjectDir(detectRepoRootWithContext(ctx)))
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	// Run health checks
	ctx, cancel := context.WithTimeout(ctx, healthCommandTimeout)
	defer cancel()
	checker := health.NewChecker(compose, verbose)
	result := checker.Run(ctx)

	// Print summary
	checker.PrintSummary(result)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(stderr, out.Fail("Health check timed out"))
		return exitcodes.ACPExitRuntime
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		fmt.Fprintln(stderr, out.Fail("Health check canceled"))
		return exitcodes.ACPExitRuntime
	}

	// Return appropriate exit code
	switch result.Overall {
	case health.StatusHealthy:
		return exitcodes.ACPExitSuccess
	case health.StatusUnhealthy:
		return exitcodes.ACPExitDomain
	default:
		return exitcodes.ACPExitRuntime
	}
}

func printHealthHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl health [OPTIONS]

Run health checks for AI Control Plane services.

Checks:
  - Docker container status (postgres, litellm)
  - LiteLLM gateway health endpoint
  - LiteLLM models endpoint
  - PostgreSQL connectivity and schema
  - OTEL collector status (optional)

Options:
  --verbose, -v     Enable detailed output
  --help, -h        Show this help message

Environment variables:
  GATEWAY_HOST      Gateway host (default: 127.0.0.1)
  LITELLM_PORT      LiteLLM port (default: 4000)
  LITELLM_MASTER_KEY  Master key for authorized gateway checks (required)
  ACP_DATABASE_MODE Database mode: embedded|external (default: embedded)

Examples:
  acpctl health              # Run health checks
  acpctl health --verbose    # Run with detailed output

Exit codes:
  0   All required services healthy
  1   One or more required services unhealthy
  2   Prerequisites not ready
  3   Runtime/internal error (including timeout or cancellation)
  64  Usage error
`)
}
