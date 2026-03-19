// Package ingest provides typed vendor-evidence ingest and normalization.
//
// Purpose:
//   - Execute local vendor evidence ingest runs that normalize supported vendor
//     exports into the tracked ACP evidence contract.
//
// Responsibilities:
//   - Parse supported vendor export payloads from JSON file/stdin content.
//   - Normalize records to the tracked schema shape and validate them.
//   - Persist auditable run artifacts via internal/artifactrun.
//
// Scope:
//   - Local ingest/normalization workflow only.
//
// Usage:
//   - Used by `acpctl evidence ingest`.
//
// Invariants/Assumptions:
//   - Ingest remains file/stdin based for the supported host-first surface.
//   - Normalized outputs are private local artifacts under demo/logs/evidence.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
)

var nowUTC = func() time.Time { return time.Now().UTC() }

// Ingest executes one local vendor evidence ingest run.
func Ingest(_ context.Context, opts Options) (*Result, error) {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if len(opts.InputPayload) == 0 {
		return nil, fmt.Errorf("input payload is required")
	}
	if opts.OutputRoot == "" {
		opts.OutputRoot = repopath.DemoLogsPath(opts.RepoRoot, "evidence", DefaultOutputSubdir)
	}
	if opts.Format == "" {
		opts.Format = FormatComplianceAPI
	}
	if opts.SourceName == "" {
		opts.SourceName = DefaultSourceService
	}

	schema, err := LoadSchema(repopath.DemoConfigPath(opts.RepoRoot, "normalized_schema.yaml"))
	if err != nil {
		return nil, err
	}
	records, rawDocument, err := normalizeInput(opts)
	if err != nil {
		return nil, err
	}

	issues := validationissues.NewIssues()
	for index, record := range records {
		issues.Extend(schema.ValidateRecord(record, index))
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
		Format:               string(opts.Format),
		SourceType:           SchemaSourceTypeValue,
		SourceName:           opts.SourceName,
		InputPath:            opts.InputPath,
		OverallStatus:        "PASS",
		RecordCount:          len(records),
		ValidationIssueCount: len(issueList),
		NormalizedPath:       filepath.Join(run.Directory, NormalizedJSONName),
		RawInputPath:         filepath.Join(run.Directory, RawInputJSONName),
		IssuesPath:           filepath.Join(run.Directory, ValidationIssuesName),
	}
	if len(issueList) > 0 {
		summary.OverallStatus = "FAIL"
	}
	if err := writeArtifacts(run.Directory, rawDocument, records, issueList, summary); err != nil {
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
	if err := writeArtifacts(run.Directory, rawDocument, records, issueList, summary); err != nil {
		return nil, err
	}
	return &Result{Summary: summary, Records: records, Issues: issueList}, nil
}

func normalizeInput(opts Options) ([]map[string]any, any, error) {
	rawRecords, rawDocument, err := parseComplianceExports(opts.InputPayload)
	if err != nil {
		return nil, nil, err
	}
	normalized := make([]map[string]any, 0, len(rawRecords))
	for _, record := range rawRecords {
		normalized = append(normalized, normalizeComplianceExport(record, opts.SourceName))
	}
	return normalized, rawDocument, nil
}

func parseComplianceExports(data []byte) ([]ComplianceExport, any, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parse input json: %w", err)
	}
	switch typed := raw.(type) {
	case []any:
		records := make([]ComplianceExport, 0, len(typed))
		for _, item := range typed {
			record, err := decodeComplianceRecord(item)
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
			records := make([]ComplianceExport, 0, len(items))
			for _, item := range items {
				record, err := decodeComplianceRecord(item)
				if err != nil {
					return nil, nil, err
				}
				records = append(records, record)
			}
			return records, raw, nil
		}
		record, err := decodeComplianceRecord(typed)
		if err != nil {
			return nil, nil, err
		}
		return []ComplianceExport{record}, raw, nil
	default:
		return nil, nil, fmt.Errorf("input must be a JSON object or array")
	}
}

func decodeComplianceRecord(value any) (ComplianceExport, error) {
	document, ok := value.(map[string]any)
	if !ok {
		return ComplianceExport{}, fmt.Errorf("record must be a JSON object")
	}
	if wrapped, ok := document["compliance_export"]; ok {
		document, ok = wrapped.(map[string]any)
		if !ok {
			return ComplianceExport{}, fmt.Errorf("compliance_export must be a JSON object")
		}
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return ComplianceExport{}, fmt.Errorf("marshal compliance record: %w", err)
	}
	var record ComplianceExport
	if err := json.Unmarshal(payload, &record); err != nil {
		return ComplianceExport{}, fmt.Errorf("decode compliance record: %w", err)
	}
	return record, nil
}

