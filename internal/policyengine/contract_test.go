// contract_test.go - Tests for the custom policy rule contract helpers.
//
// Purpose:
//   - Raise confidence in rule-contract validation and helper semantics.
//
// Responsibilities:
//   - Cover operator-specific validation branches.
//   - Cover redaction contract rules and context-aware validator helpers.
//   - Cover deterministic helper ordering and support predicates.
//
// Scope:
//   - internal/policyengine contract-helper tests only.
//
// Usage:
//   - Run via `go test ./internal/policyengine`.
//
// Invariants/Assumptions:
//   - Tests remain side-effect free and deterministic.
package policyengine

import (
	"strings"
	"testing"
)

func TestContractValidationHelpersCoverOperatorBranches(t *testing.T) {
	ctx := ValidationContext{ApprovedModels: []string{"openai-gpt5.2"}, Roles: map[string][]string{"developer": {"openai-gpt5.2"}}}
	rule := Rule{
		Action: ActionBlocked,
		Match: RuleMatch{All: []Clause{
			{Field: "principal.role", Operator: OperatorInApproved},
			{Field: "principal.role", Operator: OperatorRoleAllows},
		}},
	}
	issues := validateRuleAgainstContext("rules[0]", rule, ctx)
	joined := strings.Join(issues, "\n")
	for _, needle := range []string{"in_approved_models only supports field ai.model.id", "role_allows_model only supports field ai.model.id", "RBAC default role is required"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("validateRuleAgainstContext() missing %q\nissues=%s", needle, joined)
		}
	}

	clauseIssues := strings.Join(validateClause("rules[0]", Clause{Field: "ai.tokens.prompt", Operator: OperatorContains, Value: "bad"}), "\n")
	if !strings.Contains(clauseIssues, "only supports string fields") {
		t.Fatalf("validateClause(string mismatch) = %s", clauseIssues)
	}
	clauseIssues = strings.Join(validateClause("rules[0]", Clause{Field: "principal.role", Operator: OperatorGreaterThan, Value: 1}), "\n")
	if !strings.Contains(clauseIssues, "only supports numeric fields") {
		t.Fatalf("validateClause(number mismatch) = %s", clauseIssues)
	}
	clauseIssues = strings.Join(validateClause("rules[0]", Clause{Field: "principal.role", Operator: OperatorContainsAny}), "\n")
	if !strings.Contains(clauseIssues, "requires values") {
		t.Fatalf("validateClause(missing values) = %s", clauseIssues)
	}
	clauseIssues = strings.Join(validateClause("rules[0]", Clause{Field: "principal.role", Operator: OperatorExists, Value: "x"}), "\n")
	if !strings.Contains(clauseIssues, "does not accept value/values") {
		t.Fatalf("validateClause(exists with value) = %s", clauseIssues)
	}
	clauseIssues = strings.Join(validateClause("rules[0]", Clause{Field: "principal.role", Operator: OperatorOneOf, Values: []any{"developer", ""}}), "\n")
	if !strings.Contains(clauseIssues, "values[1] must not be blank") {
		t.Fatalf("validateClause(blank value) = %s", clauseIssues)
	}
	clauseIssues = strings.Join(validateClause("rules[0]", Clause{Field: "principal.role", Operator: OperatorEquals}), "\n")
	if !strings.Contains(clauseIssues, "requires value") {
		t.Fatalf("validateClause(missing value) = %s", clauseIssues)
	}

	redactionIssues := strings.Join(validateRedaction("rules[0]", Rule{Action: ActionBlocked, Redaction: &RedactionRule{Target: "response.content", Match: `\d+`, Replacement: "x"}}), "\n")
	if !strings.Contains(redactionIssues, "redaction is only valid for action=redacted") {
		t.Fatalf("validateRedaction(non-redacted) = %s", redactionIssues)
	}
	redactionIssues = strings.Join(validateRedaction("rules[0]", Rule{Action: ActionRedacted}), "\n")
	if !strings.Contains(redactionIssues, "action=redacted requires redaction config") {
		t.Fatalf("validateRedaction(missing config) = %s", redactionIssues)
	}
}

func TestContractHelperOrderingAndSupportPredicates(t *testing.T) {
	sorted := sortRules([]Rule{{RuleID: "PR-010", Priority: 20}, {RuleID: "PR-001", Priority: 10}, {RuleID: "PR-002", Priority: 10}})
	if got := []string{sorted[0].RuleID, sorted[1].RuleID, sorted[2].RuleID}; strings.Join(got, ",") != "PR-001,PR-002,PR-010" {
		t.Fatalf("sortRules() order = %v", got)
	}
	if !isSupportedOperator(OperatorRoleDisallows) || isSupportedOperator("bogus") {
		t.Fatalf("isSupportedOperator() returned unexpected values")
	}
	if actionRank(ActionError) <= actionRank(ActionBlocked) || actionRank(ActionBlocked) <= actionRank(ActionRedacted) || actionRank(ActionRedacted) <= actionRank(ActionRateLimited) || actionRank(ActionRateLimited) <= actionRank(DefaultActionAllowed) {
		t.Fatalf("actionRank() precedence order unexpected")
	}
	if got := dedupeSortedStrings([]string{"beta", "", "alpha", "beta"}); strings.Join(got, ",") != "alpha,beta" {
		t.Fatalf("dedupeSortedStrings() = %v, want [alpha beta]", got)
	}
}
