// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Define shared security validation types and policy document shapes.
//
// Responsibilities:
//   - Model secret-audit findings.
//   - Decode typed security policy documents.
//   - Keep shared security types separate from validator implementations.
//
// Scope:
//   - Shared type definitions for repository security validation only.
//
// Usage:
//   - Consumed by tracked-file, license, and supply-chain validators.
//
// Invariants/Assumptions:
//   - Policy structs map directly to repository JSON documents.
//   - Findings stay stable for deterministic reporting.
package security

type Finding struct {
	Path    string
	Line    int
	RuleID  string
	Message string
}

type SecretsPolicy struct {
	SchemaVersion         string                       `json:"schema_version"`
	PolicyID              string                       `json:"policy_id"`
	Description           string                       `json:"description"`
	PathRules             []SecretPathRule             `json:"path_rules"`
	ContentRules          []SecretContentRule          `json:"content_rules"`
	PlaceholderExemptions []SecretPlaceholderExemption `json:"placeholder_exemptions"`
}

type SecretPathRule struct {
	ID       string   `json:"id"`
	Message  string   `json:"message"`
	Patterns []string `json:"patterns"`
}

type SecretContentRule struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	Pattern string `json:"pattern"`
}

type SecretPlaceholderExemption struct {
	ID                   string   `json:"id"`
	PathPatterns         []string `json:"path_patterns"`
	AllowedSubstrings    []string `json:"allowed_substrings"`
	AllowEmptyAssignment bool     `json:"allow_empty_assignment,omitempty"`
}

type SupplyChainPolicy struct {
	SchemaVersion  any                   `json:"schema_version"`
	PolicyID       string                `json:"policy_id"`
	Allowlist      []map[string]any      `json:"allowlist"`
	SeverityPolicy supplyChainSeverity   `json:"severity_policy"`
	Provenance     supplyChainProvenance `json:"provenance"`
	ScannerPolicy  map[string]any        `json:"scanner_policy"`
}

type supplyChainSeverity struct {
	FailOn    []string       `json:"fail_on"`
	MaxCounts map[string]int `json:"max_counts"`
}

type supplyChainProvenance struct {
	RequireDigestPin bool `json:"require_digest_pin"`
}

type LicensePolicy struct {
	SchemaVersion        any                          `json:"schema_version"`
	PolicyID             any                          `json:"policy_id"`
	ScanScope            licenseScanScope             `json:"scan_scope"`
	RestrictedComponents []licenseRestrictedComponent `json:"restricted_components"`
}

type licenseScanScope struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type licenseRestrictedComponent struct {
	Name  string                 `json:"name"`
	Match licenseRestrictedMatch `json:"match"`
}

type licenseRestrictedMatch struct {
	PathRegex    []string `json:"path_regex"`
	ContentRegex []string `json:"content_regex"`
}
