// Package policyengine provides ACP-native local policy evaluation workflows.
//
// Purpose:
//   - Define the typed contracts, artifacts, and rule vocabulary for the
//     host-first custom policy engine.
//
// Responsibilities:
//   - Describe policy rule files, evaluation options, and generated summaries.
//   - Keep evaluation result types out of the CLI layer.
//   - Centralize artifact names and supported action/operator constants.
//
// Scope:
//   - Shared policy-engine domain types only.
//
// Usage:
//   - Used by `internal/policyengine` workflows and `acpctl policy eval` /
//     `acpctl validate policy-rules`.
//
// Invariants/Assumptions:
//   - Evaluation remains a local file/stdin workflow, not an always-on service.
//   - Policy actions stay aligned to the normalized evidence schema vocabulary.
package policyengine

const (
	SummaryJSONName       = "summary.json"
	SummaryMarkdownName   = "policy-evaluation-summary.md"
	EvaluatedJSONName     = "evaluated-records.json"
	DecisionsJSONName     = "policy-decisions.json"
	NormalizedJSONName    = "normalized-records.json"
	RawInputJSONName      = "raw-input.json"
	RulesSnapshotName     = "policy-rules.yaml"
	ValidationIssuesName  = "validation-issues.txt"
	InventoryFileName     = "policy-eval-inventory.txt"
	LatestRunPointerName  = "latest-run.txt"
	DefaultOutputSubdir   = "policy-eval"
	DefaultRulesPath      = "demo/config/custom_policy_rules.yaml"
	DefaultActionAllowed  = "allowed"
	ActionBlocked         = "blocked"
	ActionRedacted        = "redacted"
	ActionRateLimited     = "rate_limited"
	ActionError           = "error"
	StageRequest          = "request"
	StageResponse         = "response"
	StageBoth             = "both"
	OperatorEquals        = "equals"
	OperatorNotEquals     = "not_equals"
	OperatorContains      = "contains"
	OperatorContainsAny   = "contains_any"
	OperatorMatchesRegex  = "matches_regex"
	OperatorGreaterThan   = "gt"
	OperatorGreaterEqual  = "gte"
	OperatorLessThan      = "lt"
	OperatorLessEqual     = "lte"
	OperatorExists        = "exists"
	OperatorNotExists     = "not_exists"
	OperatorOneOf         = "one_of"
	OperatorNotOneOf      = "not_one_of"
	OperatorInApproved    = "in_approved_models"
	OperatorNotApproved   = "not_in_approved_models"
	OperatorRoleAllows    = "role_allows_model"
	OperatorRoleDisallows = "role_disallows_model"
)

// Options configures one local policy-evaluation run.
type Options struct {
	RepoRoot     string
	RulesPath    string
	OutputRoot   string
	InputPath    string
	InputPayload []byte
}

// RulesFile captures the tracked custom policy contract.
type RulesFile struct {
	Version       string `yaml:"version"`
	DefaultAction string `yaml:"default_action"`
	Rules         []Rule `yaml:"rules"`
}

// Rule captures one custom policy rule.
type Rule struct {
	RuleID      string         `yaml:"rule_id"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Enabled     bool           `yaml:"enabled"`
	Priority    int            `yaml:"priority"`
	Stage       string         `yaml:"stage"`
	Action      string         `yaml:"action"`
	Reason      string         `yaml:"reason"`
	Tags        []string       `yaml:"tags,omitempty"`
	Entities    []string       `yaml:"entities,omitempty"`
	Match       RuleMatch      `yaml:"match"`
	Redaction   *RedactionRule `yaml:"redaction,omitempty"`
}

// RuleMatch captures top-level all/any matching semantics.
type RuleMatch struct {
	All []Clause `yaml:"all,omitempty"`
	Any []Clause `yaml:"any,omitempty"`
}

// Clause captures one field/operator/value predicate.
type Clause struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Value    any    `yaml:"value,omitempty"`
	Values   []any  `yaml:"values,omitempty"`
}

// RedactionRule captures content redaction behavior for redacted actions.
type RedactionRule struct {
	Target      string `yaml:"target"`
	Match       string `yaml:"match"`
	Replacement string `yaml:"replacement"`
}

// ValidationContext captures repository-backed data used by rule validation and evaluation.
type ValidationContext struct {
	ApprovedModels []string
	DefaultRole    string
	Roles          map[string][]string
}

// Decision captures one matched policy rule and any applied mutation.
type Decision struct {
	RecordIndex   int               `json:"record_index"`
	RuleID        string            `json:"rule_id"`
	RuleName      string            `json:"rule_name"`
	Priority      int               `json:"priority"`
	Stage         string            `json:"stage"`
	Action        string            `json:"action"`
	Reason        string            `json:"reason"`
	Tags          []string          `json:"tags,omitempty"`
	Entities      []string          `json:"entities,omitempty"`
	MatchedFields []string          `json:"matched_fields,omitempty"`
	AppliedRedact *AppliedRedaction `json:"applied_redaction,omitempty"`
}

// AppliedRedaction captures one successful redaction mutation.
type AppliedRedaction struct {
	Target       string `json:"target"`
	Match        string `json:"match"`
	Replacement  string `json:"replacement"`
	ReplaceCount int    `json:"replace_count"`
}

// FinalDecision captures the selected final outcome for a record.
type FinalDecision struct {
	Action string `json:"action"`
	RuleID string `json:"rule_id,omitempty"`
	Rule   string `json:"rule,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// EvaluatedRecord captures the mutated record plus attached policy metadata.
type EvaluatedRecord struct {
	RecordIndex   int            `json:"record_index"`
	Record        map[string]any `json:"record"`
	FinalDecision FinalDecision  `json:"final_decision"`
	MatchedRules  []Decision     `json:"matched_rules"`
}

// Summary captures the machine-readable result of one policy-evaluation run.
type Summary struct {
	RunID                string         `json:"run_id"`
	GeneratedAtUTC       string         `json:"generated_at_utc"`
	RepoRoot             string         `json:"repo_root"`
	RunDirectory         string         `json:"run_directory"`
	RulesPath            string         `json:"rules_path"`
	InputPath            string         `json:"input_path,omitempty"`
	OverallStatus        string         `json:"overall_status"`
	RecordCount          int            `json:"record_count"`
	DecisionCount        int            `json:"decision_count"`
	ValidationIssueCount int            `json:"validation_issue_count"`
	ActionCounts         map[string]int `json:"action_counts"`
	RuleHitCounts        map[string]int `json:"rule_hit_counts"`
	RawInputPath         string         `json:"raw_input_path"`
	EvaluatedPath        string         `json:"evaluated_path"`
	DecisionsPath        string         `json:"decisions_path"`
	NormalizedPath       string         `json:"normalized_path"`
	RulesSnapshotPath    string         `json:"rules_snapshot_path"`
	IssuesPath           string         `json:"issues_path"`
	GeneratedFiles       []string       `json:"generated_files"`
}

// Result returns the generated summary plus evaluated artifacts.
type Result struct {
	Summary          *Summary
	EvaluatedRecords []EvaluatedRecord
	Decisions        []Decision
	Normalized       []map[string]any
	Issues           []string
}
