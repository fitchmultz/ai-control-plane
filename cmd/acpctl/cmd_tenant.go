// cmd_tenant.go - Tenant design-package command surface.
//
// Purpose:
//   - Provide typed CLI workflows for inspecting and validating the tracked
//     design-only multi-tenant package.
//
// Responsibilities:
//   - Define the `tenant inspect` and `tenant validate` command tree.
//   - Render concise inspection summaries for the tracked tenant design config.
//   - Delegate validation to `internal/validation` while preserving the repo's
//     truth boundary.
//
// Scope:
//   - Design-package inspection and validation only.
//
// Usage:
//   - `acpctl tenant inspect`
//   - `acpctl tenant validate`
//   - `acpctl validate tenant`
//
// Invariants/Assumptions:
//   - The tenant surface remains design-only and incubating.
//   - Commands must not imply runtime multi-tenant enforcement exists.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

type tenantInspectOptions struct {
	RepoRoot   string
	ConfigPath string
	Format     string
}

type tenantValidateOptions struct {
	RepoRoot   string
	ConfigPath string
}

func tenantCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "tenant",
		Summary:     "Inspect and validate the design-only multi-tenant package",
		Description: "Inspect and validate the design-only multi-tenant package.",
		Examples: []string{
			"acpctl tenant inspect",
			"acpctl tenant inspect --format json",
			"acpctl tenant validate",
		},
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "inspect",
				Summary:     "Print a concise summary of the tracked tenant design package",
				Description: "Print a concise summary of the tracked tenant design package.",
				Options: []commandOptionSpec{
					{Name: "file", ValueName: "PATH", Summary: "Tenant design file to inspect", Type: optionValueString, DefaultText: tenant.DefaultDesignPath},
					{Name: "format", ValueName: "FORMAT", Summary: "Output format: text or json", Type: optionValueString, DefaultText: "text"},
				},
				Bind: bindRepoParsed(bindTenantInspectOptions),
				Run:  runTenantInspectTyped,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "validate",
				Summary:     "Validate the tracked tenant design package and truth markers",
				Description: "Validate the tracked tenant design package and truth markers.",
				Options: []commandOptionSpec{
					{Name: "file", ValueName: "PATH", Summary: "Tenant design file to validate", Type: optionValueString, DefaultText: tenant.DefaultDesignPath},
				},
				Bind: bindRepoParsed(bindTenantValidateOptions),
				Run:  runTenantValidateTyped,
			}),
		},
	}
}

func validateTenantCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "tenant",
		Summary:     "Validate the tracked tenant design package and truth markers",
		Description: "Validate the tracked tenant design package and truth markers.",
		Options: []commandOptionSpec{
			{Name: "file", ValueName: "PATH", Summary: "Tenant design file to validate", Type: optionValueString, DefaultText: tenant.DefaultDesignPath},
		},
		Bind: bindRepoParsed(bindTenantValidateOptions),
		Run:  runTenantValidateTyped,
	})
}

func bindTenantInspectOptions(bindCtx commandBindContext, input parsedCommandInput) (tenantInspectOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return tenantInspectOptions{}, err
	}
	options := tenantInspectOptions{
		RepoRoot:   repoRoot,
		ConfigPath: resolveRepoInput(repoRoot, tenant.DefaultDesignPath),
		Format:     "text",
	}
	if input.Has("file") {
		options.ConfigPath = resolveRepoInput(repoRoot, input.NormalizedString("file"))
	}
	if input.Has("format") {
		options.Format = input.LowerString("format")
	}
	return options, nil
}

func bindTenantValidateOptions(bindCtx commandBindContext, input parsedCommandInput) (tenantValidateOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return tenantValidateOptions{}, err
	}
	options := tenantValidateOptions{
		RepoRoot:   repoRoot,
		ConfigPath: resolveRepoInput(repoRoot, tenant.DefaultDesignPath),
	}
	if input.Has("file") {
		options.ConfigPath = resolveRepoInput(repoRoot, input.NormalizedString("file"))
	}
	return options, nil
}

func runTenantInspectTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	out := output.New()
	opts := raw.(tenantInspectOptions)
	design, err := tenant.LoadFile(opts.ConfigPath)
	if err != nil {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "tenant inspect failed")
	}
	summary := design.Summary()
	format := opts.Format
	if format == "" {
		format = "text"
	}

	if format == "json" {
		payload, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "tenant inspect failed")
		}
		_, _ = runCtx.Stdout.Write(append(payload, '\n'))
		return exitcodes.ACPExitSuccess
	}
	if format != "text" {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitUsage, fmt.Errorf("unsupported format %q", format), "tenant inspect failed")
	}

	printCommandSection(runCtx.Stdout, out, "Tenant design package")
	printCommandDetail(runCtx.Stdout, "Design state", summary.DesignState)
	printCommandDetail(runCtx.Stdout, "Organizations", summary.OrganizationCount)
	printCommandDetail(runCtx.Stdout, "Workspaces", summary.WorkspaceCount)
	printCommandDetail(runCtx.Stdout, "Role bindings", summary.RoleBindingCount)
	printCommandDetail(runCtx.Stdout, "Reports", summary.ReportDefinitionCount)
	printCommandDetail(runCtx.Stdout, "Chargeback", summary.ChargebackBoundary)
	printCommandDetail(runCtx.Stdout, "Provider billing", summary.ProviderBillingBoundary)
	printCommandDetail(runCtx.Stdout, "Namespaces", summary.KeyNamespaces)
	printCommandDetail(runCtx.Stdout, "Predicates", summary.RequiredPredicates)
	printCommandDetail(runCtx.Stdout, "Organizations list", summary.Organizations)
	return exitcodes.ACPExitSuccess
}

func runTenantValidateTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(tenantValidateOptions)
	return runIssueValidationWithPrelude(runCtx, nil, func(out *output.Output, runCtx commandRunContext) {
		printCommandSection(runCtx.Stdout, out, "=== Tenant Design Validation ===")
		fmt.Fprintf(runCtx.Stdout, "Tenant design: %s\n", opts.ConfigPath)
	}, issueValidationConfig{
		SuccessMessage:  "Tenant design validation passed",
		FailureMessage:  "Tenant design validation failed",
		RuntimeErrorMsg: "Tenant design validation failed",
		ColorSuccess:    true,
	}, func() ([]string, error) {
		return validation.ValidateTenantConfig(runCtx.RepoRoot, validation.TenantValidationOptions{ConfigPath: opts.ConfigPath})
	})
}

func runTenantCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandGroupPath(ctx, []string{"tenant"}, args, stdout, stderr)
}
