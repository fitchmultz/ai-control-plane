// cmd_key_ops.go - Key operations command implementation
//
// Purpose: Provide native Go implementation of key generation and management.
//
// Responsibilities:
//   - Define the typed `key` command tree and binders.
//   - Validate key-generation options using internal/keygen helpers.
//   - Execute key generation and revocation via the gateway client.
//
// Non-scope:
//   - Does not own lifecycle inspection or rotation workflows.
//   - Does not own business validation rules.
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
	"errors"
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
	RepoRoot       string
	Alias          string
	Budget         float64
	RPM            int
	TPM            int
	Parallel       int
	Duration       string
	Role           string
	DryRun         bool
	TenantFile     string
	OrganizationID string
	WorkspaceID    string
}

type keyRevokeOptions struct {
	Alias string
	keyTenantScopeOptions
}

func keyCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "key",
		Summary:     "Virtual key lifecycle operations",
		Description: "Virtual key lifecycle operations.",
		Examples: []string{
			"acpctl key gen alice --budget 10.00",
			"acpctl key gen svc-claims --organization falcon-insurance --workspace claims-adjuster --budget 10.00",
			"acpctl key list",
			"acpctl key list --organization falcon-insurance",
			"acpctl key inspect falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --month 2026-02",
			"acpctl key rotate falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --dry-run",
			"acpctl key revoke falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster",
		},
		Children: []*commandSpec{
			keyGenCommandSpec("gen", "Generate a standard virtual key", ""),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "list",
				Summary:     "List virtual keys",
				Description: "List virtual keys and their configured limits, optionally filtered to a tracked tenant boundary.",
				Examples: []string{
					"acpctl key list",
					"acpctl key list --organization falcon-insurance",
					"acpctl key list --organization falcon-insurance --workspace claims-adjuster --json",
				},
				Options: []commandOptionSpec{
					{Name: "organization", ValueName: "ORG", Summary: "Filter inventory to one tenant organization", Type: optionValueString},
					{Name: "workspace", ValueName: "WORKSPACE", Summary: "Filter inventory to one tenant workspace; requires --organization", Type: optionValueString},
					{Name: "tenant-file", ValueName: "PATH", Summary: "Tenant design file used to resolve lifecycle scope", Type: optionValueString, DefaultText: "demo/config/tenant_design.yaml"},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsed(func(input parsedCommandInput) (keyListOptions, error) {
					return bindKeyListOptions(input)
				}),
				Run: runKeyList,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "inspect",
				Summary:     "Inspect a virtual key and its usage",
				Description: "Inspect a virtual key and summarize spend and usage for the selected report month, optionally enforcing a tracked tenant boundary.",
				Examples: []string{
					"acpctl key inspect alice",
					"acpctl key inspect falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --month 2026-02",
					"acpctl key inspect falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --json",
				},
				Arguments: []commandArgumentSpec{
					{Name: "alias", Summary: "Key alias to inspect", Required: true},
				},
				Options: []commandOptionSpec{
					{Name: "month", ValueName: "YYYY-MM", Summary: "Usage month", Type: optionValueString},
					{Name: "organization", ValueName: "ORG", Summary: "Enforce one tenant organization boundary", Type: optionValueString},
					{Name: "workspace", ValueName: "WORKSPACE", Summary: "Enforce one tenant workspace boundary; requires --organization", Type: optionValueString},
					{Name: "tenant-file", ValueName: "PATH", Summary: "Tenant design file used to resolve lifecycle scope", Type: optionValueString, DefaultText: "demo/config/tenant_design.yaml"},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsed(func(input parsedCommandInput) (keyInspectOptions, error) {
					return bindKeyInspectOptions(input)
				}),
				Run: runKeyInspect,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "rotate",
				Summary:     "Stage rotation for a virtual key",
				Description: "Generate a replacement key, inspect current usage, and stage a controlled cutover, optionally enforcing a tracked tenant boundary.",
				Examples: []string{
					"acpctl key rotate alice --replacement-alias alice-rotated",
					"acpctl key rotate falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --dry-run",
					"acpctl key rotate falcon-insurance--claims-adjuster--svc-claims__cc-1100 --organization falcon-insurance --workspace claims-adjuster --revoke-old",
				},
				Arguments: []commandArgumentSpec{
					{Name: "alias", Summary: "Existing key alias", Required: true},
				},
				Options: []commandOptionSpec{
					{Name: "replacement-alias", ValueName: "ALIAS", Summary: "Replacement alias; defaults to a timestamped alias", Type: optionValueString},
					{Name: "budget", ValueName: "BUDGET", Summary: "Replacement max budget in USD", Type: optionValueFloat},
					{Name: "rpm", ValueName: "RPM", Summary: "Replacement requests-per-minute limit", Type: optionValueInt},
					{Name: "tpm", ValueName: "TPM", Summary: "Replacement tokens-per-minute limit", Type: optionValueInt},
					{Name: "parallel", ValueName: "N", Summary: "Replacement max parallel requests", Type: optionValueInt},
					{Name: "duration", ValueName: "DUR", Summary: "Replacement budget duration", Type: optionValueString},
					{Name: "role", ValueName: "ROLE", Summary: "Replacement role override", Type: optionValueString, Suggestions: func(string) []string { return keygen.ValidRoles() }},
					{Name: "month", ValueName: "YYYY-MM", Summary: "Usage month for inspection context", Type: optionValueString},
					{Name: "organization", ValueName: "ORG", Summary: "Enforce one tenant organization boundary", Type: optionValueString},
					{Name: "workspace", ValueName: "WORKSPACE", Summary: "Enforce one tenant workspace boundary; requires --organization", Type: optionValueString},
					{Name: "tenant-file", ValueName: "PATH", Summary: "Tenant design file used to resolve lifecycle scope", Type: optionValueString, DefaultText: "demo/config/tenant_design.yaml"},
					{Name: "dry-run", Summary: "Preview the staged rotation plan without generating the replacement", Type: optionValueBool},
					{Name: "revoke-old", Summary: "Immediately revoke the old key after generating the replacement", Type: optionValueBool},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsed(func(input parsedCommandInput) (keyRotateOptions, error) {
					return bindKeyRotateOptions(input)
				}),
				Run: runKeyRotate,
			}),
			{
				Name:        "revoke",
				Summary:     "Revoke a virtual key by alias",
				Description: "Revoke a virtual key by alias, optionally enforcing a tracked tenant boundary.",
				Arguments: []commandArgumentSpec{
					{Name: "alias", Summary: "Key alias to revoke", Required: true},
				},
				Options: []commandOptionSpec{
					{Name: "organization", ValueName: "ORG", Summary: "Enforce one tenant organization boundary", Type: optionValueString},
					{Name: "workspace", ValueName: "WORKSPACE", Summary: "Enforce one tenant workspace boundary; requires --organization", Type: optionValueString},
					{Name: "tenant-file", ValueName: "PATH", Summary: "Tenant design file used to resolve lifecycle scope", Type: optionValueString, DefaultText: "demo/config/tenant_design.yaml"},
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
		{Name: "organization", ValueName: "ORG", Summary: "Tenant organization id for workspace-scoped key generation", Type: optionValueString},
		{Name: "workspace", ValueName: "WORKSPACE", Summary: "Tenant workspace id for workspace-scoped key generation", Type: optionValueString},
		{Name: "tenant-file", ValueName: "PATH", Summary: "Tenant design file to use for workspace-scoped key generation", Type: optionValueString, DefaultText: "demo/config/tenant_design.yaml"},
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
			gatewayContractHelpSection(),
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
				return bindKeyGenOptions(bindCtx, input, forcedRole)
			},
			NativeRun: runKeyGen,
		},
	}
}

