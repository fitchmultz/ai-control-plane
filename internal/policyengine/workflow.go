// Package policyengine provides ACP-native local policy evaluation workflows.
//
// Purpose:
//   - Execute local request/response policy evaluation runs that apply tracked
//     custom guardrail rules and emit auditable artifacts.
//
// Responsibilities:
//   - Load repository-backed rule, RBAC, model, and schema context.
//   - Evaluate request/response records against tracked policy rules.
//   - Persist auditable decision artifacts via internal/artifactrun.
//
// Scope:
//   - Local file/stdin policy evaluation only.
//
// Usage:
//   - Used by `acpctl policy eval`.
//
// Invariants/Assumptions:
//   - Evaluation remains host-first and offline/local; ACP does not claim a
//     supported always-on inline proxy policy service here.
//   - Artifacts under demo/logs/evidence remain private local outputs.
package policyengine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/ingest"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/rbac"
	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
	"gopkg.in/yaml.v3"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

var normalizedCopyFields = []string{
	"principal.id",
	"principal.type",
	"principal.email",
	"principal.role",
	"ai.model.id",
	"ai.provider",
	"ai.request.id",
	"ai.request.timestamp",
	"ai.tokens.prompt",
	"ai.tokens.completion",
	"ai.tokens.total",
	"ai.cost.amount",
	"ai.cost.currency",
	"correlation.trace.id",
	"correlation.span.id",
	"correlation.session.id",
	"source.type",
	"source.service.name",
	"source.service.version",
	"source.deployment.environment",
}

