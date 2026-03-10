// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Audit tracked files for likely secret leakage.
//
// Responsibilities:
//   - Flag suspicious tracked secret-bearing paths.
//   - Scan textual content for high-confidence secret signatures.
//   - Respect approved placeholder examples in docs and tests.
//
// Scope:
//   - Deterministic tracked-file secret auditing only.
//
// Usage:
//   - Called by `acpctl validate secrets-audit`.
//
// Invariants/Assumptions:
//   - Binary and non-UTF-8 content are skipped.
//   - Findings are sorted by path, line, then rule ID.
package security

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

const secretsPolicyPath = "docs/policy/SECRET_SCAN_POLICY.json"

func AuditTrackedSecrets(repoRoot string, trackedFiles []string) ([]Finding, error) {
	policyDoc, err := loadSecretsPolicy(repoRoot)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0)
	for _, relPath := range trackedFiles {
		findings = append(findings, checkPathRules(relPath, policyDoc.PathRules)...)
		data, err := os.ReadFile(filepath.Join(repoRoot, relPath))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !shouldScanContent(relPath, data) {
			continue
		}
		contentFindings, err := checkContentRules(relPath, data, policyDoc.ContentRules, policyDoc.PlaceholderExemptions)
		if err != nil {
			return nil, err
		}
		findings = append(findings, contentFindings...)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].RuleID < findings[j].RuleID
	})
	return findings, nil
}

type compiledSecretsPolicy struct {
	PathRules             []compiledSecretPathRule
	ContentRules          []compiledContentRule
	PlaceholderExemptions []compiledPlaceholderExemption
}

type compiledSecretPathRule struct {
	ID       string
	Message  string
	Patterns []string
}

type compiledContentRule struct {
	ID      string
	Message string
	Pattern *regexp.Regexp
}

type compiledPlaceholderExemption struct {
	ID                   string
	PathPatterns         []string
	AllowedSubstrings    []string
	AllowEmptyAssignment bool
}

func loadSecretsPolicy(repoRoot string) (*compiledSecretsPolicy, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, secretsPolicyPath))
	if err != nil {
		return nil, err
	}
	var doc SecretsPolicy
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: %w", secretsPolicyPath, err)
	}
	if err := validateSecretsPolicy(doc); err != nil {
		return nil, fmt.Errorf("%s: %w", secretsPolicyPath, err)
	}
	contentRules, err := compileContentRules(doc.ContentRules)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", secretsPolicyPath, err)
	}
	return &compiledSecretsPolicy{
		PathRules:             compilePathRules(doc.PathRules),
		ContentRules:          contentRules,
		PlaceholderExemptions: compilePlaceholderExemptions(doc.PlaceholderExemptions),
	}, nil
}

func validateSecretsPolicy(doc SecretsPolicy) error {
	if textutil.IsBlank(doc.SchemaVersion) {
		return fmt.Errorf("missing schema_version")
	}
	if textutil.IsBlank(doc.PolicyID) {
		return fmt.Errorf("missing policy_id")
	}
	if len(doc.PathRules) == 0 && len(doc.ContentRules) == 0 {
		return fmt.Errorf("at least one path rule or content rule is required")
	}
	seenIDs := make(map[string]string)
	for _, rule := range doc.PathRules {
		if err := validateSecretsPolicyID(seenIDs, rule.ID, "path rule"); err != nil {
			return err
		}
		if textutil.IsBlank(rule.Message) {
			return fmt.Errorf("path rule %q missing message", rule.ID)
		}
		if len(trimNonEmptyStrings(rule.Patterns)) == 0 {
			return fmt.Errorf("path rule %q must declare at least one pattern", rule.ID)
		}
	}
	for _, rule := range doc.ContentRules {
		if err := validateSecretsPolicyID(seenIDs, rule.ID, "content rule"); err != nil {
			return err
		}
		if textutil.IsBlank(rule.Message) {
			return fmt.Errorf("content rule %q missing message", rule.ID)
		}
		if textutil.IsBlank(rule.Pattern) {
			return fmt.Errorf("content rule %q missing pattern", rule.ID)
		}
	}
	for _, exemption := range doc.PlaceholderExemptions {
		if err := validateSecretsPolicyID(seenIDs, exemption.ID, "placeholder exemption"); err != nil {
			return err
		}
		if len(trimNonEmptyStrings(exemption.PathPatterns)) == 0 {
			return fmt.Errorf("placeholder exemption %q must declare at least one path pattern", exemption.ID)
		}
		if len(trimNonEmptyStrings(exemption.AllowedSubstrings)) == 0 && !exemption.AllowEmptyAssignment {
			return fmt.Errorf("placeholder exemption %q must allow at least one placeholder substring or empty assignment", exemption.ID)
		}
	}
	return nil
}

