// Package policyengine provides ACP-native local policy evaluation workflows.
//
// Purpose:
//   - Load and validate the tracked custom policy rule contract outside the CLI
//     layer.
//
// Responsibilities:
//   - Parse the repository-tracked YAML rule file.
//   - Enforce structural validation for rule IDs, actions, stages, predicates,
//     and redaction contracts.
//   - Keep validation deterministic for operator workflows and tests.
//
// Scope:
//   - Rule-contract loading and validation only.
//
// Usage:
//   - Used by `acpctl validate policy-rules` and `acpctl policy eval`.
//
// Invariants/Assumptions:
//   - Validation remains side-effect free.
//   - Rule IDs are unique and match the tracked policy-rule naming contract.
package policyengine

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
	"gopkg.in/yaml.v3"
)

var ruleIDPattern = regexp.MustCompile(`^PR-\d{3}$`)

var fieldKinds = map[string]string{
	"principal.id":           "string",
	"principal.type":         "string",
	"principal.email":        "string",
	"principal.role":         "string",
	"ai.model.id":            "string",
	"ai.provider":            "string",
	"ai.request.id":          "string",
	"ai.request.timestamp":   "string",
	"ai.tokens.prompt":       "number",
	"ai.tokens.completion":   "number",
	"ai.tokens.total":        "number",
	"ai.cost.amount":         "number",
	"request.content":        "string",
	"response.content":       "string",
	"source.type":            "string",
	"source.service.name":    "string",
	"correlation.session.id": "string",
}

// LoadRulesFile loads one custom policy rule file from disk.
func LoadRulesFile(path string) (RulesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RulesFile{}, fmt.Errorf("read custom policy rules %s: %w", path, err)
	}
	var doc RulesFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return RulesFile{}, fmt.Errorf("parse custom policy rules %s: %w", path, err)
	}
	if strings.TrimSpace(doc.DefaultAction) == "" {
		doc.DefaultAction = DefaultActionAllowed
	}
	return doc, nil
}

// ValidateRulesFile validates the full custom policy rule contract.
func ValidateRulesFile(doc RulesFile, ctx ValidationContext) []string {
	issues := validationissues.NewIssues()
	if strings.TrimSpace(doc.Version) == "" {
		issues.Add("demo/config/custom_policy_rules.yaml: version is required")
	}
	if !isSupportedAction(doc.DefaultAction) {
		issues.Addf("demo/config/custom_policy_rules.yaml: default_action must be one of allowed, blocked, redacted, rate_limited, error (got %q)", doc.DefaultAction)
	}
	if len(doc.Rules) == 0 {
		issues.Add("demo/config/custom_policy_rules.yaml: rules must contain at least one rule")
		return issues.Sorted()
	}

	seenIDs := make(map[string]struct{}, len(doc.Rules))
	seenNames := make(map[string]struct{}, len(doc.Rules))
	for index, rule := range doc.Rules {
		prefix := fmt.Sprintf("demo/config/custom_policy_rules.yaml: rules[%d]", index)
		ruleID := strings.TrimSpace(rule.RuleID)
		name := strings.TrimSpace(rule.Name)
		if !ruleIDPattern.MatchString(ruleID) {
			issues.Addf("%s: rule_id must match PR-### (got %q)", prefix, rule.RuleID)
		}
		if _, ok := seenIDs[ruleID]; ok {
			issues.Addf("%s: duplicate rule_id %q", prefix, ruleID)
		}
		seenIDs[ruleID] = struct{}{}
		if name == "" {
			issues.Addf("%s: name is required", prefix)
		} else {
			if _, ok := seenNames[name]; ok {
				issues.Addf("%s: duplicate name %q", prefix, name)
			}
			seenNames[name] = struct{}{}
		}
		if strings.TrimSpace(rule.Description) == "" {
			issues.Addf("%s: description is required", prefix)
		}
		if rule.Priority <= 0 {
			issues.Addf("%s: priority must be > 0", prefix)
		}
		if !isSupportedStage(rule.Stage) {
			issues.Addf("%s: stage must be request, response, or both", prefix)
		}
		if !isSupportedAction(rule.Action) {
			issues.Addf("%s: action must be one of allowed, blocked, redacted, rate_limited, error", prefix)
		}
		if strings.TrimSpace(rule.Reason) == "" {
			issues.Addf("%s: reason is required", prefix)
		}
		if len(rule.Match.All) == 0 && len(rule.Match.Any) == 0 {
			issues.Addf("%s: match must define at least one all/any clause", prefix)
		}
		for clauseIndex, clause := range rule.Match.All {
			issues.Extend(validateClause(prefix+fmt.Sprintf(" match.all[%d]", clauseIndex), clause))
		}
		for clauseIndex, clause := range rule.Match.Any {
			issues.Extend(validateClause(prefix+fmt.Sprintf(" match.any[%d]", clauseIndex), clause))
		}
		issues.Extend(validateRuleAgainstContext(prefix, rule, ctx))
		issues.Extend(validateRedaction(prefix, rule))
	}
	return issues.Sorted()
}