// Evaluate executes one local policy-evaluation run.
func Evaluate(_ context.Context, opts Options) (*Result, error) {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if len(opts.InputPayload) == 0 {
		return nil, fmt.Errorf("input payload is required")
	}
	if strings.TrimSpace(opts.RulesPath) == "" {
		opts.RulesPath = repopath.ResolveRepoPath(opts.RepoRoot, DefaultRulesPath)
	}
	if strings.TrimSpace(opts.OutputRoot) == "" {
		opts.OutputRoot = repopath.DemoLogsPath(opts.RepoRoot, "evidence", DefaultOutputSubdir)
	}

	rulesDoc, err := LoadRulesFile(opts.RulesPath)
	if err != nil {
		return nil, err
	}
	validationCtx, err := LoadValidationContext(opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	if issues := ValidateRulesFile(rulesDoc, validationCtx); len(issues) > 0 {
		return nil, fmt.Errorf("custom policy rules validation failed:\n%s", strings.Join(issues, "\n"))
	}
	schema, err := ingest.LoadSchema(repopath.DemoConfigPath(opts.RepoRoot, "normalized_schema.yaml"))
	if err != nil {
		return nil, err
	}

	records, rawDocument, err := parseInputRecords(opts.InputPayload)
	if err != nil {
		return nil, err
	}
	rules := sortRules(enabledRules(rulesDoc.Rules))
	contentScanConfigured := hasContentInspectionRule(rules)

	issues := validationissues.NewIssues()
	evaluated := make([]EvaluatedRecord, 0, len(records))
	decisions := make([]Decision, 0)
	normalized := make([]map[string]any, 0, len(records))
	actionCounts := map[string]int{}
	ruleHitCounts := map[string]int{}

	for index, sourceRecord := range records {
		working := normalizeInputRecord(deepCloneMap(sourceRecord), validationCtx)
		issues.Extend(validateRecordContext(index, working, validationCtx))

		recordDecisions := make([]Decision, 0)
		for _, rule := range rules {
			decision, matched, err := evaluateRule(index, working, rule, validationCtx)
			if err != nil {
				issues.Addf("record[%d]: evaluate rule %s: %v", index, rule.RuleID, err)
				continue
			}
			if !matched {
				continue
			}
			recordDecisions = append(recordDecisions, decision)
			decisions = append(decisions, decision)
			ruleHitCounts[decision.RuleID]++
		}

		finalDecision := selectFinalDecision(rulesDoc.DefaultAction, recordDecisions)
		actionCounts[finalDecision.Action]++

		evaluatedRecord := deepCloneMap(working)
		setPath(evaluatedRecord, "policy_engine.final_action", finalDecision.Action)
		if strings.TrimSpace(finalDecision.RuleID) != "" {
			setPath(evaluatedRecord, "policy_engine.final_rule_id", finalDecision.RuleID)
		}
		if strings.TrimSpace(finalDecision.Rule) != "" {
			setPath(evaluatedRecord, "policy_engine.final_rule", finalDecision.Rule)
		}
		if strings.TrimSpace(finalDecision.Reason) != "" {
			setPath(evaluatedRecord, "policy_engine.final_reason", finalDecision.Reason)
		}
		setPath(evaluatedRecord, "policy_engine.matched_rule_count", len(recordDecisions))

		normalizedRecord := buildNormalizedRecord(working, finalDecision, recordDecisions, contentScanConfigured)
		issues.Extend(schema.ValidateRecord(normalizedRecord, index))

		evaluated = append(evaluated, EvaluatedRecord{
			RecordIndex:   index,
			Record:        evaluatedRecord,
			FinalDecision: finalDecision,
			MatchedRules:  append([]Decision(nil), recordDecisions...),
		})
		normalized = append(normalized, normalizedRecord)
	}

	issueList := issues.Sorted()
	run, err := artifactrun.Create(opts.OutputRoot, DefaultOutputSubdir, nowUTC())
	if err != nil {
		return nil, err
	}
	summary := &Summary{
		RunID:                run.ID,
		GeneratedAtUTC:       nowUTC().Format(time.RFC3339),
		RepoRoot:             opts.RepoRoot,
		RunDirectory:         run.Directory,
		RulesPath:            opts.RulesPath,
		InputPath:            opts.InputPath,
		OverallStatus:        "PASS",
		RecordCount:          len(records),
		DecisionCount:        len(decisions),
		ValidationIssueCount: len(issueList),
		ActionCounts:         cloneIntMap(actionCounts),
		RuleHitCounts:        cloneIntMap(ruleHitCounts),
		RawInputPath:         filepath.Join(run.Directory, RawInputJSONName),
		EvaluatedPath:        filepath.Join(run.Directory, EvaluatedJSONName),
		DecisionsPath:        filepath.Join(run.Directory, DecisionsJSONName),
		NormalizedPath:       filepath.Join(run.Directory, NormalizedJSONName),
		RulesSnapshotPath:    filepath.Join(run.Directory, RulesSnapshotName),
		IssuesPath:           filepath.Join(run.Directory, ValidationIssuesName),
	}
	if len(issueList) > 0 {
		summary.OverallStatus = "FAIL"
	}
	if err := writeArtifacts(run.Directory, rawDocument, rulesDoc, evaluated, decisions, normalized, issueList, summary); err != nil {
		return nil, err
	}
	files, err := artifactrun.Finalize(run.Directory, opts.OutputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryFileName,
		LatestPointers: []string{LatestRunPointerName},
	})
	if err != nil {
		return nil, err
	}
	summary.GeneratedFiles = files
	if err := writeArtifacts(run.Directory, rawDocument, rulesDoc, evaluated, decisions, normalized, issueList, summary); err != nil {
		return nil, err
	}
	return &Result{
		Summary:          summary,
		EvaluatedRecords: evaluated,
		Decisions:        decisions,
		Normalized:       normalized,
		Issues:           issueList,
	}, nil
}

// LoadValidationContext loads repository-backed approved-model and RBAC context.
func LoadValidationContext(repoRoot string) (ValidationContext, error) {
	modelCatalog, err := catalog.LoadModelCatalog(repopath.DemoConfigPath(repoRoot, "model_catalog.yaml"))
	if err != nil {
		return ValidationContext{}, fmt.Errorf("load model catalog: %w", err)
	}
	rbacConfig, err := rbac.LoadFile(repopath.DemoConfigPath(repoRoot, "roles.yaml"))
	if err != nil {
		return ValidationContext{}, fmt.Errorf("load RBAC config: %w", err)
	}
	approvedModels := modelCatalog.OnlineAliases()
	roles := make(map[string][]string, len(rbacConfig.Roles))
	for _, roleName := range rbacConfig.RoleNames() {
		roles[roleName] = rbacConfig.ModelsForRole(roleName, approvedModels)
	}
	return ValidationContext{
		ApprovedModels: approvedModels,
		DefaultRole:    rbacConfig.DefaultRoleName(),
		Roles:          roles,
	}, nil
}

