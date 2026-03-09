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

type validationContracts struct {
	Detections  contracts.DetectionRulesFile
	SIEMQueries contracts.SIEMQueriesFile
	LiteLLM     catalog.LiteLLMConfig
	RulesPath   string
	SIEMPath    string
	LiteLLMPath string
}

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
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    "detections",
		Summary: "Validate detection rule output",
		Options: []commandOptionSpec{
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Bind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
			return validateDetectionsOptions{Verbose: input.Bool("verbose")}, nil
		},
		Run: runValidateDetectionsTyped,
	})
}

func validateSIEMQueriesCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    "siem-queries",
		Summary: "Validate SIEM query sync",
		Options: []commandOptionSpec{
			{Name: "validate-schema", Summary: "Validate schema coverage", Type: optionValueBool},
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Bind: func(_ commandBindContext, input parsedCommandInput) (any, error) {
			return validateSIEMQueriesOptions{
				ValidateSchema: input.Bool("validate-schema"),
				Verbose:        input.Bool("verbose"),
			}, nil
		},
		Run: runValidateSIEMQueriesTyped,
	})
}

func validateConfigCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "config",
		Summary:     "Validate deployment configuration (use --production for host contract checks)",
		Description: "Validate deployment configuration, including production host contract checks when --production is set.",
		Options: []commandOptionSpec{
			{Name: "production", Summary: "Enforce the production deployment contract", Type: optionValueBool},
			{Name: "secrets-env-file", ValueName: "PATH", Summary: "Canonical production secrets file", Type: optionValueString},
		},
		Bind: bindValidateConfigOptions,
		Run:  runValidateConfigTyped,
	})
}

func validateComposeHealthchecksCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("compose-healthchecks", "Validate Docker Compose healthchecks", runValidateComposeHealthchecksTyped)
}

func validateHeadersCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("headers", "Validate Go source file header policy", runValidateHeadersTyped)
}

func validateEnvAccessCommandSpec() *commandSpec {
	return newNativeLeafCommandSpec("env-access", "Fail on direct environment access outside internal/config", runValidateEnvAccessTyped)
}

func bindValidateConfigOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return validateConfigOptions{
		Production:     input.Bool("production"),
		SecretsEnvFile: strings.TrimSpace(input.String("secrets-env-file")),
	}, nil
}

func loadValidationContracts(repoRoot string) (validationContracts, error) {
	artifacts := validationContracts{
		RulesPath:   filepath.Join(repoRoot, detectionRulesRelativePath),
		SIEMPath:    filepath.Join(repoRoot, siemQueriesRelativePath),
		LiteLLMPath: filepath.Join(repoRoot, litellmConfigRelativePath),
	}
	var err error
	if artifacts.Detections, err = contracts.LoadDetectionRulesFile(artifacts.RulesPath); err != nil {
		return validationContracts{}, fmt.Errorf("failed to load detection rules: %w", err)
	}
	if artifacts.SIEMQueries, err = contracts.LoadSIEMQueriesFile(artifacts.SIEMPath); err != nil {
		return validationContracts{}, fmt.Errorf("failed to load SIEM query mappings: %w", err)
	}
	if artifacts.LiteLLM, err = catalog.LoadLiteLLMConfig(artifacts.LiteLLMPath); err != nil {
		return validationContracts{}, fmt.Errorf("failed to load LiteLLM config: %w", err)
	}
	return artifacts, nil
}

func printValidationContractPaths(stdout *os.File, contracts validationContracts) {
	fmt.Fprintf(stdout, "Detection rules: %s\n", contracts.RulesPath)
	fmt.Fprintf(stdout, "SIEM query mappings: %s\n", contracts.SIEMPath)
	fmt.Fprintf(stdout, "Approved models source: %s\n", contracts.LiteLLMPath)
}

func printValidationIssues(stderr *os.File, issues []string, failureMessage string, out *output.Output) int {
	for _, issue := range issues {
		fmt.Fprintf(stderr, "- %s\n", issue)
	}
	if failureMessage != "" {
		fmt.Fprintln(stderr, out.Fail(failureMessage))
	}
	return exitcodes.ACPExitDomain
}

func runValidateDetectionsTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateDetectionsOptions)
	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== Detection Rules Validation ==="))
	artifacts, err := loadValidationContracts(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if opts.Verbose {
		printValidationContractPaths(runCtx.Stdout, artifacts)
	}
	issues := contracts.ValidateDetectionContracts(artifacts.Detections, artifacts.SIEMQueries, artifacts.LiteLLM)
	if len(issues) > 0 {
		return printValidationIssues(runCtx.Stderr, issues, "Detection validation failed", out)
	}
	validatedCount := 0
	decisionGradeCount := 0
	for _, rule := range artifacts.Detections.DetectionRules {
		if strings.EqualFold(rule.OperationalStatus, "validated") {
			validatedCount++
		}
		if strings.EqualFold(rule.CoverageTier, "decision-grade") {
			decisionGradeCount++
		}
	}
	fmt.Fprintf(runCtx.Stdout, "Validated %d detection rule(s) (%d validated, %d decision-grade)\n", len(artifacts.Detections.DetectionRules), validatedCount, decisionGradeCount)
	fmt.Fprintln(runCtx.Stdout, out.Green("Detection rules validation passed"))
	return exitcodes.ACPExitSuccess
}

func runValidateSIEMQueriesTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateSIEMQueriesOptions)
	out := output.New()
	fmt.Fprintln(runCtx.Stdout, out.Bold("=== SIEM Queries Validation ==="))
	artifacts, err := loadValidationContracts(runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return mapValidationLoadExitCode(err)
	}
	if opts.Verbose {
		printValidationContractPaths(runCtx.Stdout, artifacts)
	}
	issues := contracts.ValidateSIEMContracts(artifacts.Detections, artifacts.SIEMQueries, artifacts.LiteLLM, opts.ValidateSchema)
	if len(issues) > 0 {
		return printValidationIssues(runCtx.Stderr, issues, "SIEM query validation failed", out)
	}
	fmt.Fprintf(runCtx.Stdout, "Validated %d SIEM mapping(s) against %d enabled detection rule(s)\n", len(artifacts.SIEMQueries.SIEMQueries), contracts.CountEnabledDetectionRules(artifacts.Detections))
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
	return printValidationIssues(runCtx.Stderr, issues, "Healthcheck validation failed", out)
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
