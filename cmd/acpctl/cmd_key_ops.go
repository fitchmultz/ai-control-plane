// cmd_key_ops.go - Key operations command implementation
//
// Purpose: Provide native Go implementation of key generation and management.
//
// Responsibilities:
//   - Define the typed `key` command tree and binders.
//   - Validate key options using internal/keygen helpers.
//   - Execute key generation via the gateway client.
//
// Non-scope:
//   - Does not own business validation rules.
//   - Does not implement revoke support in the gateway.
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
	"os"

	acpconfig "github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

type keyGenOptions struct {
	Alias    string
	Budget   float64
	RPM      int
	TPM      int
	Parallel int
	Duration string
	Role     string
	DryRun   bool
}

type keyRevokeOptions struct {
	Alias string
}

func keyCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "key",
		Summary:     "Virtual key lifecycle operations",
		Description: "Virtual key lifecycle operations.",
		Examples: []string{
			"acpctl key gen alice --budget 10.00",
			"acpctl key revoke alice",
		},
		Children: []*commandSpec{
			keyGenCommandSpec("gen", "Generate a standard virtual key", ""),
			{
				Name:        "revoke",
				Summary:     "Revoke a virtual key by alias",
				Description: "Revoke a virtual key by alias.",
				Arguments: []commandArgumentSpec{
					{Name: "alias", Summary: "Key alias to revoke", Required: true},
				},
				Backend: commandBackend{
					Kind:       commandBackendNative,
					NativeBind: bindKeyRevokeOptions,
					NativeRun:  runKeyRevoke,
				},
			},
			keyGenCommandSpec("gen-dev", "Generate a developer key", "developer"),
			keyGenCommandSpec("gen-lead", "Generate a team-lead key", "team-lead"),
		},
	}
}

func keyGenCommandSpec(name string, summary string, forcedRole string) *commandSpec {
	options := []commandOptionSpec{
		{Name: "budget", ValueName: "BUDGET", Summary: "Maximum budget in USD", Type: optionValueFloat, DefaultText: "10.00"},
		{Name: "rpm", ValueName: "RPM", Summary: "Requests per minute limit", Type: optionValueInt},
		{Name: "tpm", ValueName: "TPM", Summary: "Tokens per minute limit", Type: optionValueInt},
		{Name: "parallel", ValueName: "N", Summary: "Max parallel requests", Type: optionValueInt},
		{Name: "duration", ValueName: "DUR", Summary: "Budget reset duration", Type: optionValueString, DefaultText: "30d"},
		{Name: "dry-run", Summary: "Preview the request without executing", Type: optionValueBool},
	}
	if forcedRole == "" {
		options = append(options, commandOptionSpec{
			Name:        "role",
			ValueName:   "ROLE",
			Summary:     "Role for key generation",
			Type:        optionValueString,
			DefaultText: "developer",
			Suggestions: func(string) []string { return keygen.ValidRoles() },
		})
	}
	return &commandSpec{
		Name:        name,
		Summary:     summary,
		Description: summary + ".",
		Arguments: []commandArgumentSpec{
			{Name: "alias", Summary: "Key alias/name", Required: true},
		},
		Options: options,
		Sections: []commandHelpSection{
			{
				Title: "Environment",
				Lines: []string{
					"LITELLM_MASTER_KEY",
					"GATEWAY_HOST",
					"LITELLM_PORT",
				},
			},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
				return bindKeyGenOptions(input, forcedRole)
			},
			NativeRun: runKeyGen,
		},
	}
}

func bindKeyGenOptions(input parsedCommandInput, forcedRole string) (any, error) {
	alias := input.Argument(0)
	if alias == "" {
		return nil, fmt.Errorf("alias is required")
	}
	budget, err := input.FloatDefault("budget", 10.00)
	if err != nil {
		return nil, fmt.Errorf("invalid budget: %s", input.String("budget"))
	}
	rpm, err := input.IntDefault("rpm", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid RPM: %s", input.String("rpm"))
	}
	tpm, err := input.IntDefault("tpm", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid TPM: %s", input.String("tpm"))
	}
	parallel, err := input.IntDefault("parallel", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid parallel: %s", input.String("parallel"))
	}
	role := forcedRole
	if role == "" {
		role = input.StringDefault("role", "developer")
	}
	return keyGenOptions{
		Alias:    alias,
		Budget:   budget,
		RPM:      rpm,
		TPM:      tpm,
		Parallel: parallel,
		Duration: input.StringDefault("duration", "30d"),
		Role:     role,
		DryRun:   input.Bool("dry-run"),
	}, nil
}

func bindKeyRevokeOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	alias := input.Argument(0)
	if alias == "" {
		return nil, fmt.Errorf("alias is required")
	}
	return keyRevokeOptions{Alias: alias}, nil
}

func runKeyGen(ctx context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(keyGenOptions)
	out := output.New()
	plan, err := keygen.PlanGenerateRequest(keygen.GenerateRequestConfig{
		Alias:    options.Alias,
		Budget:   options.Budget,
		RPM:      options.RPM,
		TPM:      options.TPM,
		Parallel: options.Parallel,
		Duration: options.Duration,
		Role:     options.Role,
	})
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Invalid key request: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	if err := keygen.CheckPrerequisites(!options.DryRun); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		if !options.DryRun {
			fmt.Fprintln(runCtx.Stderr, "Set it in your environment: export LITELLM_MASTER_KEY=...")
		}
		return exitcodes.ACPExitPrereq
	}

	if options.DryRun {
		return runDryRun(options, plan, runCtx.Stdout, out)
	}

	if !prereq.CommandExists("curl") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("curl is required"))
		return exitcodes.ACPExitPrereq
	}

	return generateKey(ctx, plan, runCtx.Stdout, runCtx.Stderr, out)
}

func runDryRun(config keyGenOptions, plan keygen.GenerateRequestPlan, stdout *os.File, out *output.Output) int {
	printKeyGenerationPlan(stdout, out, "=== Key Generation (Dry Run) ===", config.Budget, plan)
	return exitcodes.ACPExitSuccess
}

func generateKey(ctx context.Context, plan keygen.GenerateRequestPlan, stdout *os.File, stderr *os.File, out *output.Output) int {
	masterKey := acpconfig.NewLoader().Gateway(true).MasterKey
	client := gateway.NewClient(gateway.WithMasterKey(masterKey))

	printKeyGenerationProgress(stderr, plan)

	resp, err := client.GenerateKey(ctx, &plan.Request)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Key generation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, resp.Key)
	return exitcodes.ACPExitSuccess
}

func runKeyRevoke(ctx context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(keyRevokeOptions)
	out := output.New()

	if err := keygen.CheckPrerequisites(true); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	client := gateway.NewClient(gateway.WithMasterKey(acpconfig.NewLoader().Gateway(true).MasterKey))
	printKeyRevokeProgress(runCtx.Stderr, config.Alias)
	if err := client.DeleteKey(ctx, config.Alias); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key revocation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	printKeyRevokeSuccess(runCtx.Stdout, out, config.Alias)

	return exitcodes.ACPExitSuccess
}

func runKeyGenCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"key", "gen"}, args, stdout, stderr)
}

func runDeveloperKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"key", "gen-dev"}, args, stdout, stderr)
}

func runLeadKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"key", "gen-lead"}, args, stdout, stderr)
}

func runKeyRevokeCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"key", "revoke"}, args, stdout, stderr)
}