func enabledRules(rules []Rule) []Rule {
	filtered := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		if rule.Enabled {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func hasContentInspectionRule(rules []Rule) bool {
	for _, rule := range rules {
		for _, clauses := range [][]Clause{rule.Match.All, rule.Match.Any} {
			for _, clause := range clauses {
				if clause.Field == "request.content" || clause.Field == "response.content" {
					return true
				}
			}
		}
	}
	return false
}

func validateRecordContext(index int, record map[string]any, ctx ValidationContext) []string {
	issues := validationissues.NewIssues()
	if roleValue, ok := lookupPath(record, "principal.role"); ok {
		role := strings.TrimSpace(fmt.Sprintf("%v", roleValue))
		if role != "" {
			if _, ok := ctx.Roles[role]; !ok {
				issues.Addf("record[%d]: principal.role %q is not defined in demo/config/roles.yaml", index, role)
			}
		}
	}
	return issues.ToSlice()
}

func parseInputRecords(data []byte) ([]map[string]any, any, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parse input json: %w", err)
	}
	switch typed := raw.(type) {
	case []any:
		records := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			record, err := asRecord(item)
			if err != nil {
				return nil, nil, err
			}
			records = append(records, record)
		}
		return records, raw, nil
	case map[string]any:
		if recordsValue, ok := typed["records"]; ok {
			items, ok := recordsValue.([]any)
			if !ok {
				return nil, nil, fmt.Errorf("records must be a JSON array")
			}
			records := make([]map[string]any, 0, len(items))
			for _, item := range items {
				record, err := asRecord(item)
				if err != nil {
					return nil, nil, err
				}
				records = append(records, record)
			}
			return records, raw, nil
		}
		record, err := asRecord(typed)
		if err != nil {
			return nil, nil, err
		}
		return []map[string]any{record}, raw, nil
	default:
		return nil, nil, fmt.Errorf("input must be a JSON object or array")
	}
}

func asRecord(value any) (map[string]any, error) {
	record, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("record must be a JSON object")
	}
	return deepCloneMap(record), nil
}

func normalizeInputRecord(record map[string]any, ctx ValidationContext) map[string]any {
	if value, ok := lookupPath(record, "principal.type"); !ok || isBlankValue(value) {
		setPath(record, "principal.type", "unknown")
	}
	if value, ok := lookupPath(record, "principal.role"); !ok || isBlankValue(value) {
		if strings.TrimSpace(ctx.DefaultRole) != "" {
			setPath(record, "principal.role", ctx.DefaultRole)
		}
	}
	prompt, promptOK := numberValue(record, "ai.tokens.prompt")
	completion, completionOK := numberValue(record, "ai.tokens.completion")
	if _, ok := lookupPath(record, "ai.tokens.total"); !ok && (promptOK || completionOK) {
		setPath(record, "ai.tokens.total", int(prompt+completion))
	}
	if amount, ok := lookupPath(record, "ai.cost.amount"); ok && !isBlankValue(amount) {
		if _, exists := lookupPath(record, "ai.cost.currency"); !exists {
			setPath(record, "ai.cost.currency", "USD")
		}
	}
	return record
}

func evaluateRule(index int, record map[string]any, rule Rule, ctx ValidationContext) (Decision, bool, error) {
	allMatched := true
	matchedFields := make([]string, 0, len(rule.Match.All)+len(rule.Match.Any))
	for _, clause := range rule.Match.All {
		matched, err := clauseMatches(record, clause, ctx)
		if err != nil {
			return Decision{}, false, err
		}
		if !matched {
			allMatched = false
			break
		}
		matchedFields = append(matchedFields, clause.Field)
	}
	if !allMatched {
		return Decision{}, false, nil
	}

	anyMatched := len(rule.Match.Any) == 0
	for _, clause := range rule.Match.Any {
		matched, err := clauseMatches(record, clause, ctx)
		if err != nil {
			return Decision{}, false, err
		}
		if matched {
			anyMatched = true
			matchedFields = append(matchedFields, clause.Field)
		}
	}
	if !anyMatched {
		return Decision{}, false, nil
	}

	decision := Decision{
		RecordIndex:   index,
		RuleID:        rule.RuleID,
		RuleName:      rule.Name,
		Priority:      rule.Priority,
		Stage:         rule.Stage,
		Action:        normalizeAction(rule.Action),
		Reason:        strings.TrimSpace(rule.Reason),
		Tags:          append([]string(nil), rule.Tags...),
		Entities:      dedupeSortedStrings(rule.Entities),
		MatchedFields: dedupeSortedStrings(matchedFields),
	}
	if decision.Action == ActionRedacted && rule.Redaction != nil {
		redaction, err := applyRedaction(record, *rule.Redaction)
		if err != nil {
			return Decision{}, false, err
		}
		decision.AppliedRedact = redaction
	}
	return decision, true, nil
}

