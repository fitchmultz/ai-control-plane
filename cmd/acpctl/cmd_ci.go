// cmd_ci.go - CI subcommand implementation
//
// Purpose: Implement CI-related commands for runtime scope decisions
// Responsibilities:
//   - Parse CI subcommand flags
//   - Execute should-run-runtime decision logic
//
// Non-scope:
//   - Does not execute runtime checks directly
//   - Does not modify git state
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/ci"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type ciShouldRunRuntimeOptions struct {
	Paths []string
	Quiet bool
}

func ciCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "ci",
		Summary:     "CI and local gate helpers",
		Description: "CI and local gate helpers.",
		Examples: []string{
			"acpctl ci should-run-runtime --quiet",
			"acpctl ci wait --timeout 120",
		},
		Children: []*commandSpec{
			{
				Name:        "should-run-runtime",
				Summary:     "Decide whether runtime checks should run",
				Description: "Determine whether CI runtime checks should run for the current change set.",
				Examples: []string{
					"acpctl ci should-run-runtime --quiet",
					"acpctl ci should-run-runtime --path docs/README.md --path mk/runtime.mk",
				},
				Options: []commandOptionSpec{
					{Name: "path", ValueName: "PATH", Summary: "Add a changed path explicitly", Type: optionValueString, Repeatable: true},
					{Name: "quiet", Short: "q", Summary: "Print no informational output", Type: optionValueBool},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindCIShouldRunRuntimeOptions,
					NativeRun:  runCIShouldRunRuntime,
				},
			},
			ciWaitCommandSpec(),
		},
	}
}

func bindCIShouldRunRuntimeOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	opts := ciShouldRunRuntimeOptions{
		Paths: input.Strings("path"),
		Quiet: input.Bool("quiet"),
	}
	if err := ci.ValidateDecisionArgs(opts.Paths); err != nil {
		return nil, err
	}
	return opts, nil
}

func runCIShouldRunRuntime(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(ciShouldRunRuntimeOptions)
	decisionCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := ci.DecideRuntimeScope(decisionCtx, ci.DecisionOptions{
		RepoRoot: runCtx.RepoRoot,
		Paths:    opts.Paths,
		CIFull:   config.NewLoader().Tooling().CIFull,
	})
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: runtime scope decision failed: %v\n", err)
		return exitcodes.ACPExitRuntime
	}

	if !opts.Quiet && result.Reason != "" {
		fmt.Fprintln(runCtx.Stdout, result.Reason)
	}

	if result.ShouldRun {
		return exitcodes.ACPExitSuccess
	}
	return exitcodes.ACPExitDomain
}
