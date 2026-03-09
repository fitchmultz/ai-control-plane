// governance.go - Governance contract loading and validation.
//
// Purpose:
//
//	Own the typed loading and validation of detection, SIEM, and approved-model
//	governance contracts outside the CLI layer.
//
// Responsibilities:
//   - Load detection and SIEM contract files from YAML.
//   - Validate detection rule structure and approved-model placeholders.
//   - Validate SIEM coverage parity and normalized governance mappings.
//
// Scope:
//   - Read-only contract validation for committed repository configuration.
//
// Usage:
//   - Used by `acpctl validate` subcommands and related tests.
//
// Invariants/Assumptions:
//   - Validation is deterministic and side-effect free.
//   - Issue lists are sorted for stable output and tests.
package contracts

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"gopkg.in/yaml.v3"
)

var detectionRuleIDPattern = regexp.MustCompile(`^DR-\d{3}$`)

// DetectionRulesFile captures the detection rules contract file.
type DetectionRulesFile struct {
	DetectionRules []DetectionRule `yaml:"detection_rules"`
}

// DetectionRule captures one detection rule definition.
type DetectionRule struct {
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
	AutoResponse      DetectionAutoAction `yaml:"auto_response"`
}

// DetectionAutoAction captures auto-response metadata.
type DetectionAutoAction struct {
	Enabled            bool   `yaml:"enabled"`
	Action             string `yaml:"action"`
	GracePeriodMinutes int    `yaml:"grace_period_minutes"`
}

// SIEMQueriesFile captures the SIEM contract file.
type SIEMQueriesFile struct {
	SIEMQueries []SIEMQueryRule `yaml:"siem_queries"`
	SIEMConfig  SIEMConfig      `yaml:"siem_config"`
}

// SIEMQueryRule captures one SIEM mapping entry.
type SIEMQueryRule struct {
	RuleID      string            `yaml:"rule_id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Severity    string            `yaml:"severity"`
	Category    string            `yaml:"category"`
	Enabled     bool              `yaml:"enabled"`
	Splunk      SIEMPlatformQuery `yaml:"splunk"`
	ELKKQL      SIEMPlatformQuery `yaml:"elk_kql"`
	SentinelKQL SIEMPlatformQuery `yaml:"sentinel_kql"`
	Sigma       SigmaQueryRule    `yaml:"sigma"`
}

// SIEMPlatformQuery captures one vendor-specific query block.
type SIEMPlatformQuery struct {
	Platform    string `yaml:"platform"`
	Query       string `yaml:"query"`
	Aggregation string `yaml:"aggregation"`
	Notes       string `yaml:"notes"`
	Severity    string `yaml:"severity"`
}

// SigmaQueryRule captures the sigma representation for a rule.
type SigmaQueryRule struct {
	Title       string         `yaml:"title"`
	Status      string         `yaml:"status"`
	Description string         `yaml:"description"`
	Level       string         `yaml:"level"`
	Detection   map[string]any `yaml:"detection"`
	Tags        []string       `yaml:"tags"`
}

// SIEMConfig captures normalized schema mapping metadata.
type SIEMConfig struct {
	FieldMappings []FieldMapping `yaml:"field_mappings"`
}

// FieldMapping captures one normalized schema mapping.
type FieldMapping struct {
	Normalized string `yaml:"normalized"`
	Splunk     string `yaml:"splunk"`
	Elk        string `yaml:"elk"`
	Sentinel   string `yaml:"sentinel"`
}

// LoadDetectionRulesFile loads the detection rules contract.
func LoadDetectionRulesFile(path string) (DetectionRulesFile, error) {
	var config DetectionRulesFile
	if err := loadYAMLFile(path, &config); err != nil {
		return DetectionRulesFile{}, err
	}
	return config, nil
}

// LoadSIEMQueriesFile loads the SIEM queries contract.
func LoadSIEMQueriesFile(path string) (SIEMQueriesFile, error) {
	var config SIEMQueriesFile
	if err := loadYAMLFile(path, &config); err != nil {
		return SIEMQueriesFile{}, err
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

// ValidateDetectionContracts validates the full detection contract surface.
func ValidateDetectionContracts(detections DetectionRulesFile, siemQueries SIEMQueriesFile, litellmConfig catalog.LiteLLMConfig) []string {
	issues := validateDetectionRules(detections, litellmConfig)
	issues = append(issues, validateSIEMRuleCoverage(detections, siemQueries, litellmConfig)...)
	return sortIssues(issues)
}

// ValidateSIEMContracts validates SIEM coverage and optional schema mappings.
func ValidateSIEMContracts(detections DetectionRulesFile, siemQueries SIEMQueriesFile, litellmConfig catalog.LiteLLMConfig, validateSchema bool) []string {
	issues := validateSIEMRuleCoverage(detections, siemQueries, litellmConfig)
	if validateSchema {
		issues = append(issues, validateSIEMSchemaMappings(siemQueries.SIEMConfig)...)
	}
	return sortIssues(issues)
}

// ApprovedModels returns sorted configured approved model names.
func ApprovedModels(config catalog.LiteLLMConfig) []string {
	return catalog.ApprovedModelNames(config)
}

// CountEnabledDetectionRules reports the enabled detection rule count.
func CountEnabledDetectionRules(detections DetectionRulesFile) int {
	count := 0
	for _, rule := range detections.DetectionRules {
		if rule.Enabled {
			count++
		}
	}
	return count
}

func validateDetectionRules(detections DetectionRulesFile, litellmConfig catalog.LiteLLMConfig) []string {
	issues := make([]string, 0)
	if len(detections.DetectionRules) == 0 {
		issues = append(issues, "demo/config/detection_rules.yaml: detection_rules must contain at least one rule")
		return issues
	}

	if len(ApprovedModels(litellmConfig)) == 0 {
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

func validateSIEMRuleCoverage(detections DetectionRulesFile, siemQueries SIEMQueriesFile, litellmConfig catalog.LiteLLMConfig) []string {
	issues := make([]string, 0)
	if len(siemQueries.SIEMQueries) == 0 {
		issues = append(issues, "demo/config/siem_queries.yaml: siem_queries must contain at least one mapping")
		return issues
	}

	detectionByID := make(map[string]DetectionRule)
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

	if len(ApprovedModels(litellmConfig)) == 0 {
		issues = append(issues, "demo/config/litellm.yaml: model_list must contain at least one approved model")
	}

	return issues
}

func validatePlatformQuery(prefix string, query SIEMPlatformQuery) []string {
	issues := make([]string, 0)
	if strings.TrimSpace(query.Platform) == "" {
		issues = append(issues, fmt.Sprintf("%s: platform is required", prefix))
	}
	if strings.TrimSpace(query.Query) == "" {
		issues = append(issues, fmt.Sprintf("%s: query is required", prefix))
	}
	return issues
}

func validateSigmaQuery(prefix string, query SigmaQueryRule) []string {
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

func validateApprovedModelPlaceholders(prefix string, query SIEMQueryRule) []string {
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

func validateSIEMSchemaMappings(config SIEMConfig) []string {
	issues := make([]string, 0)
	required := []string{"principal.id", "ai.model.id", "ai.request.timestamp", "ai.cost.amount", "ai.tokens.total", "policy.action"}
	fieldMap := make(map[string]FieldMapping)
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

func sortIssues(issues []string) []string {
	if len(issues) == 0 {
		return issues
	}
	sorted := append([]string(nil), issues...)
	sort.Strings(sorted)
	return sorted
}