func validateSecretsPolicyID(seenIDs map[string]string, id string, kind string) error {
	trimmedID := textutil.Trim(id)
	if trimmedID == "" {
		return fmt.Errorf("%s missing id", kind)
	}
	if previousKind, exists := seenIDs[trimmedID]; exists {
		return fmt.Errorf("duplicate policy id %q used by %s and %s", trimmedID, previousKind, kind)
	}
	seenIDs[trimmedID] = kind
	return nil
}

func compilePathRules(rules []SecretPathRule) []compiledSecretPathRule {
	compiled := make([]compiledSecretPathRule, 0, len(rules))
	for _, rule := range rules {
		compiled = append(compiled, compiledSecretPathRule{
			ID:       rule.ID,
			Message:  rule.Message,
			Patterns: trimNonEmptyStrings(rule.Patterns),
		})
	}
	return compiled
}

func compileContentRules(rules []SecretContentRule) ([]compiledContentRule, error) {
	compiled := make([]compiledContentRule, 0, len(rules))
	for _, rule := range rules {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, compiledContentRule{
			ID:      rule.ID,
			Message: rule.Message,
			Pattern: pattern,
		})
	}
	return compiled, nil
}

func compilePlaceholderExemptions(exemptions []SecretPlaceholderExemption) []compiledPlaceholderExemption {
	compiled := make([]compiledPlaceholderExemption, 0, len(exemptions))
	for _, exemption := range exemptions {
		allowedSubstrings := trimNonEmptyStrings(exemption.AllowedSubstrings)
		for index := range allowedSubstrings {
			allowedSubstrings[index] = textutil.LowerTrim(allowedSubstrings[index])
		}
		compiled = append(compiled, compiledPlaceholderExemption{
			ID:                   exemption.ID,
			PathPatterns:         trimNonEmptyStrings(exemption.PathPatterns),
			AllowedSubstrings:    allowedSubstrings,
			AllowEmptyAssignment: exemption.AllowEmptyAssignment,
		})
	}
	return compiled
}

func trimNonEmptyStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		candidate := textutil.Trim(value)
		if candidate == "" {
			continue
		}
		trimmed = append(trimmed, candidate)
	}
	return trimmed
}

func checkPathRules(relPath string, rules []compiledSecretPathRule) []Finding {
	findings := make([]Finding, 0)
	for _, rule := range rules {
		if policy.MatchAnyGlob(relPath, rule.Patterns) {
			findings = append(findings, Finding{Path: relPath, RuleID: rule.ID, Message: rule.Message})
		}
	}
	return findings
}

func shouldScanContent(relPath string, data []byte) bool {
	if len(data) == 0 || !utf8.Valid(data) {
		return false
	}
	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".tar", ".gz", ".tgz":
		return false
	default:
		return true
	}
}

func checkContentRules(relPath string, data []byte, rules []compiledContentRule, exemptions []compiledPlaceholderExemption) ([]Finding, error) {
	findings := make([]Finding, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		for _, rule := range rules {
			if !rule.Pattern.MatchString(line) {
				continue
			}
			if isAllowedPlaceholder(relPath, line, exemptions) {
				continue
			}
			findings = append(findings, Finding{Path: relPath, Line: lineNumber, RuleID: rule.ID, Message: rule.Message})
		}
	}
	return findings, scanner.Err()
}

func isAllowedPlaceholder(relPath string, line string, exemptions []compiledPlaceholderExemption) bool {
	lowerLine := strings.ToLower(line)
	for _, exemption := range exemptions {
		if !policy.MatchAnyGlob(relPath, exemption.PathPatterns) {
			continue
		}
		if exemption.AllowEmptyAssignment && strings.HasSuffix(textutil.Trim(line), "=") {
			return true
		}
		for _, allowedSubstring := range exemption.AllowedSubstrings {
			if strings.Contains(lowerLine, allowedSubstring) {
				return true
			}
		}
	}
	return false
}
