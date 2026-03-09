// cmd_validate.go - Validation command adapters.
//
// Purpose:
//   - Expose typed repository validation workflows through spec-owned bind/run flows.
//
// Responsibilities:
//   - Define the typed `validate` command tree.
//   - Delegate validation logic to internal packages.
//   - Render deterministic findings for local CI and operators.
//
// Scope:
//   - Command-layer adapters only; validators live in internal packages.
//
// Usage:
//   - Invoked through `acpctl validate <subcommand>`.
//
// Invariants/Assumptions:
//   - Validators are side-effect free.
//   - The CLI package does not own validation policy.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/contracts"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

type validateDetectionsOptions struct {
	Verbose bool
}

type validateSIEMQueriesOptions struct {
	ValidateSchema bool
	Verbose        bool
}

type validateConfigOptions struct {
	Production     bool
	SecretsEnvFile string
}

func validateCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "validate",
		Summary:     "Configuration and policy validation operations",
		Description: "Configuration and policy validation operations.",
		Examples: []string{
			"acpctl validate config",
			"acpctl validate config --production --secrets-env-file /etc/ai-control-plane/secrets.env",
			"acpctl validate lint",
			"acpctl validate detections",
		},
		Children: []*commandSpec{
			makeLeafSpec("lint", "Run static validation/lint gate", "lint"),
			validateConfigCommandSpec(),
			validateDetectionsCommandSpec(),
			validateSIEMQueriesCommandSpec(),
			validatePublicHygieneCommandSpec(),
			validateLicenseCommandSpec(),
			validateSupplyChainCommandSpec(),
			validateSecretsAuditCommandSpec(),
			validateComposeHealthchecksCommandSpec(),
			validateHeadersCommandSpec(),
			validateEnvAccessCommandSpec(),
			makeLeafSpec("security", "Run Make-composed security gate (hygiene, secrets, license, supply chain)", "security-gate"),
		},
	}
}

func validateDetectionsCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "detections",
		Summary:     "Validate detection rule output",
		Description: "Validate detection rule output.",
		Options: []commandOptionSpec{
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
				return validateDetectionsOptions{Verbose: input.Bool("verbose")}, nil
			},
			NativeRun: runValidateDetectionsTyped,
		},
	}
}

func validateSIEMQueriesCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "siem-queries",
		Summary:     "Validate SIEM query sync",
		Description: "Validate SIEM query sync.",
		Options: []commandOptionSpec{
			{Name: "validate-schema", Summary: "Validate schema coverage", Type: optionValueBool},
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
				return validateSIEMQueriesOptions{
					ValidateSchema: input.Bool("validate-schema"),
					Verbose:        input.Bool("verbose"),
				}, nil
			},
			NativeRun: runValidateSIEMQueriesTyped,
		},
	}
}

func validateConfigCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "config",
		Summary:     "Validate deployment configuration (use --production for host contract checks)",
		Description: "Validate deployment configuration, including production host contract checks when --production is set.",
		Options: []commandOptionSpec{
			{Name: "production", Summary: "Enforce the production deployment contract", Type: optionValueBool},
			{Name: "secrets-env-file", ValueName: "PATH", Summary: "Canonical production secrets file", Type: optionValueString},
		},
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindValidateConfigOptions,
			NativeRun:  runValidateConfigTyped,
		},
	}
}

func validateComposeHealthchecksCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "compose-healthchecks",
		Summary:     "Validate Docker Compose healthchecks",
		Description: "Validate Docker Compose healthchecks.",
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, _ parsedCommandInput) (any, error) {
				return struct{}{}, nil
			},
			NativeRun: runValidateComposeHealthchecksTyped,
		},
	}
}

func validateHeadersCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "headers",
		Summary:     "Validate Go source file header policy",
		Description: "Validate Go source file header policy.",
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, _ parsedCommandInput) (any, error) {
				return struct{}{}, nil
			},
			NativeRun: runValidateHeadersTyped,
		},
	}
}

func validateEnvAccessCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "env-access",
		Summary:     "Fail on direct environment access outside internal/config",
		Description: "Fail on direct environment access outside internal/config.",
		Backend: commandBackend{
			Kind: commandBackendNative,
			NativeBind: func(_ commandBindContext, _ parsedCommandInput) (any, error) {
				return struct{}{}, nil
			},
			NativeRun: runValidateEnvAccessTyped,
		},
	}
}

func bindValidateConfigOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return validateConfigOptions{
		Production:     input.Bool("production"),
		SecretsEnvFile: strings.TrimSpace(input.String("secrets-env-file")),
	}, nil
}

func runValidateDetectionsTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateDetectionsOptions)
	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Detection Rules Validation ==="))
	repoRoot := runCtx.RepoRoot
	rulesPath := filepath.Join(repoRoot, detectionRulesRelativePath)
	siemPath := filepath.Join(repoRoot, siemQueriesRelativePath)
	litellmPath := filepath.Join(repoRoot, litellmConfigRelativePath)
	detections, err := contracts.LoadDetectionRulesFile(rulesPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load detection rules: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	siemQueries, err := contracts.LoadSIEMQueriesFile(siemPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load SIEM query mappings: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	litellmConfig, err := catalog.LoadLiteLLMConfig(litellmPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load LiteLLM config: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stdout, "Detection rules: %s\n", rulesPath)
		fmt.Fprintf(runCtx.Stdout, "SIEM query mappings: %s\n", siemPath)
		fmt.Fprintf(runCtx.Stdout, "Approved models source: %s\n", litellmPath)
	}
	issues := contracts.ValidateDetectionContracts(detections, siemQueries, litellmConfig)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(runCtx.Stderr, "- %s\n", issue)
		}
		fmt.Fprintln(runCtx.Stderr, out.Fail("Detection validation failed"))
		return exitcodes.ACPExitDomain
	}
	validatedCount := 0
	decisionGradeCount := 0
	for _, rule := range detections.DetectionRules {
		if strings.EqualFold(rule.OperationalStatus, "validated") {
			validatedCount++
		}
		if strings.EqualFold(rule.CoverageTier, "decision-grade") {
			decisionGradeCount++
		}
	}
	fmt.Fprintf(runCtx.Stdout, "Validated %d detection rule(s) (%d validated, %d decision-grade)\n", len(detections.DetectionRules), validatedCount, decisionGradeCount)
	fmt.Fprintln(runCtx.Stdout, out.Green("Detection rules validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateSIEMQueriesTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateSIEMQueriesOptions)
	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== SIEM Queries Validation ==="))
	repoRoot := runCtx.RepoRoot
	rulesPath := filepath.Join(repoRoot, detectionRulesRelativePath)
	siemPath := filepath.Join(repoRoot, siemQueriesRelativePath)
	litellmPath := filepath.Join(repoRoot, litellmConfigRelativePath)
	detections, err := contracts.LoadDetectionRulesFile(rulesPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load detection rules: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	siemQueries, err := contracts.LoadSIEMQueriesFile(siemPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load SIEM query mappings: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	litellmConfig, err := catalog.LoadLiteLLMConfig(litellmPath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to load LiteLLM config: %v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if opts.Verbose {
		fmt.Fprintf(runCtx.Stdout, "Detection rules: %s\n", rulesPath)
		fmt.Fprintf(runCtx.Stdout, "SIEM query mappings: %s\n", siemPath)
		fmt.Fprintf(runCtx.Stdout, "Approved models source: %s\n", litellmPath)
	}
	issues := contracts.ValidateSIEMContracts(detections, siemQueries, litellmConfig, opts.ValidateSchema)
	if len(issues) > 0 {
		for _, issue := range issues {
			fmt.Fprintf(runCtx.Stderr, "- %s\n", issue)
		}
		fmt.Fprintln(runCtx.Stderr, out.Fail("SIEM query validation failed"))
		return exitcodes.ACPExitDomain
	}
	fmt.Fprintf(runCtx.Stdout, "Validated %d SIEM mapping(s) against %d enabled detection rule(s)\n", len(siemQueries.SIEMQueries), contracts.CountEnabledDetectionRules(detections))
	if opts.ValidateSchema {
		fmt.Fprintln(runCtx.Stdout, "Schema mapping coverage validated")
	}
	fmt.Fprintln(runCtx.Stdout, out.Green("SIEM query validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateConfigTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateConfigOptions)
	options := validation.ConfigValidationOptions{}
	if opts.Production {
		options.Profile = validation.ConfigValidationProfileProduction
	}
	options.SecretsEnvFile = opts.SecretsEnvFile

	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Deployment Configuration Validation ==="))
	if options.Profile == validation.ConfigValidationProfileProduction {
		fmt.Fprintf(runCtx.Stdout, "Profile: %s\n", options.Profile)
		if strings.TrimSpace(options.SecretsEnvFile) != "" {
			fmt.Fprintf(runCtx.Stdout, "Secrets file: %s\n", options.SecretsEnvFile)
		}
	}
	issues, err := validation.ValidateDeploymentConfig(runCtx.RepoRoot, options)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Configuration validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(runCtx.Stdout, out.Green("Configuration validation passed"))
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintf(runCtx.Stderr, "- %s\n", issue)
	}
	fmt.Fprintln(runCtx.Stderr, out.Fail("Configuration validation failed"))
	return exitcodes.ACPExitDomain
}

func runValidateComposeHealthchecksTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Docker Compose Healthchecks Validation ==="))
	issues, err := validation.ValidateComposeHealthchecks(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Healthcheck validation failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(runCtx.Stdout, out.Green("Healthcheck validation passed"))
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintf(runCtx.Stderr, "- %s\n", issue)
	}
	fmt.Fprintln(runCtx.Stderr, out.Fail("Healthcheck validation failed"))
	return exitcodes.ACPExitDomain
}

func runValidateHeadersTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	issues, err := validation.ValidateGoHeaders(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(runCtx.Stdout, "Go header policy validation passed")
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintln(runCtx.Stderr, issue)
	}
	return exitcodes.ACPExitDomain
}

func runValidateEnvAccessTyped(_ context.Context, runCtx commandRunContext, _ any) int {
	issues, err := validation.ValidateDirectEnvAccess(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(issues) == 0 {
		fmt.Fprintln(runCtx.Stdout, "Direct environment access policy passed")
		return exitcodes.ACPExitSuccess
	}
	for _, issue := range issues {
		fmt.Fprintln(runCtx.Stderr, issue)
	}
	return exitcodes.ACPExitDomain
}

func runValidateDetections(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "detections"}, args, stdout, stderr)
}

func runValidateSiemQueries(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "siem-queries"}, args, stdout, stderr)
}

func runValidateConfig(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "config"}, args, stdout, stderr)
}

func runValidateComposeHealthchecks(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "compose-healthchecks"}, args, stdout, stderr)
}

func runValidateHeaders(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "headers"}, args, stdout, stderr)
}

func runValidateEnvAccess(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"validate", "env-access"}, args, stdout, stderr)
}
