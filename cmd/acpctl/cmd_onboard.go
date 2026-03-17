// cmd_onboard.go - Onboarding command implementation.
//
// Purpose:
//   - Expose the guided typed onboarding workflow for supported local tools.
//
// Responsibilities:
//   - Define the onboarding command surface.
//   - Bind optional tool preselection into `internal/onboard.Options`.
//   - Delegate execution to `internal/onboard`.
//
// Scope:
//   - CLI integration only; workflow logic lives in internal/onboard.
//
// Usage:
//   - Invoked through `acpctl onboard`.
//
// Invariants/Assumptions:
//   - Exit codes follow the ACP contract.
//   - The command reads demo/.env as data, never as sourced shell.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/onboard"
)

func onboardCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "onboard",
		Summary:     "Launch the guided onboarding wizard",
		Description: "Interactive guided setup for supported local tools that route through the AI Control Plane gateway.",
		Examples: []string{
			"acpctl onboard",
			"acpctl onboard codex",
			"make onboard",
			"make onboard-codex",
		},
		Arguments: []commandArgumentSpec{{Name: "tool", Summary: "Optional tool name to preselect in the wizard"}},
		Sections: []commandHelpSection{
			{
				Title: "Supported tools",
				Lines: []string{"codex", "claude", "opencode", "cursor"},
			},
			{
				Title: "Notes",
				Lines: []string{
					"Run `acpctl onboard` for the full wizard.",
					"Run `acpctl onboard codex` to skip only the tool-selection step.",
					"Legacy onboarding flags were removed; answer the prompts instead.",
				},
			},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (onboard.Options, error) {
				return onboard.Options{RepoRoot: bindCtx.RepoRoot, Tool: input.NormalizedArgument(0)}, nil
			}),
			NativeRun: runOnboard,
		},
	}
}

func runOnboard(ctx context.Context, runCtx commandRunContext, raw any) int {
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "onboard")))
	opts := raw.(onboard.Options)
	opts.Stdin = os.Stdin
	opts.Stdout = runCtx.Stdout
	opts.Stderr = runCtx.Stderr
	result := onboard.Run(ctx, opts)
	onboard.WriteHuman(runCtx.Stdout, runCtx.Stderr, result)
	return result.ExitCode
}

func runOnboardCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"onboard"}, args, stdout, stderr)
}