func normalizeComplianceExport(record ComplianceExport, sourceName string) map[string]any {
	normalized := make(map[string]any)
	setPath(normalized, "principal.id", strings.TrimSpace(record.User.Email))
	setPath(normalized, "principal.type", "user")
	if email := strings.TrimSpace(record.User.Email); email != "" {
		setPath(normalized, "principal.email", email)
	}
	setPath(normalized, "ai.model.id", strings.TrimSpace(record.Model))
	setPath(normalized, "ai.provider", strings.TrimSpace(record.Provider))
	setPath(normalized, "ai.request.id", strings.TrimSpace(record.RequestID))
	setPath(normalized, "ai.request.timestamp", strings.TrimSpace(record.CreatedAt))
	setPath(normalized, "ai.tokens.prompt", record.Usage.PromptTokens)
	setPath(normalized, "ai.tokens.completion", record.Usage.CompletionTokens)
	setPath(normalized, "ai.tokens.total", record.Usage.PromptTokens+record.Usage.CompletionTokens)
	setPath(normalized, "ai.cost.amount", record.Usage.TotalCost)
	setPath(normalized, "ai.cost.currency", "USD")
	setPath(normalized, "policy.action", normalizePolicyAction(record.PolicyAction))
	if sessionID := strings.TrimSpace(record.SessionID); sessionID != "" {
		setPath(normalized, "correlation.session.id", sessionID)
	}
	setPath(normalized, "source.type", SchemaSourceTypeValue)
	setPath(normalized, "source.service.name", strings.TrimSpace(sourceName))
	return normalized
}

func normalizePolicyAction(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "allow", "allowed", "success":
		return "allowed"
	case "block", "blocked", "deny", "denied":
		return "blocked"
	case "rate-limit", "rate_limited", "rate-limited":
		return "rate_limited"
	case "redact", "redacted":
		return "redacted"
	case "error", "failure", "failed":
		return "error"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func writeArtifacts(runDir string, rawDocument any, records []map[string]any, issues []string, summary *Summary) error {
	var rawInput any = rawDocument
	if rawInput == nil {
		rawInput = map[string]any{}
	}
	issuesPayload := strings.Join(issues, "\n")
	if issuesPayload != "" {
		issuesPayload += "\n"
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, RawInputJSONName), rawInput); err != nil {
		return fmt.Errorf("write raw input json: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, NormalizedJSONName), records); err != nil {
		return fmt.Errorf("write normalized records json: %w", err)
	}
	if err := fsutil.AtomicWritePrivateFile(filepath.Join(runDir, ValidationIssuesName), []byte(issuesPayload)); err != nil {
		return fmt.Errorf("write validation issues: %w", err)
	}
	if err := artifactrun.WriteJSON(filepath.Join(runDir, SummaryJSONName), summary); err != nil {
		return fmt.Errorf("write ingest summary json: %w", err)
	}
	return artifactrun.WriteArtifacts(runDir, []artifactrun.Artifact{{
		Path: SummaryMarkdownName,
		Body: []byte(renderSummaryMarkdown(summary, issues)),
		Perm: fsutil.PrivateFilePerm,
	}})
}

func renderSummaryMarkdown(summary *Summary, issues []string) string {
	var builder strings.Builder
	builder.WriteString("# Vendor Evidence Ingest Summary\n\n")
	builder.WriteString(fmt.Sprintf("- Run ID: `%s`\n", summary.RunID))
	builder.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAtUTC))
	builder.WriteString(fmt.Sprintf("- Format: `%s`\n", summary.Format))
	builder.WriteString(fmt.Sprintf("- Source type: `%s`\n", summary.SourceType))
	builder.WriteString(fmt.Sprintf("- Source name: `%s`\n", summary.SourceName))
	if summary.InputPath != "" {
		builder.WriteString(fmt.Sprintf("- Input path: `%s`\n", summary.InputPath))
	} else {
		builder.WriteString("- Input path: `stdin`\n")
	}
	builder.WriteString(fmt.Sprintf("- Overall status: **%s**\n", summary.OverallStatus))
	builder.WriteString(fmt.Sprintf("- Record count: `%d`\n", summary.RecordCount))
	builder.WriteString(fmt.Sprintf("- Validation issue count: `%d`\n", summary.ValidationIssueCount))
	builder.WriteString("\n## Artifacts\n\n")
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.RawInputPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.NormalizedPath))
	builder.WriteString(fmt.Sprintf("- `%s`\n", summary.IssuesPath))
	if len(issues) > 0 {
		builder.WriteString("\n## Validation Issues\n\n")
		for _, issue := range issues {
			builder.WriteString(fmt.Sprintf("- %s\n", issue))
		}
	}
	return builder.String()
}
