// cmd_validate_contracts_commands.go - Contract validation command adapters.
//
// Purpose:
//   - Own detection and SIEM contract validation command surfaces and shared
//     contract-loading helpers.
//
// Responsibilities:
//   - Define the typed detection and SIEM validation commands.
//   - Load tracked detection, SIEM, and LiteLLM config artifacts.
//   - Render deterministic contract validation summaries.
//
// Scope:
//   - Contract-oriented validation command adapters only.
//
// Usage:
//   - Invoked through `acpctl validate detections` and
//     `acpctl validate siem-queries`.
//
// Invariants/Assumptions:
//   - Contract validation logic remains in `internal/contracts`.
//   - Tracked contract file locations remain repository-relative.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/contracts"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

const (
	detectionRulesRelativePath = "demo/config/detection_rules.yaml"
	siemQueriesRelativePath    = "demo/config/siem_queries.yaml"
	litellmConfigRelativePath  = "demo/config/litellm.yaml"
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

func validateDetectionsCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    "detections",
		Summary: "Validate detection rule output",
		Options: []commandOptionSpec{
			{Name: "verbose", Short: "v", Summary: "Enable detailed output", Type: optionValueBool},
		},
		Bind: bindParsedValue(func(input parsedCommandInput) validateDetectionsOptions {
			return validateDetectionsOptions{Verbose: input.Bool("verbose")}
		}),
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
		Bind: bindParsedValue(func(input parsedCommandInput) validateSIEMQueriesOptions {
			return validateSIEMQueriesOptions{
				ValidateSchema: input.Bool("validate-schema"),
				Verbose:        input.Bool("verbose"),
			}
		}),
		Run: runValidateSIEMQueriesTyped,
	})
}

func loadValidationContracts(repoRoot string) (validationContracts, error) {
	artifacts := validationContracts{
		RulesPath:   repopath.DemoConfigPath(repoRoot, "detection_rules.yaml"),
		SIEMPath:    repopath.DemoConfigPath(repoRoot, "siem_queries.yaml"),
		LiteLLMPath: repopath.DemoConfigPath(repoRoot, "litellm.yaml"),
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

func printValidationContractPaths(stdout *os.File, artifacts validationContracts) {
	fmt.Fprintf(stdout, "Detection rules: %s\n", artifacts.RulesPath)
	fmt.Fprintf(stdout, "SIEM query mappings: %s\n", artifacts.SIEMPath)
	fmt.Fprintf(stdout, "Approved models source: %s\n", artifacts.LiteLLMPath)
}

func runValidateDetectionsTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateDetectionsOptions)
	return withValidationContracts(runCtx, "=== Detection Rules Validation ===", opts.Verbose, "detection validation failed", func(out *output.Output, artifacts validationContracts) int {
		issues := contracts.ValidateDetectionContracts(artifacts.Detections, artifacts.SIEMQueries, artifacts.LiteLLM)
		if len(issues) > 0 {
			return failValidation(runCtx.Stderr, out, issues, "Detection validation failed")
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
	})
}

func runValidateSIEMQueriesTyped(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(validateSIEMQueriesOptions)
	return withValidationContracts(runCtx, "=== SIEM Queries Validation ===", opts.Verbose, "SIEM query validation failed", func(out *output.Output, artifacts validationContracts) int {
		issues := contracts.ValidateSIEMContracts(artifacts.Detections, artifacts.SIEMQueries, artifacts.LiteLLM, opts.ValidateSchema)
		if len(issues) > 0 {
			return failValidation(runCtx.Stderr, out, issues, "SIEM query validation failed")
		}
		fmt.Fprintf(runCtx.Stdout, "Validated %d SIEM mapping(s) against %d enabled detection rule(s)\n", len(artifacts.SIEMQueries.SIEMQueries), contracts.CountEnabledDetectionRules(artifacts.Detections))
		if opts.ValidateSchema {
			fmt.Fprintln(runCtx.Stdout, "Schema mapping coverage validated")
		}
		fmt.Fprintln(runCtx.Stdout, out.Green("SIEM query validation passed"))
		return exitcodes.ACPExitSuccess
	})
}
