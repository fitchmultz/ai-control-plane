// cmd_ci_wait.go - CI wait for health command
//
// Purpose: Wait for Docker services to be healthy before proceeding
//
// Responsibilities:
//   - Wait for Compose services to report healthy status
//   - Verify LiteLLM /health endpoint responds successfully
//   - Enforce configurable timeout and interval polling
//
// Non-scope:
//   - Does NOT start or create containers
//   - Does NOT run full health suite

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

func runCIWaitCommand(args []string, stdout *os.File, stderr *os.File) int {
	timeout := 120
	interval := 5
	verbose := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--timeout":
			if i+1 < len(args) {
				t, err := strconv.Atoi(args[i+1])
				if err != nil || t <= 0 {
					fmt.Fprintf(stderr, "Invalid --timeout value: '%s' (must be a positive integer)\n", args[i+1])
					return exitcodes.ACPExitUsage
				}
				timeout = t
				i++
			}
		case "--verbose", "-v":
			verbose = true
		case "--help", "-h":
			printCIWaitHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "Unknown option: %s\n", args[i])
				return exitcodes.ACPExitUsage
			}
		}
	}

	out := output.New()

	// Prerequisite checks
	if !prereq.CommandExists("docker") {
		fmt.Fprintln(stderr, out.Fail("docker not found"))
		return exitcodes.ACPExitPrereq
	}

	// Detect Docker Compose
	compose, err := docker.NewCompose(docker.DefaultProjectDir(detectRepoRoot()))
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	// Create gateway client for API checks
	gw := gateway.NewClient()
	if !gw.HasMasterKey() {
		fmt.Fprintln(stderr, out.Fail("LITELLM_MASTER_KEY is required for authorized gateway health checks"))
		fmt.Fprintln(stderr, "Set LITELLM_MASTER_KEY in your environment or demo/.env")
		return exitcodes.ACPExitUsage
	}

	fmt.Fprintln(stdout, out.Bold("Waiting for services to become healthy..."))
	if verbose {
		fmt.Fprintf(stdout, "Timeout: %ds, Check interval: %ds\n", timeout, interval)
	}

	startTime := time.Now()
	postgresHealthy := false
	litellmHealthy := false
	litellmAPIReady := false

	ctx := context.Background()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		elapsed := int(time.Since(startTime).Seconds())
		if elapsed >= timeout {
			break
		}

		// Get container status
		ps, _ := compose.PS(ctx)

		// Check PostgreSQL health
		if strings.Contains(ps, "postgres") && strings.Contains(ps, "healthy") {
			if !postgresHealthy {
				fmt.Fprintln(stdout, out.Pass("PostgreSQL is healthy"))
				postgresHealthy = true
			}
		} else if verbose {
			fmt.Fprintln(stdout, "  PostgreSQL not healthy yet")
		}

		// Check LiteLLM container health
		if strings.Contains(ps, "litellm") && strings.Contains(ps, "healthy") {
			if !litellmHealthy {
				fmt.Fprintln(stdout, out.Pass("LiteLLM is healthy"))
				litellmHealthy = true
			}
		} else if verbose {
			fmt.Fprintln(stdout, "  LiteLLM not healthy yet")
		}

		// Check LiteLLM API health endpoint
		if !litellmAPIReady {
			healthy, _, err := gw.Health(ctx)
			if err == nil && healthy {
				fmt.Fprintln(stdout, out.Pass("LiteLLM API is responding (authorized HTTP 200)"))
				litellmAPIReady = true
			} else if verbose {
				fmt.Fprintln(stdout, "  LiteLLM API not responding yet")
			}
		}

		// Check if all critical services are ready
		if postgresHealthy && litellmHealthy && litellmAPIReady {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, out.Green("All services are healthy and ready"))
			return exitcodes.ACPExitSuccess
		}

		// Progress indicator (every 10 seconds if not verbose)
		if !verbose && elapsed%10 == 0 && elapsed > 0 {
			fmt.Fprintf(stdout, "  Waiting... (%ds/%ds)\n", elapsed, timeout)
		}

		// Wait for next tick or timeout
		select {
		case <-ticker.C:
			continue
		case <-time.After(time.Duration(timeout-elapsed) * time.Second):
			break
		}
	}

	// Timeout reached
	fmt.Fprintln(stdout, "")
	fmt.Fprintf(stdout, out.Fail("Timeout: Services did not become healthy within %d seconds\n"), timeout)
	fmt.Fprintln(stdout, "")

	// Show current status for debugging
	fmt.Fprintln(stdout, "Current container status:")
	ps, _ := compose.PS(ctx)
	fmt.Fprintln(stdout, ps)

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Recent logs:")
	// Note: We'd need to implement log retrieval in docker package
	fmt.Fprintln(stdout, "(Use 'docker-compose logs --tail 20' to view recent logs)")

	return exitcodes.ACPExitDomain
}

func printCIWaitHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl ci wait [OPTIONS]

Wait for Docker services to be healthy before proceeding.
Exits with error if services don't become healthy within timeout.

Options:
  --timeout SECONDS  Maximum time to wait (default: 120)
  --verbose, -v      Enable verbose output
  --help, -h         Show this help message

Environment variables:
  LITELLM_MASTER_KEY  Master key for authorized gateway checks (required)

Examples:
  acpctl ci wait              # Wait with default timeout (120 seconds)
  acpctl ci wait --timeout 60 # Wait with 60 second timeout
  acpctl ci wait --verbose    # Verbose mode for debugging

Exit codes:
  0   All services healthy
  1   Timeout or services unhealthy
  2   Prerequisites not ready
  64  Usage error
`)
}
