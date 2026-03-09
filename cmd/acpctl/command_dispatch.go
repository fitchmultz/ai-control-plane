// command_dispatch.go - Runtime dispatch for compiled acpctl commands.
//
// Purpose:
//
//	Construct the runtime command context and dispatch parsed invocations to
//	native, Make-backed, or bridge-backed backends.
//
// Responsibilities:
//   - Seed the command runtime logger and repo root.
//   - Bind native command inputs before execution.
//   - Route non-native commands through canonical adapters.
//
// Scope:
//   - Execution-time dispatch only.
//
// Usage:
//   - Called by main.go and typed command adapters after parsing.
//
// Invariants/Assumptions:
//   - Structured workflow logs go to stderr.
//   - Final human render output remains owned by command handlers.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

func executeInvocation(ctx context.Context, invocation commandInvocation, stdout *os.File, stderr *os.File) int {
	if invocation.HelpOnly {
		printCommandHelp(stdout, invocation.Path)
		return exitcodes.ACPExitSuccess
	}
	runCtx := commandRunContext{
		RepoRoot: detectRepoRootWithContext(ctx),
		Stdout:   stdout,
		Stderr:   stderr,
		Logger:   buildCommandLogger(stderr, invocation.Path),
	}
	switch invocation.Spec.Backend.Kind {
	case commandBackendNative:
		opts, err := invocation.Spec.Backend.NativeBind(commandBindContext{RepoRoot: runCtx.RepoRoot}, invocation.Input)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			printCommandHelp(stderr, invocation.Path)
			return exitcodes.ACPExitUsage
		}
		ctx = logging.WithLogger(ctx, runCtx.Logger)
		return invocation.Spec.Backend.NativeRun(ctx, runCtx, opts)
	case commandBackendMake:
		return runMakeTarget(ctx, invocation.Spec.Backend.MakeTarget, invocation.Input.Trailing(), stdout, stderr)
	case commandBackendBridge:
		return runBridgeScript(
			ctx,
			invocation.Spec.Backend.BridgeRelativePath,
			invocation.Spec.Name,
			invocation.Spec.Backend.BridgeArgs,
			invocation.Input.Trailing(),
			stdout,
			stderr,
		)
	default:
		fmt.Fprintln(stderr, "Error: invalid command backend")
		return exitcodes.ACPExitRuntime
	}
}

func buildCommandLogger(stderr *os.File, path []*commandSpec) *slog.Logger {
	attrs := []slog.Attr{
		slog.String("component", "acpctl"),
		slog.String("command", commandPathLabel(path[1:])),
	}
	return logging.New(logging.Options{
		Writer: stderr,
		Format: logging.FormatText,
		Level:  slog.LevelInfo,
		Attrs:  attrs,
	}).With(slog.String("command_path", strings.TrimSpace(commandPathKey(path[1:]))))
}