func clauseMatches(record map[string]any, clause Clause, ctx ValidationContext) (bool, error) {
	field := strings.TrimSpace(clause.Field)
	operator := normalizeOperator(clause.Operator)
	value, ok := lookupPath(record, field)

	switch operator {
	case OperatorExists:
		return ok && !isBlankValue(value), nil
	case OperatorNotExists:
		return !ok || isBlankValue(value), nil
	case OperatorInApproved:
		if !ok {
			return false, nil
		}
		return containsString(ctx.ApprovedModels, fmt.Sprintf("%v", value)), nil
	case OperatorNotApproved:
		if !ok {
			return false, nil
		}
		return !containsString(ctx.ApprovedModels, fmt.Sprintf("%v", value)), nil
	case OperatorRoleAllows, OperatorRoleDisallows:
		model, ok := lookupPath(record, "ai.model.id")
		if !ok {
			return false, nil
		}
		role, _ := lookupPath(record, "principal.role")
		resolvedRole := strings.TrimSpace(fmt.Sprintf("%v", role))
		if resolvedRole == "" {
			resolvedRole = strings.TrimSpace(ctx.DefaultRole)
		}
		allowed := ctx.Roles[resolvedRole]
		matched := containsString(allowed, fmt.Sprintf("%v", model))
		if operator == OperatorRoleAllows {
			return matched, nil
		}
		return !matched, nil
	}

	if !ok {
		return false, nil
	}

	switch operator {
	case OperatorEquals:
		return equalValues(value, clause.Value), nil
	case OperatorNotEquals:
		return !equalValues(value, clause.Value), nil
	case OperatorContains:
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), strings.ToLower(fmt.Sprintf("%v", clause.Value))), nil
	case OperatorContainsAny:
		content := strings.ToLower(fmt.Sprintf("%v", value))
		for _, candidate := range clause.Values {
			if strings.Contains(content, strings.ToLower(fmt.Sprintf("%v", candidate))) {
				return true, nil
			}
		}
		return false, nil
	case OperatorMatchesRegex:
		compiled, err := regexp.Compile(fmt.Sprintf("%v", clause.Value))
		if err != nil {
			return false, fmt.Errorf("compile regex for %s: %w", clause.Field, err)
		}
		return compiled.MatchString(fmt.Sprintf("%v", value)), nil
	case OperatorGreaterThan, OperatorGreaterEqual, OperatorLessThan, OperatorLessEqual:
		left, ok := asFloat64(value)
		if !ok {
			return false, nil
		}
		right, ok := asFloat64(clause.Value)
		if !ok {
			return false, fmt.Errorf("numeric operator %s requires numeric value", operator)
		}
		switch operator {
		case OperatorGreaterThan:
			return left > right, nil
		case OperatorGreaterEqual:
			return left >= right, nil
		case OperatorLessThan:
			return left < right, nil
		default:
			return left <= right, nil
		}
	case OperatorOneOf:
		for _, candidate := range clause.Values {
			if equalValues(value, candidate) {
				return true, nil
			}
		}
		return false, nil
	case OperatorNotOneOf:
		for _, candidate := range clause.Values {
			if equalValues(value, candidate) {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported operator %s", operator)
	}
}

func applyRedaction(record map[string]any, redaction RedactionRule) (*AppliedRedaction, error) {
	value, ok := lookupPath(record, redaction.Target)
	if !ok {
		return &AppliedRedaction{Target: redaction.Target, Match: redaction.Match, Replacement: redaction.Replacement, ReplaceCount: 0}, nil
	}
	content := fmt.Sprintf("%v", value)
	compiled, err := regexp.Compile(redaction.Match)
	if err != nil {
		return nil, fmt.Errorf("compile redaction regex: %w", err)
	}
	matches := compiled.FindAllStringIndex(content, -1)
	updated := compiled.ReplaceAllString(content, redaction.Replacement)
	setPath(record, redaction.Target, updated)
	return &AppliedRedaction{
		Target:       redaction.Target,
		Match:        redaction.Match,
		Replacement:  redaction.Replacement,
		ReplaceCount: len(matches),
	}, nil
}

