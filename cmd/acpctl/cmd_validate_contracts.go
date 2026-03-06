// cmd_validate_contracts.go - Detection and SIEM validation contracts.
//
// Purpose: Parse and validate enterprise-governance configuration contracts.
// Responsibilities:
//   - Load YAML-backed detection, SIEM, and LiteLLM model definitions.
//   - Enforce structural consistency between detections and SIEM mappings.
//   - Validate approved-model placeholders and normalized schema mappings.
//
// Non-scope:
//   - Does not execute detections against live infrastructure.
//   - Does not render SIEM queries for a specific customer environment.

package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"gopkg.in/yaml.v3"
)

const (
	detectionRulesRelativePath = "demo/config/detection_rules.yaml"
	siemQueriesRelativePath    = "demo/config/siem_queries.yaml"
	litellmConfigRelativePath  = "demo/config/litellm.yaml"
)

var detectionRuleIDPattern = regexp.MustCompile(`^DR-\d{3}$`)

type detectionRulesFile struct {
	DetectionRules []detectionRule `yaml:"detection_rules"`
}

type detectionRule struct {
	RuleID            string              `yaml:"rule_id"`
	Name              string              `yaml:"name"`
	Description       string              `yaml:"description"`
	Severity          string              `yaml:"severity"`
	Category          string              `yaml:"category"`
	OperationalStatus string              `yaml:"operational_status"`
	CoverageTier      string              `yaml:"coverage_tier"`
	ExpectedSignal    string              `yaml:"expected_signal"`
	Enabled           bool                `yaml:"enabled"`
	SQLQuery          string              `yaml:"sql_query"`
	Remediation       string              `yaml:"remediation"`
	AutoResponse      detectionAutoAction `yaml:"auto_response"`
}

type detectionAutoAction struct {
	Enabled            bool   `yaml:"enabled"`
	Action             string `yaml:"action"`
	GracePeriodMinutes int    `yaml:"grace_period_minutes"`
}

type siemQueriesFile struct {
	SIEMQueries []siemQueryRule `yaml:"siem_queries"`
	SIEMConfig  siemConfig      `yaml:"siem_config"`
}

