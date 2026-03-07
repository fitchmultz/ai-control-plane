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
	"errors"
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

type ciWaitCompose interface {
	PS(ctx context.Context) (string, error)
}

type ciWaitGateway interface {
	Health(ctx context.Context) (bool, int, error)
	HasMasterKey() bool
}

var newCIWaitCompose = func(repoRoot string) (ciWaitCompose, error) {
	projectName := strings.TrimSpace(os.Getenv("ACP_COMPOSE_PROJECT"))
	if projectName == "" {
		slot := strings.TrimSpace(os.Getenv("ACP_SLOT"))
		if slot == "" {
			slot = "active"
		}
		projectName = "ai-control-plane-" + slot
	}
	return docker.NewComposeWithOptions(docker.DefaultProjectDir(repoRoot), docker.ComposeOptions{
		ProjectName: projectName,
		Files:       []string{"docker-compose.offline.yml"},
	})
}

var newCIWaitGateway = func() ciWaitGateway {
	return gateway.NewClient()
}

func runCIWaitCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	timeout := 120 * time.Second
	interval := 5 * time.Second
	verbose := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--timeout":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Invalid --timeout value")
				return exitcodes.ACPExitUsage
			}
			t, err := strconv.Atoi(args[i+1])
			if err != nil || t <= 0 {
				fmt.Fprintf(stderr, "Invalid --timeout value: '%s' (must be a positive integer)\n", args[i+1])
				return exitcodes.ACPExitUsage
			}
			timeout = time.Duration(t) * time.Second
			i++
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

	compose, err := newCIWaitCompose(detectRepoRootWithContext(ctx))
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Docker Compose not available: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	gw := newCIWaitGateway()
	if !gw.HasMasterKey() {
		fmt.Fprintln(stderr, out.Fail("LITELLM_MASTER_KEY is required for authorized gateway health checks"))
		fmt.Fprintln(stderr, "Set LITELLM_MASTER_KEY in your environment or demo/.env")
		return exitcodes.ACPExitUsage
	}

	fmt.Fprintln(stdout, out.Bold("Waiting for services to become healthy..."))
	if verbose {
		fmt.Fprintf(stdout, "Timeout: %s, Check interval: %s\n", timeout, interval)
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	postgresHealthy := false
	litellmHealthy := false
	litellmAPIReady := false

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	probe := func() {
		ps, _ := compose.PS(waitCtx)

		if strings.Contains(ps, "postgres") && strings.Contains(ps, "healthy") {
			if !postgresHealthy {
				fmt.Fprintln(stdout, out.Pass("PostgreSQL is healthy"))
				postgresHealthy = true
			}
		} else if verbose {
			fmt.Fprintln(stdout, "  PostgreSQL not healthy yet")
		}

		if strings.Contains(ps, "litellm") && strings.Contains(ps, "healthy") {
			if !litellmHealthy {
				fmt.Fprintln(stdout, out.Pass("LiteLLM is healthy"))
				litellmHealthy = true
			}
		} else if verbose {
			fmt.Fprintln(stdout, "  LiteLLM not healthy yet")
		}

		if !litellmAPIReady {
			healthy, _, err := gw.Health(waitCtx)
			if err == nil && healthy {
				fmt.Fprintln(stdout, out.Pass("LiteLLM API is responding (authorized HTTP 200)"))
				litellmAPIReady = true
			} else if verbose {
				fmt.Fprintln(stdout, "  LiteLLM API not responding yet")
			}
		}
	}

	for {
		probe()
		if postgresHealthy && litellmHealthy && litellmAPIReady {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout, out.Green("All services are healthy and ready"))
			return exitcodes.ACPExitSuccess
		}

		select {
		case <-waitCtx.Done():
			fmt.Fprintln(stdout)
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				fmt.Fprintln(stderr, out.Fail("CI wait canceled"))
				return exitcodes.ACPExitRuntime
			}
			fmt.Fprintf(stdout, out.Fail("Timeout: Services did not become healthy within %s\n"), timeout)
			statusCtx, statusCancel := context.WithTimeout(ctx, 2*time.Second)
			defer statusCancel()
			if ps, err := compose.PS(statusCtx); err == nil && strings.TrimSpace(ps) != "" {
				fmt.Fprintln(stdout)
				fmt.Fprintln(stdout, "Current container status:")
				fmt.Fprintln(stdout, ps)
			}
			return exitcodes.ACPExitDomain
		case <-ticker.C:
			continue
		}
	}
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
  3   Runtime/internal error (including cancellation)
  64  Usage error
`)
}