func selectFinalDecision(defaultAction string, decisions []Decision) FinalDecision {
	if len(decisions) == 0 {
		return FinalDecision{Action: normalizeFallbackAction(defaultAction)}
	}
	best := decisions[0]
	for _, decision := range decisions[1:] {
		if compareDecisionPriority(decision, best) < 0 {
			best = decision
		}
	}
	return FinalDecision{
		Action: best.Action,
		RuleID: best.RuleID,
		Rule:   best.RuleName,
		Reason: best.Reason,
	}
}

func normalizeFallbackAction(value string) string {
	action := normalizeAction(value)
	if !isSupportedAction(action) {
		return DefaultActionAllowed
	}
	return action
}

func compareDecisionPriority(left Decision, right Decision) int {
	leftRank := actionRank(left.Action)
	rightRank := actionRank(right.Action)
	if leftRank != rightRank {
		if leftRank > rightRank {
			return -1
		}
		return 1
	}
	if left.Priority != right.Priority {
		if left.Priority < right.Priority {
			return -1
		}
		return 1
	}
	if left.RuleID < right.RuleID {
		return -1
	}
	if left.RuleID > right.RuleID {
		return 1
	}
	return 0
}

func actionRank(action string) int {
	switch normalizeAction(action) {
	case ActionError:
		return 5
	case ActionBlocked:
		return 4
	case ActionRedacted:
		return 3
	case ActionRateLimited:
		return 2
	default:
		return 1
	}
}

func buildNormalizedRecord(record map[string]any, finalDecision FinalDecision, decisions []Decision, contentScanConfigured bool) map[string]any {
	normalized := make(map[string]any)
	for _, field := range normalizedCopyFields {
		copyPathIfPresent(normalized, record, field)
	}
	setPath(normalized, "policy.action", finalDecision.Action)
	if strings.TrimSpace(finalDecision.RuleID) != "" {
		setPath(normalized, "policy.rule", finalDecision.RuleID)
	}
	if strings.TrimSpace(finalDecision.Reason) != "" {
		setPath(normalized, "policy.reason", finalDecision.Reason)
	}
	if !contentScanConfigured {
		return normalized
	}
	setPath(normalized, "content_analysis.scan_performed", true)
	entities := collectEntities(decisions)
	if len(entities) > 0 {
		setPath(normalized, "content_analysis.pii_detected", true)
		setPath(normalized, "content_analysis.pii_entities", entities)
	}
	actionTaken := contentActionTaken(finalDecision.Action)
	if actionTaken != "" {
		setPath(normalized, "content_analysis.action_taken", actionTaken)
	}
	if actionTaken == "blocked" && strings.TrimSpace(finalDecision.Reason) != "" {
		setPath(normalized, "content_analysis.block_reason", finalDecision.Reason)
	}
	return normalized
}

func copyPathIfPresent(dst map[string]any, src map[string]any, path string) {
	if value, ok := lookupPath(src, path); ok {
		setPath(dst, path, deepCloneValue(value))
	}
}

func collectEntities(decisions []Decision) []string {
	entities := make([]string, 0)
	for _, decision := range decisions {
		entities = append(entities, decision.Entities...)
	}
	return dedupeSortedStrings(entities)
}

func contentActionTaken(action string) string {
	switch normalizeAction(action) {
	case ActionBlocked:
		return "blocked"
	case ActionRedacted:
		return "masked"
	case DefaultActionAllowed:
		return "scanned"
	default:
		return "scanned"
	}
}