func bindKeyGenOptions(bindCtx commandBindContext, input parsedCommandInput, forcedRole string) (any, error) {
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
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return nil, err
	}
	organizationID := input.NormalizedString("organization")
	workspaceID := input.NormalizedString("workspace")
	if (organizationID == "") != (workspaceID == "") {
		return nil, fmt.Errorf("--organization and --workspace must be provided together")
	}
	role := forcedRole
	if role == "" {
		role = input.StringDefault("role", "developer")
	}
	return keyGenOptions{
		RepoRoot:       repoRoot,
		Alias:          alias,
		Budget:         budget,
		RPM:            rpm,
		TPM:            tpm,
		Parallel:       parallel,
		Duration:       input.StringDefault("duration", "30d"),
		Role:           role,
		DryRun:         input.Bool("dry-run"),
		TenantFile:     input.NormalizedString("tenant-file"),
		OrganizationID: organizationID,
		WorkspaceID:    workspaceID,
	}, nil
}

func bindKeyRevokeOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	alias := input.Argument(0)
	if alias == "" {
		return nil, fmt.Errorf("alias is required")
	}
	scopeOptions, err := bindKeyTenantScopeOptions(input)
	if err != nil {
		return nil, err
	}
	return keyRevokeOptions{Alias: alias, keyTenantScopeOptions: scopeOptions}, nil
}

func runKeyGen(ctx context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(keyGenOptions)
	out := output.New()
	plan, err := keygen.PlanGenerateRequest(keygen.GenerateRequestConfig{
		Alias:            options.Alias,
		Budget:           options.Budget,
		RPM:              options.RPM,
		TPM:              options.TPM,
		Parallel:         options.Parallel,
		Duration:         options.Duration,
		Role:             options.Role,
		RepoRoot:         options.RepoRoot,
		TenantConfigPath: options.TenantFile,
		OrganizationID:   options.OrganizationID,
		WorkspaceID:      options.WorkspaceID,
	})
	if err != nil {
		var validationErr *keygen.ValidationError
		if errors.As(err, &validationErr) {
			fmt.Fprintf(runCtx.Stderr, "Invalid key request: %v\n", err)
			return exitcodes.ACPExitUsage
		}
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key planning failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
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
	tenantScope, err := resolveKeyLifecycleTenantScope(ctx, runCtx.RepoRoot, config.keyTenantScopeOptions)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key revocation failed: %v\n"), err)
		if isKeyLifecycleUsageError(err) {
			return exitcodes.ACPExitUsage
		}
		return exitcodes.ACPExitRuntime
	}
	if err := keygen.RequireAliasInTenantScope(config.Alias, tenantScope); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key revocation failed: %v\n"), err)
		return exitcodes.ACPExitUsage
	}

	client := gateway.NewClient(gateway.WithMasterKey(acpconfig.NewLoader().Gateway(true).MasterKey))
	printKeyRevokeProgress(runCtx.Stderr, config.Alias, tenantScope)
	if err := client.DeleteKey(ctx, config.Alias); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key revocation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	printKeyRevokeSuccess(runCtx.Stdout, out, config.Alias, tenantScope)

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
