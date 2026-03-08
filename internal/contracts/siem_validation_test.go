// siem_validation_test.go - SIEM mapping validation coverage.
//
// Purpose:
//   - Verify SIEM parity, placeholder, and schema-mapping validation behavior.
//
// Responsibilities:
//   - Cover mapping mismatch, placeholder, and schema field failures.
//   - Cover successful SIEM contract validation.
//
// Scope:
//   - SIEM contract validation only.
//
// Usage:
//   - Run via `go test ./internal/contracts`.
//
// Invariants/Assumptions:
//   - Expected issues are stable and sorted.
package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestValidateSIEMContractsReportsCoverageAndSchemaIssues(t *testing.T) {
	detections := DetectionRulesFile{
		DetectionRules: []DetectionRule{
			validDetectionRule("DR-001", "Approved Models", true),
			validDetectionRule("DR-002", "Budget Guardrail", true),
		},
	}
	siemQueries := SIEMQueriesFile{
		SIEMQueries: []SIEMQueryRule{
			{
				RuleID:      "DR-001",
				Name:        "Wrong Name",
				Description: "desc",
				Severity:    "medium",
				Category:    "other",
				Enabled:     false,
				Splunk:      SIEMPlatformQuery{},
				ELKKQL:      SIEMPlatformQuery{},
				SentinelKQL: SIEMPlatformQuery{},
				Sigma:       SigmaQueryRule{},
			},
			{
				RuleID:      "DR-001",
				Name:        "Duplicate",
				Description: "desc",
				Severity:    "high",
				Category:    "policy",
				Enabled:     true,
				Splunk:      SIEMPlatformQuery{Platform: "splunk", Query: "search good"},
				ELKKQL:      SIEMPlatformQuery{Platform: "elk", Query: "good"},
				SentinelKQL: SIEMPlatformQuery{Platform: "sentinel", Query: "good"},
				Sigma:       SigmaQueryRule{Title: "sigma", Level: "high", Detection: map[string]any{"selection": "value"}},
			},
		},
		SIEMConfig: SIEMConfig{
			FieldMappings: []FieldMapping{
				{Normalized: "principal.id", Splunk: "principal", Elk: "", Sentinel: "principalId"},
			},
		},
	}

	issues := ValidateSIEMContracts(detections, siemQueries, catalog.LiteLLMConfig{}, true)

	requiredSnippets := []string{
		`duplicate rule_id "DR-001"`,
		`name must match detection rule`,
		`severity must match detection rule`,
		`category must match detection rule`,
		`enabled must match detection rule`,
		`splunk: platform is required`,
		`splunk: query is required`,
		`elk_kql: platform is required`,
		`sentinel_kql: platform is required`,
		`sigma: title is required`,
		`sigma: level is required`,
		`sigma: detection block is required`,
		`missing {{APPROVED_MODELS_SPLUNK}} placeholder`,
		`missing {{APPROVED_MODELS_ELK_OR}} placeholder`,
		`missing {{APPROVED_MODELS_JSON}} placeholder`,
		`missing {{APPROVED_MODELS_SIGMA}} placeholder`,
		`missing SIEM mapping for enabled detection rule "DR-002"`,
		`model_list must contain at least one approved model`,
		`field mapping "principal.id" must define splunk, elk, and sentinel names`,
		`missing siem_config.field_mappings entry for "policy.action"`,
	}
	for _, snippet := range requiredSnippets {
		if !containsIssue(issues, snippet) {
			t.Fatalf("expected issue containing %q, got %v", snippet, issues)
		}
	}
}

func TestValidateSIEMContractsSuccess(t *testing.T) {
	detections := DetectionRulesFile{
		DetectionRules: []DetectionRule{
			validDetectionRule("DR-001", "Approved Models", true),
		},
	}
	siemQueries := SIEMQueriesFile{
		SIEMQueries: []SIEMQueryRule{
			validSIEMRule("DR-001", "Approved Models", "high", "policy"),
		},
		SIEMConfig: SIEMConfig{
			FieldMappings: []FieldMapping{
				{Normalized: "principal.id", Splunk: "principal", Elk: "principal", Sentinel: "principalId"},
				{Normalized: "ai.model.id", Splunk: "model", Elk: "model", Sentinel: "modelId"},
				{Normalized: "ai.request.timestamp", Splunk: "_time", Elk: "@timestamp", Sentinel: "TimeGenerated"},
				{Normalized: "ai.cost.amount", Splunk: "cost", Elk: "cost", Sentinel: "Cost"},
				{Normalized: "ai.tokens.total", Splunk: "tokens", Elk: "tokens", Sentinel: "Tokens"},
				{Normalized: "policy.action", Splunk: "action", Elk: "action", Sentinel: "Action"},
			},
		},
	}
	litellmConfig := catalog.LiteLLMConfig{
		ModelList: []catalog.LiteLLMModel{{ModelName: "gpt-5.2"}},
	}

	issues := ValidateSIEMContracts(detections, siemQueries, litellmConfig, true)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func writeContractFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func validDetectionRule(ruleID string, name string, enabled bool) DetectionRule {
	return DetectionRule{
		RuleID:            ruleID,
		Name:              name,
		Description:       "desc",
		Severity:          "high",
		Category:          "policy",
		OperationalStatus: "validated",
		CoverageTier:      "decision-grade",
		ExpectedSignal:    "signal",
		Enabled:           enabled,
		SQLQuery:          "SELECT APPROVED_MODELS_JSON",
		Remediation:       "fix",
		AutoResponse: DetectionAutoAction{
			Enabled: false,
			Action:  "notify",
		},
	}
}

func validSIEMRule(ruleID string, name string, severity string, category string) SIEMQueryRule {
	return SIEMQueryRule{
		RuleID:      ruleID,
		Name:        name,
		Description: "desc",
		Severity:    severity,
		Category:    category,
		Enabled:     true,
		Splunk: SIEMPlatformQuery{
			Platform: "splunk",
			Query:    "search {{APPROVED_MODELS_SPLUNK}}",
		},
		ELKKQL: SIEMPlatformQuery{
			Platform: "elk",
			Query:    "model:{{APPROVED_MODELS_ELK_OR}}",
		},
		SentinelKQL: SIEMPlatformQuery{
			Platform: "sentinel",
			Query:    "ApprovedModels == {{APPROVED_MODELS_JSON}}",
		},
		Sigma: SigmaQueryRule{
			Title: "sigma",
			Level: "high",
			Detection: map[string]any{
				"selection": "{{APPROVED_MODELS_SIGMA}}",
			},
		},
	}
}

func containsIssue(issues []string, snippet string) bool {
	for _, issue := range issues {
		if strings.Contains(issue, snippet) {
			return true
		}
	}
	return false
}