type siemQueryRule struct {
	RuleID      string            `yaml:"rule_id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Severity    string            `yaml:"severity"`
	Category    string            `yaml:"category"`
	Enabled     bool              `yaml:"enabled"`
	Splunk      siemPlatformQuery `yaml:"splunk"`
	ELKKQL      siemPlatformQuery `yaml:"elk_kql"`
	SentinelKQL siemPlatformQuery `yaml:"sentinel_kql"`
	Sigma       sigmaQueryRule    `yaml:"sigma"`
}

type siemPlatformQuery struct {
	Platform    string `yaml:"platform"`
	Query       string `yaml:"query"`
	Aggregation string `yaml:"aggregation"`
	Notes       string `yaml:"notes"`
	Severity    string `yaml:"severity"`
}

type sigmaQueryRule struct {
	Title       string         `yaml:"title"`
	Status      string         `yaml:"status"`
	Description string         `yaml:"description"`
	Level       string         `yaml:"level"`
	Detection   map[string]any `yaml:"detection"`
	Tags        []string       `yaml:"tags"`
}

type siemConfig struct {
	FieldMappings []fieldMapping `yaml:"field_mappings"`
}

type fieldMapping struct {
	Normalized string `yaml:"normalized"`
	Splunk     string `yaml:"splunk"`
	Elk        string `yaml:"elk"`
	Sentinel   string `yaml:"sentinel"`
}

type liteLLMValidationConfig struct {
	ModelList []liteLLMModel `yaml:"model_list"`
}

type liteLLMModel struct {
	ModelName string `yaml:"model_name"`
}

func loadDetectionRulesFile(path string) (detectionRulesFile, error) {
	var config detectionRulesFile
	if err := loadYAMLFile(path, &config); err != nil {
		return detectionRulesFile{}, err
	}
	return config, nil
}

func loadSIEMQueriesFile(path string) (siemQueriesFile, error) {
	var config siemQueriesFile
	if err := loadYAMLFile(path, &config); err != nil {
		return siemQueriesFile{}, err
	}
	return config, nil
}

func loadLiteLLMValidationConfig(path string) (liteLLMValidationConfig, error) {
	var config liteLLMValidationConfig
	if err := loadYAMLFile(path, &config); err != nil {
		return liteLLMValidationConfig{}, err
	}
	return config, nil
}

func loadYAMLFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s not found", path)
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func mapValidationLoadExitCode(err error) int {
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return exitcodes.ACPExitPrereq
	}
	return exitcodes.ACPExitRuntime
}

func validateDetectionContracts(detections detectionRulesFile, siemQueries siemQueriesFile, litellmConfig liteLLMValidationConfig) []string {
	issues := validateDetectionRules(detections, litellmConfig)
	issues = append(issues, validateSIEMRuleCoverage(detections, siemQueries, litellmConfig)...)
	return sortIssues(issues)
}

func validateSIEMContracts(detections detectionRulesFile, siemQueries siemQueriesFile, litellmConfig liteLLMValidationConfig, validateSchema bool) []string {
	issues := validateSIEMRuleCoverage(detections, siemQueries, litellmConfig)
	if validateSchema {
		issues = append(issues, validateSIEMSchemaMappings(siemQueries.SIEMConfig)...)
	}
	return sortIssues(issues)
}

func validateDetectionRules(detections detectionRulesFile, litellmConfig liteLLMValidationConfig) []string {
	issues := make([]string, 0)
	if len(detections.DetectionRules) == 0 {
		issues = append(issues, "demo/config/detection_rules.yaml: detection_rules must contain at least one rule")
		return issues
	}

	if len(approvedModels(litellmConfig)) == 0 {
		issues = append(issues, "demo/config/litellm.yaml: model_list must contain at least one approved model")
	}

	seenIDs := make(map[string]struct{})
	seenNames := make(map[string]struct{})
	allowedSeverity := map[string]struct{}{"low": {}, "medium": {}, "high": {}}
	allowedStatus := map[string]struct{}{"validated": {}, "example": {}}
	allowedCoverage := map[string]struct{}{"decision-grade": {}, "demo": {}}

	for idx, rule := range detections.DetectionRules {
		prefix := fmt.Sprintf("demo/config/detection_rules.yaml: rule[%d]", idx)
		ruleID := strings.TrimSpace(rule.RuleID)
		name := strings.TrimSpace(rule.Name)
		if !detectionRuleIDPattern.MatchString(ruleID) {
			issues = append(issues, fmt.Sprintf("%s: rule_id must match DR-### (got %q)", prefix, rule.RuleID))
		}
		if _, ok := seenIDs[ruleID]; ok {
			issues = append(issues, fmt.Sprintf("%s: duplicate rule_id %q", prefix, ruleID))
		}
		seenIDs[ruleID] = struct{}{}
		if name == "" {
			issues = append(issues, fmt.Sprintf("%s: name is required", prefix))
		} else {
			if _, ok := seenNames[name]; ok {
				issues = append(issues, fmt.Sprintf("%s: duplicate rule name %q", prefix, name))
			}
			seenNames[name] = struct{}{}
		}
		if _, ok := allowedSeverity[strings.ToLower(strings.TrimSpace(rule.Severity))]; !ok {
			issues = append(issues, fmt.Sprintf("%s: severity must be low, medium, or high", prefix))
		}
		if strings.TrimSpace(rule.Category) == "" {
			issues = append(issues, fmt.Sprintf("%s: category is required", prefix))
		}
		if strings.TrimSpace(rule.Description) == "" {
			issues = append(issues, fmt.Sprintf("%s: description is required", prefix))
		}
		if strings.TrimSpace(rule.Remediation) == "" {
			issues = append(issues, fmt.Sprintf("%s: remediation is required", prefix))
		}
		if _, ok := allowedStatus[strings.ToLower(strings.TrimSpace(rule.OperationalStatus))]; !ok {
			issues = append(issues, fmt.Sprintf("%s: operational_status must be validated or example", prefix))
		}
		if _, ok := allowedCoverage[strings.ToLower(strings.TrimSpace(rule.CoverageTier))]; !ok {
			issues = append(issues, fmt.Sprintf("%s: coverage_tier must be decision-grade or demo", prefix))
		}
		if strings.EqualFold(rule.CoverageTier, "decision-grade") && !strings.EqualFold(rule.OperationalStatus, "validated") {
			issues = append(issues, fmt.Sprintf("%s: decision-grade rules must also be operational_status=validated", prefix))
		}
		if strings.TrimSpace(rule.ExpectedSignal) == "" {
			issues = append(issues, fmt.Sprintf("%s: expected_signal is required", prefix))
		}
		if rule.Enabled && strings.TrimSpace(rule.SQLQuery) == "" {
			issues = append(issues, fmt.Sprintf("%s: enabled rules must include sql_query", prefix))
		}
		if rule.AutoResponse.Enabled && strings.TrimSpace(rule.AutoResponse.Action) == "" {
			issues = append(issues, fmt.Sprintf("%s: auto_response.enabled requires auto_response.action", prefix))
		}
		if ruleID == "DR-001" && !strings.Contains(rule.SQLQuery, "APPROVED_MODELS_JSON") {
			issues = append(issues, fmt.Sprintf("%s: DR-001 sql_query must use APPROVED_MODELS_JSON placeholder", prefix))
		}
	}

	return issues
}

func validateSIEMRuleCoverage(detections detectionRulesFile, siemQueries siemQueriesFile, litellmConfig liteLLMValidationConfig) []string {
	issues := make([]string, 0)
	if len(siemQueries.SIEMQueries) == 0 {
		issues = append(issues, "demo/config/siem_queries.yaml: siem_queries must contain at least one mapping")
		return issues
	}

	detectionByID := make(map[string]detectionRule)
	for _, rule := range detections.DetectionRules {
		if rule.Enabled {
			detectionByID[rule.RuleID] = rule
		}
	}

	seenIDs := make(map[string]struct{})
	for idx, query := range siemQueries.SIEMQueries {
		prefix := fmt.Sprintf("demo/config/siem_queries.yaml: siem_queries[%d]", idx)
		ruleID := strings.TrimSpace(query.RuleID)
		if !detectionRuleIDPattern.MatchString(ruleID) {
			issues = append(issues, fmt.Sprintf("%s: rule_id must match DR-### (got %q)", prefix, query.RuleID))
		}
		if _, ok := seenIDs[ruleID]; ok {
			issues = append(issues, fmt.Sprintf("%s: duplicate rule_id %q", prefix, ruleID))
		}
		seenIDs[ruleID] = struct{}{}

		detectionRule, ok := detectionByID[ruleID]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: no enabled detection rule matches %q", prefix, ruleID))
			continue
		}
		if strings.TrimSpace(query.Name) != strings.TrimSpace(detectionRule.Name) {
			issues = append(issues, fmt.Sprintf("%s: name must match detection rule %q", prefix, detectionRule.Name))
		}
		if !strings.EqualFold(query.Severity, detectionRule.Severity) {
			issues = append(issues, fmt.Sprintf("%s: severity must match detection rule %q", prefix, detectionRule.Severity))
		}
		if !strings.EqualFold(query.Category, detectionRule.Category) {
			issues = append(issues, fmt.Sprintf("%s: category must match detection rule %q", prefix, detectionRule.Category))
		}
		if query.Enabled != detectionRule.Enabled {
			issues = append(issues, fmt.Sprintf("%s: enabled must match detection rule (%t)", prefix, detectionRule.Enabled))
		}
		issues = append(issues, validatePlatformQuery(prefix+" splunk", query.Splunk)...)
		issues = append(issues, validatePlatformQuery(prefix+" elk_kql", query.ELKKQL)...)
		issues = append(issues, validatePlatformQuery(prefix+" sentinel_kql", query.SentinelKQL)...)
		issues = append(issues, validateSigmaQuery(prefix+" sigma", query.Sigma)...)
		if ruleID == "DR-001" {
			issues = append(issues, validateApprovedModelPlaceholders(prefix, query)...)
		}
	}

	for ruleID := range detectionByID {
		if _, ok := seenIDs[ruleID]; !ok {
			issues = append(issues, fmt.Sprintf("demo/config/siem_queries.yaml: missing SIEM mapping for enabled detection rule %q", ruleID))
		}
	}

	if len(approvedModels(litellmConfig)) == 0 {
		issues = append(issues, "demo/config/litellm.yaml: model_list must contain at least one approved model")
	}

	return issues
}

func validatePlatformQuery(prefix string, query siemPlatformQuery) []string {
	issues := make([]string, 0)
	if strings.TrimSpace(query.Platform) == "" {
		issues = append(issues, fmt.Sprintf("%s: platform is required", prefix))
	}
	if strings.TrimSpace(query.Query) == "" {
		issues = append(issues, fmt.Sprintf("%s: query is required", prefix))
	}
	return issues
}

func validateSigmaQuery(prefix string, query sigmaQueryRule) []string {
	issues := make([]string, 0)
	if strings.TrimSpace(query.Title) == "" {
		issues = append(issues, fmt.Sprintf("%s: title is required", prefix))
	}
	if strings.TrimSpace(query.Level) == "" {
		issues = append(issues, fmt.Sprintf("%s: level is required", prefix))
	}
	if len(query.Detection) == 0 {
		issues = append(issues, fmt.Sprintf("%s: detection block is required", prefix))
	}
	return issues
}

func validateApprovedModelPlaceholders(prefix string, query siemQueryRule) []string {
	issues := make([]string, 0)
	if !strings.Contains(query.Splunk.Query, "{{APPROVED_MODELS_SPLUNK}}") {
		issues = append(issues, fmt.Sprintf("%s splunk: missing {{APPROVED_MODELS_SPLUNK}} placeholder for DR-001", prefix))
	}
	if !strings.Contains(query.ELKKQL.Query, "{{APPROVED_MODELS_ELK_OR}}") {
		issues = append(issues, fmt.Sprintf("%s elk_kql: missing {{APPROVED_MODELS_ELK_OR}} placeholder for DR-001", prefix))
	}
	if !strings.Contains(query.SentinelKQL.Query, "{{APPROVED_MODELS_JSON}}") {
		issues = append(issues, fmt.Sprintf("%s sentinel_kql: missing {{APPROVED_MODELS_JSON}} placeholder for DR-001", prefix))
	}
	sigmaEncoded, _ := yaml.Marshal(query.Sigma)
	if !strings.Contains(string(sigmaEncoded), "{{APPROVED_MODELS_SIGMA}}") {
		issues = append(issues, fmt.Sprintf("%s sigma: missing {{APPROVED_MODELS_SIGMA}} placeholder for DR-001", prefix))
	}
	return issues
}

func validateSIEMSchemaMappings(config siemConfig) []string {
	issues := make([]string, 0)
	required := []string{"principal.id", "ai.model.id", "ai.request.timestamp", "ai.cost.amount", "ai.tokens.total", "policy.action"}
	fieldMap := make(map[string]fieldMapping)
	for _, mapping := range config.FieldMappings {
		fieldMap[strings.TrimSpace(mapping.Normalized)] = mapping
	}
	for _, field := range required {
		mapping, ok := fieldMap[field]
		if !ok {
			issues = append(issues, fmt.Sprintf("demo/config/siem_queries.yaml: missing siem_config.field_mappings entry for %q", field))
			continue
		}
		if strings.TrimSpace(mapping.Splunk) == "" || strings.TrimSpace(mapping.Elk) == "" || strings.TrimSpace(mapping.Sentinel) == "" {
			issues = append(issues, fmt.Sprintf("demo/config/siem_queries.yaml: field mapping %q must define splunk, elk, and sentinel names", field))
		}
	}
	return issues
}

func approvedModels(config liteLLMValidationConfig) []string {
	models := make([]string, 0, len(config.ModelList))
	for _, model := range config.ModelList {
		name := strings.TrimSpace(model.ModelName)
		if name != "" {
			models = append(models, name)
		}
	}
	sort.Strings(models)
	return models
}

func countEnabledDetectionRules(detections detectionRulesFile) int {
	count := 0
	for _, rule := range detections.DetectionRules {
		if rule.Enabled {
			count++
		}
	}
	return count
}

func sortIssues(issues []string) []string {
	if len(issues) == 0 {
		return issues
	}
	sorted := append([]string(nil), issues...)
	sort.Strings(sorted)
	return sorted
}
