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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var secretPathRules = []pathRule{
	{
		ID:      "tracked-env-file",
		Message: "tracked environment file",
		Match: func(relPath string) bool {
			return filepath.Base(relPath) == ".env"
		},
	},
	{
		ID:      "private-key-file",
		Message: "suspicious private-key filename",
		Match: func(relPath string) bool {
			base := filepath.Base(relPath)
			return base == "id_rsa" || base == "id_ed25519"
		},
	},
	{
		ID:      "secret-bearing-file",
		Message: "suspicious certificate/key archive filename",
		Match: func(relPath string) bool {
			switch strings.ToLower(filepath.Ext(relPath)) {
			case ".pem", ".p12", ".pfx":
				return true
			default:
				return false
			}
		},
	},
}

var secretContentRules = []contentRule{
	{ID: "private-key-block", Message: "private key material", Pattern: `-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`},
	{ID: "aws-access-key-id", Message: "AWS access key ID", Pattern: `\bAKIA[0-9A-Z]{16}\b`},
	{ID: "github-token", Message: "GitHub token", Pattern: `\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`},
	{ID: "slack-token", Message: "Slack token", Pattern: `\bxox[baprs]-[A-Za-z0-9-]{20,}\b`},
	{ID: "google-api-key", Message: "Google API key", Pattern: `\bAIza[0-9A-Za-z_-]{20,}\b`},
	{ID: "openai-style-key", Message: "OpenAI-style API key", Pattern: `\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\b`},
}

func AuditTrackedSecrets(repoRoot string, trackedFiles []string) ([]Finding, error) {
	compiledRules, err := compileContentRules(secretContentRules)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0)
	for _, relPath := range trackedFiles {
		findings = append(findings, checkPathRules(relPath)...)
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
		contentFindings, err := checkContentRules(relPath, data, compiledRules)
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

func compileContentRules(rules []contentRule) ([]compiledContentRule, error) {
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

type compiledContentRule struct {
	ID      string
	Message string
	Pattern *regexp.Regexp
}

func checkPathRules(relPath string) []Finding {
	findings := make([]Finding, 0)
	for _, rule := range secretPathRules {
		if rule.Match(relPath) {
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

func checkContentRules(relPath string, data []byte, rules []compiledContentRule) ([]Finding, error) {
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
			if isAllowedPlaceholder(relPath, line) {
				continue
			}
			findings = append(findings, Finding{Path: relPath, Line: lineNumber, RuleID: rule.ID, Message: rule.Message})
		}
	}
	return findings, scanner.Err()
}

func isAllowedPlaceholder(relPath string, line string) bool {
	lowerLine := strings.ToLower(line)
	switch {
	case relPath == "demo/.env.example":
		return strings.Contains(lowerLine, "change-me") || strings.HasSuffix(strings.TrimSpace(line), "=")
	case strings.HasSuffix(relPath, "_test.go"), strings.Contains(relPath, "/tests/"):
		return strings.Contains(lowerLine, "sk-test-") || strings.Contains(lowerLine, "change-me") || strings.Contains(lowerLine, "sk-litellm-")
	case strings.HasPrefix(relPath, "deploy/helm/ai-control-plane/examples/"):
		return strings.Contains(lowerLine, "sk-demo-") || strings.Contains(lowerLine, "sk-offline-demo-")
	case relPath == "README.md", relPath == "demo/README.md", strings.HasPrefix(relPath, "docs/"):
		return strings.Contains(lowerLine, "change-me") ||
			strings.Contains(lowerLine, "sk-demo-") ||
			strings.Contains(lowerLine, "sk-offline-demo-") ||
			strings.Contains(lowerLine, "sk-your-") ||
			strings.Contains(lowerLine, "sk-personal-") ||
			strings.Contains(lowerLine, "sk-litellm-")
	default:
		return false
	}
}
