// cmd_onboard.go - Onboarding command implementation.
//
// Purpose:
//   - Expose typed onboarding workflows for local tools and IDE integrations.
//
// Responsibilities:
//   - Define the typed onboarding command surface.
//   - Bind typed CLI options into `internal/onboard.Options`.
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
		Summary:     "Configure local tools to route through the gateway",
		Description: "Configure local tools to route through the gateway.",
		Examples: []string{
			"acpctl onboard codex --mode subscription --verify",
			"acpctl onboard codex --mode direct --write-config",
			"acpctl onboard --help",
		},
		Arguments: []commandArgumentSpec{
			{Name: "tool", Summary: "Tool to onboard"},
		},
		Options: []commandOptionSpec{
			{Name: "mode", ValueName: "MODE", Summary: "Onboarding mode", Type: optionValueString},
			{Name: "alias", ValueName: "ALIAS", Summary: "Generated key alias override", Type: optionValueString},
			{Name: "budget", ValueName: "USD", Summary: "Budget override", Type: optionValueString, DefaultText: onboard.DefaultBudget},
			{Name: "model", ValueName: "MODEL", Summary: "Model override", Type: optionValueString},
			{Name: "host", ValueName: "HOST", Summary: "Gateway host override", Type: optionValueString, DefaultText: onboard.DefaultHost},
			{Name: "port", ValueName: "PORT", Summary: "Gateway port override", Type: optionValueString, DefaultText: onboard.DefaultPort},
			{Name: "tls", Summary: "Use TLS when constructing gateway URLs", Type: optionValueBool},
			{Name: "verify", Summary: "Verify subscription or gateway readiness", Type: optionValueBool},
			{Name: "write-config", Summary: "Write ACP-managed Codex configuration", Type: optionValueBool},
			{Name: "show-key", Summary: "Display generated key material", Type: optionValueBool},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (onboard.Options, error) {
				return onboard.Options{
					RepoRoot:    bindCtx.RepoRoot,
					Tool:        input.NormalizedArgument(0),
					Mode:        input.NormalizedString("mode"),
					Alias:       input.NormalizedString("alias"),
					Budget:      input.NormalizedString("budget"),
					Model:       input.NormalizedString("model"),
					Host:        input.NormalizedString("host"),
					Port:        input.NormalizedString("port"),
					UseTLS:      input.Bool("tls"),
					Verify:      input.Bool("verify"),
					WriteConfig: input.Bool("write-config"),
					ShowKey:     input.Bool("show-key"),
				}, nil
			}),
			NativeRun: runOnboard,
		},
	}
}

func runOnboard(ctx context.Context, runCtx commandRunContext, raw any) int {
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "onboard")))
	result := onboard.Run(ctx, raw.(onboard.Options))
	onboard.WriteHuman(runCtx.Stdout, runCtx.Stderr, result)
	return result.ExitCode
}

func runOnboardCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"onboard"}, args, stdout, stderr)
}
