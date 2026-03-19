// Package ingest provides typed vendor-evidence ingest and normalization.
//
// Purpose:
//   - Define the typed inputs, outputs, and supported vendor formats for local
//     evidence ingestion workflows.
//
// Responsibilities:
//   - Describe ingest options and generated run summaries.
//   - Define the supported vendor export payload for normalization.
//   - Keep shared ingest types out of the CLI layer.
//
// Scope:
//   - Shared ingest domain types only.
//
// Usage:
//   - Used by `internal/ingest` workflows and `acpctl evidence ingest`.
//
// Invariants/Assumptions:
//   - Supported ingest is local/file/stdin based, not a long-running network service.
package ingest

const (
	SummaryJSONName       = "summary.json"
	SummaryMarkdownName   = "vendor-evidence-summary.md"
	NormalizedJSONName    = "normalized-records.json"
	RawInputJSONName      = "raw-input.json"
	ValidationIssuesName  = "validation-issues.txt"
	InventoryFileName     = "ingest-inventory.txt"
	LatestRunPointerName  = "latest-run.txt"
	DefaultSourceService  = "vendor-compliance-export"
	DefaultOutputSubdir   = "vendor-ingest"
	SchemaSourceTypeValue = "compliance_api"
)

// Format identifies the supported vendor evidence input formats.
type Format string

const (
	// FormatComplianceAPI ingests compliance export records aligned with the
	// normalized_schema.yaml compliance_api_to_schema mapping.
	FormatComplianceAPI Format = "compliance-api"
)

// Options configures one vendor ingest run.
type Options struct {
	RepoRoot     string
	OutputRoot   string
	InputPath    string
	SourceName   string
	Format       Format
	InputPayload []byte
}

// Summary captures the machine-readable result of one ingest run.
type Summary struct {
	RunID                string   `json:"run_id"`
	GeneratedAtUTC       string   `json:"generated_at_utc"`
	RepoRoot             string   `json:"repo_root"`
	RunDirectory         string   `json:"run_directory"`
	Format               string   `json:"format"`
	SourceType           string   `json:"source_type"`
	SourceName           string   `json:"source_name"`
	InputPath            string   `json:"input_path,omitempty"`
	OverallStatus        string   `json:"overall_status"`
	RecordCount          int      `json:"record_count"`
	ValidationIssueCount int      `json:"validation_issue_count"`
	NormalizedPath       string   `json:"normalized_path"`
	RawInputPath         string   `json:"raw_input_path"`
	IssuesPath           string   `json:"issues_path"`
	GeneratedFiles       []string `json:"generated_files"`
}

// Result returns the generated summary plus normalized records and issues.
type Result struct {
	Summary *Summary
	Records []map[string]any
	Issues  []string
}

// ComplianceExport captures the supported vendor export payload.
type ComplianceExport struct {
	RequestID    string `json:"request_id"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	CreatedAt    string `json:"created_at"`
	PolicyAction string `json:"policy_action"`
	SessionID    string `json:"session_id,omitempty"`
	User         struct {
		Email string `json:"email"`
	} `json:"user"`
	Usage struct {
		PromptTokens     int     `json:"prompt_tokens"`
		CompletionTokens int     `json:"completion_tokens"`
		TotalCost        float64 `json:"total_cost"`
	} `json:"usage"`
}
