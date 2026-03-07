// cmd_validate_contracts.go - CLI adapters for typed validation contracts.
//
// Purpose:
//
//	Expose internal contract loading and validation through thin CLI-facing
//	adapters instead of owning domain logic in the command layer.
//
// Responsibilities:
//   - Define canonical repository-relative contract file paths.
//   - Map contract loading errors to CLI exit codes.
//   - Re-export typed loaders and validators for command handlers and tests.
//
// Scope:
//   - CLI integration glue only; validation logic lives in internal/contracts.
//
// Usage:
//   - Called by cmd_validate.go and related tests.
//
// Invariants/Assumptions:
//   - Validation ownership remains outside the CLI package.
//   - Error-to-exit-code mapping stays stable for missing-file cases.
package main

import (
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/contracts"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

const (
	detectionRulesRelativePath = "demo/config/detection_rules.yaml"
	siemQueriesRelativePath    = "demo/config/siem_queries.yaml"
	litellmConfigRelativePath  = "demo/config/litellm.yaml"
	demoPresetsRelativePath    = "demo/config/demo_presets.yaml"
)

type detectionRulesFile = contracts.DetectionRulesFile
type detectionRule = contracts.DetectionRule
type detectionAutoAction = contracts.DetectionAutoAction
type siemQueriesFile = contracts.SIEMQueriesFile
type siemQueryRule = contracts.SIEMQueryRule
type siemPlatformQuery = contracts.SIEMPlatformQuery
type sigmaQueryRule = contracts.SigmaQueryRule
type siemConfig = contracts.SIEMConfig
type fieldMapping = contracts.FieldMapping
type liteLLMValidationConfig = catalog.LiteLLMConfig
type liteLLMModel = catalog.LiteLLMModel

func loadDetectionRulesFile(path string) (detectionRulesFile, error) {
	return contracts.LoadDetectionRulesFile(path)
}

func loadSIEMQueriesFile(path string) (siemQueriesFile, error) {
	return contracts.LoadSIEMQueriesFile(path)
}

func loadLiteLLMValidationConfig(path string) (liteLLMValidationConfig, error) {
	return catalog.LoadLiteLLMConfig(path)
}

func mapValidationLoadExitCode(err error) int {
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return exitcodes.ACPExitPrereq
	}
	return exitcodes.ACPExitRuntime
}

func validateDetectionContracts(detections detectionRulesFile, siemQueries siemQueriesFile, litellmConfig liteLLMValidationConfig) []string {
	return contracts.ValidateDetectionContracts(detections, siemQueries, litellmConfig)
}

func validateSIEMContracts(detections detectionRulesFile, siemQueries siemQueriesFile, litellmConfig liteLLMValidationConfig, validateSchema bool) []string {
	return contracts.ValidateSIEMContracts(detections, siemQueries, litellmConfig, validateSchema)
}

func approvedModels(config liteLLMValidationConfig) []string {
	return contracts.ApprovedModels(config)
}

func countEnabledDetectionRules(detections detectionRulesFile) int {
	return contracts.CountEnabledDetectionRules(detections)
}
