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

	if err := keygen.ValidateAlias(options.Alias); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Invalid alias: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	role := keygen.ResolveRole(options.Role)
	if err := keygen.ValidateRole(role); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Invalid role: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	if err := keygen.CheckPrerequisites(!options.DryRun); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		if !options.DryRun {
			fmt.Fprintln(runCtx.Stderr, "Set it in your environment: export LITELLM_MASTER_KEY=...")
		}
		return exitcodes.ACPExitPrereq
	}

	models := keygen.GetModelsForRole(role)
	req := &gateway.GenerateKeyRequest{
		KeyAlias:       options.Alias,
		MaxBudget:      options.Budget,
		BudgetDuration: options.Duration,
		Models:         models,
	}
	if options.RPM > 0 {
		req.RPMLimit = options.RPM
	}
	if options.TPM > 0 {
		req.TPMLimit = options.TPM
	}
	if options.Parallel > 0 {
		req.MaxParallelRequests = options.Parallel
	}

	if options.DryRun {
		return runDryRun(options, role, models, runCtx.Stdout, out)
	}

	if !prereq.CommandExists("curl") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("curl is required"))
		return exitcodes.ACPExitPrereq
	}

	return generateKey(ctx, options, req, runCtx.Stdout, runCtx.Stderr, out)
}

func runDryRun(config keyGenOptions, role string, models []string, stdout *os.File, out *output.Output) int {
	fmt.Fprintln(stdout, out.Bold("=== Key Generation (Dry Run) ==="))
	fmt.Fprintf(stdout, "Alias: %s\n", config.Alias)
	fmt.Fprintf(stdout, "Budget: $%.2f\n", config.Budget)
	fmt.Fprintf(stdout, "Duration: %s\n", config.Duration)
	fmt.Fprintf(stdout, "Role: %s\n", role)
	fmt.Fprintf(stdout, "Models: %v\n", models)
	if config.RPM > 0 {
		fmt.Fprintf(stdout, "RPM: %d\n", config.RPM)
	}
	if config.TPM > 0 {
		fmt.Fprintf(stdout, "TPM: %d\n", config.TPM)
	}
	if config.Parallel > 0 {
		fmt.Fprintf(stdout, "Max Parallel: %d\n", config.Parallel)
	}
	return exitcodes.ACPExitSuccess
}

func generateKey(ctx context.Context, config keyGenOptions, req *gateway.GenerateKeyRequest, stdout *os.File, stderr *os.File, out *output.Output) int {
	masterKey := acpconfig.NewLoader().Gateway(true).MasterKey
	client := gateway.NewClient(gateway.WithMasterKey(masterKey))

	role := keygen.ResolveRole(config.Role)
	fmt.Fprintf(stderr, "Generating key '%s' with budget: $%.2f (role: %s)\n", config.Alias, config.Budget, role)
	if config.RPM > 0 {
		fmt.Fprintf(stderr, "  RPM limit: %d\n", config.RPM)
	}
	if config.TPM > 0 {
		fmt.Fprintf(stderr, "  TPM limit: %d\n", config.TPM)
	}
	if config.Parallel > 0 {
		fmt.Fprintf(stderr, "  Max parallel: %d\n", config.Parallel)
	}
	fmt.Fprintf(stderr, "  Budget duration: %s\n", config.Duration)
	fmt.Fprintln(stderr, "")

	resp, err := client.GenerateKey(ctx, req)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Key generation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprintln(stdout, resp.Key)
	return exitcodes.ACPExitSuccess
}

func runKeyRevoke(_ context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(keyRevokeOptions)
	out := output.New()

	if err := keygen.CheckPrerequisites(true); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	fmt.Fprintf(runCtx.Stdout, "Revoking key: %s\n", config.Alias)
	fmt.Fprintln(runCtx.Stdout, out.Yellow("Note: Key revocation requires LiteLLM admin API"))
	fmt.Fprintln(runCtx.Stdout, "This would call DELETE /key/{alias} if supported by LiteLLM")

	return exitcodes.ACPExitSuccess
}

func runKeyGenCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"key", "gen"}, args, stdout, stderr)
}

func runDeveloperKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"key", "gen-dev"}, args, stdout, stderr)
}

func runLeadKeyGenLegacy(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"key", "gen-lead"}, args, stdout, stderr)
}

func runKeyRevokeCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"key", "revoke"}, args, stdout, stderr)
}