func writeArtifacts(runDir string, rawDocument any, rulesDoc RulesFile, evaluated []EvaluatedRecord, decisions []Decision, normalized []map[string]any, issues []string, summary *Summary) error {
	if rawDocument == nil {
		rawDocument = map[string]any{}
	}
	issuesPayload := strings.Join(issues, "\n")
	if issuesPayload != "" {
		issuesPayload += "\n"
	}
	rulesYAML, err := yaml.Marshal(rulesDoc)
	if err != nil {
		return fmt.Errorf("marshal rule snapshot: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, RawInputJSONName), rawDocument); err != nil {
		return fmt.Errorf("write raw input json: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, EvaluatedJSONName), evaluated); err != nil {
		return fmt.Errorf("write evaluated records json: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, DecisionsJSONName), decisions); err != nil {
		return fmt.Errorf("write policy decisions json: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, NormalizedJSONName), normalized); err != nil {
		return fmt.Errorf("write normalized policy records json: %w", err)
	}
	if err := fsutil.AtomicWritePrivateFile(filepath.Join(runDir, ValidationIssuesName), []byte(issuesPayload)); err != nil {
		return fmt.Errorf("write policy validation issues: %w", err)
	}
	if err := fsutil.AtomicWritePrivateFile(filepath.Join(runDir, RulesSnapshotName), rulesYAML); err != nil {
		return fmt.Errorf("write rules snapshot yaml: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		return fmt.Errorf("write policy summary json: %w", err)
	}
	return artifactrun.WriteArtifacts(runDir, []artifactrun.Artifact{{
		Path: SummaryMarkdownName,
		Body: []byte(renderSummaryMarkdown(summary, issues)),
		Perm: fsutil.PrivateFilePerm,
	}})
}

func renderSummaryMarkdown(summary *Summary, issues []string) string {
	var builder strings.Builder
	builder.WriteString("# Policy Evaluation Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Rules path: `%s`\n", summary.RulesPath))
	if summary.InputPath != "" {
		builder.WriteString(fmt.Sprintf("- Input path: `%s`\n", summary.InputPath))
	} else {
		builder.WriteString("- Input path: `stdin`\n")
	}
	builder.WriteString(fmt.Sprintf("- Overall status: **%s**\n", summary.OverallStatus))
	builder.WriteString(fmt.Sprintf("- Record count: `%d`\n", summary.RecordCount))
	builder.WriteString(fmt.Sprintf("- Decision count: `%d`\n", summary.DecisionCount))
	builder.WriteString(fmt.Sprintf("- Validation issue count: `%d`\n", summary.ValidationIssueCount))
	builder.WriteString("\n## Final Actions\n\n")
	for _, action := range sortedIntKeys(summary.ActionCounts) {
		builder.WriteString(fmt.Sprintf("- `%s`: `%d`\n", action, summary.ActionCounts[action]))
	}
	builder.WriteString("\n## Triggered Rules\n\n")
	for _, ruleID := range sortedIntKeys(summary.RuleHitCounts) {
		builder.WriteString(fmt.Sprintf("- `%s`: `%d`\n", ruleID, summary.RuleHitCounts[ruleID]))
	}
	builder.WriteString("\n## Artifacts\n\n")
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.RawInputPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.EvaluatedPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.DecisionsPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.NormalizedPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.RulesSnapshotPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.IssuesPath))
	if len(issues) > 0 {
		builder.WriteString("\n## Validation Issues\n\n")
		for _, issue := range issues {
			builder.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}
	return builder.String()
}

func sortedIntKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneIntMap(values map[string]int) map[string]int {
	cloned := make(map[string]int, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func containsString(values []string, target string) bool {
	needle := strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == needle {
			return true
		}
	}
	return false
}

func equalValues(left any, right any) bool {
	if leftNumber, ok := asFloat64(left); ok {
		if rightNumber, ok := asFloat64(right); ok {
			return leftNumber == rightNumber
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%v", left)) == strings.TrimSpace(fmt.Sprintf("%v", right))
}

func numberValue(record map[string]any, path string) (float64, bool) {
	value, ok := lookupPath(record, path)
	if !ok {
		return 0, false
	}
	return asFloat64(value)
}

func asFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func lookupPath(root map[string]any, dottedPath string) (any, bool) {
	parts := strings.Split(strings.TrimSpace(dottedPath), ".")
	if len(parts) == 0 {
		return nil, false
	}
	var current any = root
	for _, part := range parts {
		mapping, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := mapping[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func setPath(root map[string]any, dottedPath string, value any) {
	parts := strings.Split(strings.TrimSpace(dottedPath), ".")
	if len(parts) == 0 {
		return
	}
	cursor := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := cursor[part].(map[string]any)
		if !ok {
			next = make(map[string]any)
			cursor[part] = next
		}
		cursor = next
	}
	cursor[parts[len(parts)-1]] = value
}

func deepCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = deepCloneValue(value)
	}
	return cloned
}

func deepCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return deepCloneMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = deepCloneValue(item)
		}
		return cloned
	default:
		return typed
	}
}

func isBlankValue(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}