func validateRuleAgainstContext(prefix string, rule Rule, ctx ValidationContext) []string {
	issues := validationissues.NewIssues()
	if len(ctx.ApprovedModels) == 0 {
		issues.Add(prefix + ": approved model catalog is empty")
	}
	for _, clauses := range [][]Clause{rule.Match.All, rule.Match.Any} {
		for _, clause := range clauses {
			switch normalizeOperator(clause.Operator) {
			case OperatorInApproved, OperatorNotApproved:
				if strings.TrimSpace(clause.Field) != "ai.model.id" {
					issues.Addf("%s: %s only supports field ai.model.id", prefix, normalizeOperator(clause.Operator))
				}
			case OperatorRoleAllows, OperatorRoleDisallows:
				if strings.TrimSpace(clause.Field) != "ai.model.id" {
					issues.Addf("%s: %s only supports field ai.model.id", prefix, normalizeOperator(clause.Operator))
				}
				if strings.TrimSpace(ctx.DefaultRole) == "" {
					issues.Add(prefix + ": RBAC default role is required for role-aware policy operators")
				}
			}
		}
	}
	return issues.ToSlice()
}

func validateClause(prefix string, clause Clause) []string {
	issues := validationissues.NewIssues()
	field := strings.TrimSpace(clause.Field)
	operator := normalizeOperator(clause.Operator)
	if _, ok := fieldKinds[field]; !ok {
		issues.Addf("%s: unsupported field %q", prefix, clause.Field)
		return issues.ToSlice()
	}
	if !isSupportedOperator(operator) {
		issues.Addf("%s: unsupported operator %q", prefix, clause.Operator)
		return issues.ToSlice()
	}

	switch operator {
	case OperatorExists, OperatorNotExists, OperatorInApproved, OperatorNotApproved, OperatorRoleAllows, OperatorRoleDisallows:
		if clause.Value != nil || len(clause.Values) > 0 {
			issues.Addf("%s: %s does not accept value/values", prefix, operator)
		}
	case OperatorContainsAny, OperatorOneOf, OperatorNotOneOf:
		if len(clause.Values) == 0 {
			issues.Addf("%s: %s requires values", prefix, operator)
		}
	default:
		if clause.Value == nil {
			issues.Addf("%s: %s requires value", prefix, operator)
		}
	}

	kind := fieldKinds[field]
	switch operator {
	case OperatorContains, OperatorContainsAny, OperatorMatchesRegex:
		if kind != "string" {
			issues.Addf("%s: %s only supports string fields", prefix, operator)
		}
	case OperatorGreaterThan, OperatorGreaterEqual, OperatorLessThan, OperatorLessEqual:
		if kind != "number" {
			issues.Addf("%s: %s only supports numeric fields", prefix, operator)
		}
	}

	if operator == OperatorMatchesRegex && clause.Value != nil {
		if _, err := regexp.Compile(fmt.Sprintf("%v", clause.Value)); err != nil {
			issues.Addf("%s: invalid regex %q: %v", prefix, clause.Value, err)
		}
	}

	for valueIndex, value := range clause.Values {
		if strings.TrimSpace(fmt.Sprintf("%v", value)) == "" {
			issues.Addf("%s: values[%d] must not be blank", prefix, valueIndex)
		}
	}
	return issues.ToSlice()
}

func validateRedaction(prefix string, rule Rule) []string {
	issues := validationissues.NewIssues()
	if normalizeAction(rule.Action) != ActionRedacted {
		if rule.Redaction != nil {
			issues.Addf("%s: redaction is only valid for action=redacted", prefix)
		}
		return issues.ToSlice()
	}
	if rule.Redaction == nil {
		issues.Addf("%s: action=redacted requires redaction config", prefix)
		return issues.ToSlice()
	}
	target := strings.TrimSpace(rule.Redaction.Target)
	if target != "request.content" && target != "response.content" {
		issues.Addf("%s: redaction.target must be request.content or response.content", prefix)
	}
	if strings.TrimSpace(rule.Redaction.Match) == "" {
		issues.Addf("%s: redaction.match is required", prefix)
	} else if _, err := regexp.Compile(rule.Redaction.Match); err != nil {
		issues.Addf("%s: invalid redaction.match %q: %v", prefix, rule.Redaction.Match, err)
	}
	if strings.TrimSpace(rule.Redaction.Replacement) == "" {
		issues.Addf("%s: redaction.replacement is required", prefix)
	}
	return issues.ToSlice()
}

func sortRules(rules []Rule) []Rule {
	cloned := append([]Rule(nil), rules...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Priority != cloned[j].Priority {
			return cloned[i].Priority < cloned[j].Priority
		}
		return strings.TrimSpace(cloned[i].RuleID) < strings.TrimSpace(cloned[j].RuleID)
	})
	return cloned
}

func isSupportedAction(value string) bool {
	switch normalizeAction(value) {
	case DefaultActionAllowed, ActionBlocked, ActionRedacted, ActionRateLimited, ActionError:
		return true
	default:
		return false
	}
}

func normalizeAction(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isSupportedStage(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case StageRequest, StageResponse, StageBoth:
		return true
	default:
		return false
	}
}

func normalizeOperator(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isSupportedOperator(value string) bool {
	switch normalizeOperator(value) {
	case OperatorEquals,
		OperatorNotEquals,
		OperatorContains,
		OperatorContainsAny,
		OperatorMatchesRegex,
		OperatorGreaterThan,
		OperatorGreaterEqual,
		OperatorLessThan,
		OperatorLessEqual,
		OperatorExists,
		OperatorNotExists,
		OperatorOneOf,
		OperatorNotOneOf,
		OperatorInApproved,
		OperatorNotApproved,
		OperatorRoleAllows,
		OperatorRoleDisallows:
		return true
	default:
		return false
	}
}

func dedupeSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}
