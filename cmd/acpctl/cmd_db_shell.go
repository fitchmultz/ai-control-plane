// cmd_db_shell.go - Database shell command implementation.
//
// Purpose:
//   - Own the native interactive database shell workflow for embedded and
//     external PostgreSQL modes.
//
// Responsibilities:
//   - Resolve the correct shell target from typed database configuration.
//   - Launch an attached `psql` session without the default subprocess timeout.
//   - Fail with actionable prerequisite guidance for missing tools or config.
//
// Scope:
//   - `acpctl db shell` only.
//
// Usage:
//   - Invoked through `acpctl db shell` or `make db-shell`.
//
// Invariants/Assumptions:
//   - The shell remains an operator-facing interactive command.
//   - Embedded mode targets the repo-local Compose postgres service.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"github.com/mitchfultz/ai-control-plane/pkg/terminal"
)

type dbShellInvocation struct {
	Request proc.Request
	Mode    string
	Target  string
}

var buildDBShellInvocation = defaultBuildDBShellInvocation
var runAttachedSubprocess = proc.RunAttached

func runDBShell(ctx context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	logger := workflowLogger(runCtx, "db_shell")
	workflowStart(logger)

	invocation, err := buildDBShellInvocation(ctx, runCtx)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}

	printDBWorkflowHeader(runCtx.Stdout, out, "=== Database Shell ===", map[string]string{
		"Mode":   invocation.Mode,
		"Target": invocation.Target,
		"Exit":   `\q`,
	})
	fmt.Fprintln(runCtx.Stdout)

	res := runAttachedSubprocess(ctx, invocation.Request)
	if res.Err != nil {
		workflowFailure(logger, res.Err, "mode", invocation.Mode)
		fmt.Fprintln(runCtx.Stderr, out.Fail(res.Err.Error()))
		return proc.ACPExitCode(res.Err)
	}

	workflowComplete(logger, "mode", invocation.Mode)
	return exitcodes.ACPExitSuccess
}

func defaultBuildDBShellInvocation(ctx context.Context, runCtx commandRunContext) (dbShellInvocation, error) {
	settings := config.NewLoader().WithRepoRoot(runCtx.RepoRoot).Database(ctx)
	if settings.AmbiguousErr != nil {
		return dbShellInvocation{}, settings.AmbiguousErr
	}

	switch {
	case settings.Mode.IsExternal():
		if strings.TrimSpace(settings.URL) == "" {
			return dbShellInvocation{}, fmt.Errorf("DATABASE_URL not set for external database mode")
		}
		if err := proc.ValidateExecutable("psql"); err != nil {
			return dbShellInvocation{}, fmt.Errorf("psql is required for external database mode: %w", err)
		}
		return dbShellInvocation{
			Request: proc.Request{
				Name:   "psql",
				Args:   []string{settings.URL},
				Stdin:  os.Stdin,
				Stdout: runCtx.Stdout,
				Stderr: runCtx.Stderr,
			},
			Mode:   settings.Mode.String(),
			Target: settings.URL,
		}, nil
	default:
		if err := checkDBPrereqs(); err != nil {
			return dbShellInvocation{}, err
		}
		compose, err := docker.NewACPCompose(runCtx.RepoRoot, nil)
		if err != nil {
			return dbShellInvocation{}, fmt.Errorf("docker compose not available: %w", err)
		}
		containerID, err := compose.ContainerID(ctx, "postgres")
		if err != nil {
			return dbShellInvocation{}, fmt.Errorf("database container lookup failed: %w", err)
		}

		args := []string{"exec", "-i"}
		if terminal.IsTerminal(os.Stdin) && terminal.IsTerminal(runCtx.Stdout) {
			args = append(args, "-t")
		}
		databaseUser := strings.TrimSpace(settings.User)
		if databaseUser == "" {
			databaseUser = "litellm"
		}
		databaseName := strings.TrimSpace(settings.Name)
		if databaseName == "" {
			databaseName = "litellm"
		}
		args = append(args, containerID, "psql", "-X", "-U", databaseUser, "-d", databaseName)

		return dbShellInvocation{
			Request: proc.Request{
				Name:   "docker",
				Args:   args,
				Stdin:  os.Stdin,
				Stdout: runCtx.Stdout,
				Stderr: runCtx.Stderr,
			},
			Mode:   settings.Mode.String(),
			Target: "postgres container",
		}, nil
	}
}

func runDBShellCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "shell"}, args, stdout, stderr)
}
