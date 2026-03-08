// detection_validation_test.go - Detection-rule validation coverage.
//
// Purpose:
//   - Verify detection contract validation surfaces structural issues deterministically.
//
// Responsibilities:
//   - Cover empty-file and invalid-rule failures.
//   - Cover successful validation and sorted issue output.
//
// Scope:
//   - Detection contract validation only.
//
// Usage:
//   - Run via `go test ./internal/contracts`.
//
// Invariants/Assumptions:
//   - Validation stays side-effect free for equivalent inputs.
package contracts

import (
	"slices"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestValidateDetectionContractsRequiresRulesAndApprovedModels(t *testing.T) {
	issues := ValidateDetectionContracts(DetectionRulesFile{}, SIEMQueriesFile{
		SIEMQueries: []SIEMQueryRule{{RuleID: "DR-001"}},
	}, catalog.LiteLLMConfig{})

	if len(issues) != 3 {
		t.Fatalf("expected three issues, got %v", issues)
	}
	if !strings.Contains(issues[0], "detection_rules must contain at least one rule") && !strings.Contains(issues[1], "detection_rules must contain at least one rule") {
		t.Fatalf("expected missing detection-rules issue, got %v", issues)
	}
}

func TestValidateDetectionContractsReportsInvalidRuleFields(t *testing.T) {
	detections := DetectionRulesFile{
		DetectionRules: []DetectionRule{
			{
				RuleID:            "bad",
				Name:              "",
				Description:       "",
				Severity:          "critical",
				Category:          "",
				OperationalStatus: "pending",
				CoverageTier:      "gold",
				ExpectedSignal:    "",
				Enabled:           true,
				SQLQuery:          "",
				Remediation:       "",
				AutoResponse: DetectionAutoAction{
					Enabled: true,
				},
			},
			{
				RuleID:            "DR-001",
				Name:              "Duplicate",
				Description:       "desc",
				Severity:          "low",
				Category:          "policy",
				OperationalStatus: "example",
				CoverageTier:      "decision-grade",
				ExpectedSignal:    "signal",
				Enabled:           true,
				SQLQuery:          "SELECT 1",
				Remediation:       "fix",
			},
			{
				RuleID:            "DR-001",
				Name:              "Duplicate",
				Description:       "desc",
				Severity:          "medium",
				Category:          "policy",
				OperationalStatus: "validated",
				CoverageTier:      "demo",
				ExpectedSignal:    "signal",
				Enabled:           false,
				SQLQuery:          "SELECT APPROVED_MODELS_JSON",
				Remediation:       "fix",
			},
		},
	}

	issues := ValidateDetectionContracts(detections, SIEMQueriesFile{
		SIEMQueries: []SIEMQueryRule{{RuleID: "DR-002"}},
	}, catalog.LiteLLMConfig{})

	requiredSnippets := []string{
		`rule_id must match DR-###`,
		`name is required`,
		`severity must be low, medium, or high`,
		`category is required`,
		`description is required`,
		`remediation is required`,
		`operational_status must be validated or example`,
		`coverage_tier must be decision-grade or demo`,
		`decision-grade rules must also be operational_status=validated`,
		`expected_signal is required`,
		`enabled rules must include sql_query`,
		`auto_response.enabled requires auto_response.action`,
		`DR-001 sql_query must use APPROVED_MODELS_JSON placeholder`,
		`duplicate rule_id "DR-001"`,
		`duplicate rule name "Duplicate"`,
		`model_list must contain at least one approved model`,
	}

	for _, snippet := range requiredSnippets {
		if !containsIssue(issues, snippet) {
			t.Fatalf("expected issue containing %q, got %v", snippet, issues)
		}
	}
	if !slices.IsSorted(issues) {
		t.Fatalf("expected sorted issues, got %v", issues)
	}
}

func TestValidateDetectionContractsSuccess(t *testing.T) {
	detections := DetectionRulesFile{
		DetectionRules: []DetectionRule{
			validDetectionRule("DR-001", "Approved Models", true),
			validDetectionRule("DR-002", "Budget Guardrail", true),
		},
	}
	siemQueries := SIEMQueriesFile{
		SIEMQueries: []SIEMQueryRule{
			validSIEMRule("DR-001", "Approved Models", "high", "policy"),
			validSIEMRule("DR-002", "Budget Guardrail", "high", "policy"),
		},
	}
	litellmConfig := catalog.LiteLLMConfig{
		ModelList: []catalog.LiteLLMModel{{ModelName: "gpt-5.2"}},
	}

	issues := ValidateDetectionContracts(detections, siemQueries, litellmConfig)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}
