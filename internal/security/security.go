// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Move security policy enforcement out of CLI command files.
//
// Responsibilities:
//   - Audit tracked files for likely secret leakage.
//   - Enforce public-release tracked-file hygiene.
//   - Validate license-boundary and supply-chain policy contracts.
//
// Scope:
//   - Read-only repository security validation only.
//
// Usage:
//   - Called by `acpctl validate` command adapters and CI gates.
//
// Invariants/Assumptions:
//   - Findings are deterministic and stably sorted.
//   - Deployment-surface scope comes from `internal/policy`, not command code.
package security

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
	"gopkg.in/yaml.v3"
)

type Finding struct {
	Path    string
	Line    int
	RuleID  string
	Message string
}

type pathRule struct {
	ID      string
	Message string
	Match   func(relPath string) bool
}

type contentRule struct {
	ID      string
	Message string
	Pattern *regexp.Regexp
}

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
	{ID: "private-key-block", Message: "private key material", Pattern: regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)},
	{ID: "aws-access-key-id", Message: "AWS access key ID", Pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{ID: "github-token", Message: "GitHub token", Pattern: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`)},
	{ID: "slack-token", Message: "Slack token", Pattern: regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{20,}\b`)},
	{ID: "google-api-key", Message: "Google API key", Pattern: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{20,}\b`)},
	{ID: "openai-style-key", Message: "OpenAI-style API key", Pattern: regexp.MustCompile(`\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\b`)},
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

func ListTrackedFiles(ctx context.Context, repoRoot string) ([]string, error) {
	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    []string{"ls-files", "-z"},
		Dir:     repoRoot,
		Timeout: 5_000_000_000,
	})
	if res.Err != nil {
		return nil, res.Err
	}
	rawPaths := bytes.Split([]byte(res.Stdout), []byte{0})
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		if len(rawPath) == 0 {
			continue
		}
		paths = append(paths, filepath.Clean(string(rawPath)))
	}
	sort.Strings(paths)
	return paths, nil
}

func AuditTrackedSecrets(repoRoot string, trackedFiles []string) ([]Finding, error) {
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
		contentFindings, err := checkContentRules(relPath, data)
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

func ValidatePublicHygiene(trackedFiles []string) []string {
	var violations []string
	for _, relPath := range trackedFiles {
		if !IsLocalOnlyTrackedPath(relPath) {
			continue
		}
		if strings.HasSuffix(relPath, "/.gitkeep") || strings.HasSuffix(relPath, "/.gitignore") {
			continue
		}
		violations = append(violations, relPath)
	}
	sort.Strings(violations)
	return violations
}

func ValidateLicensePolicy(repoRoot string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "policy", "THIRD_PARTY_LICENSE_MATRIX.json"))
	if err != nil {
		return nil, err
	}
	var policyDoc LicensePolicy
	if err := json.Unmarshal(data, &policyDoc); err != nil {
		return nil, err
	}
	if policyDoc.SchemaVersion == nil || policyDoc.PolicyID == nil || len(policyDoc.ScanScope.Include) == 0 || len(policyDoc.RestrictedComponents) == 0 {
		return []string{"docs/policy/THIRD_PARTY_LICENSE_MATRIX.json: missing required policy fields"}, nil
	}
	return findRestrictedLicenseReferences(repoRoot, policyDoc)
}

func ValidateSupplyChainPolicy(repoRoot string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json"))
	if err != nil {
		return nil, err
	}
	var policyDoc SupplyChainPolicy
	if err := json.Unmarshal(data, &policyDoc); err != nil {
		return nil, err
	}
	issues := make([]string, 0)
	if policyDoc.PolicyID == "" || len(policyDoc.SeverityPolicy.FailOn) == 0 {
		issues = append(issues, "demo/config/supply_chain_vulnerability_policy.json: missing required policy fields")
	}
	targets, err := policy.ExpandDeploymentSurfaces(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if !hasRule(target.Rules, policy.RuleImagePinning) {
			continue
		}
		targetIssues, err := validatePinnedImagesForTarget(repoRoot, target)
		if err != nil {
			return nil, err
		}
		issues = append(issues, targetIssues...)
	}
	sort.Strings(issues)
	return issues, nil
}

func hasRule(rules []policy.SurfaceRule, target policy.SurfaceRule) bool {
	for _, rule := range rules {
		if rule == target {
			return true
		}
	}
	return false
}

func validatePinnedImagesForTarget(repoRoot string, target policy.SurfaceTarget) ([]string, error) {
	absPath := filepath.Join(repoRoot, filepath.FromSlash(target.Path))
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	switch target.Kind {
	case policy.SurfaceCompose:
		return validateComposeImages(absPath, target.Path)
	case policy.SurfaceHelmValues:
		return validateHelmImages(absPath, target.Path)
	case policy.SurfaceDockerfile:
		return validateDockerfileBaseImages(absPath, target.Path)
	default:
		return nil, nil
	}
}

func validateComposeImages(path string, relPath string) ([]string, error) {
	root, err := loadYAMLNode(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", relPath, err)}, nil
	}
	servicesNode := mapValue(root, "services")
	if servicesNode == nil || servicesNode.Kind != yaml.MappingNode {
		return []string{fmt.Sprintf("%s: missing services mapping", relPath)}, nil
	}
	issues := make([]string, 0)
	for i := 0; i < len(servicesNode.Content); i += 2 {
		serviceName := servicesNode.Content[i].Value
		serviceNode := servicesNode.Content[i+1]
		imageNode := mapValue(serviceNode, "image")
		if imageNode == nil {
			continue
		}
		image := strings.TrimSpace(imageNode.Value)
		if isDigestPinnedImage(image) {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s: service %q image must be digest pinned (got %q)", relPath, serviceName, image))
	}
	return issues, nil
}

func validateHelmImages(path string, relPath string) ([]string, error) {
	root, err := loadYAMLNode(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", relPath, err)}, nil
	}
	issues := make([]string, 0)
	visitMappings(root, "", func(node *yaml.Node, currentPath string) {
		repository := mapValue(node, "repository")
		if repository == nil || strings.TrimSpace(repository.Value) == "" {
			return
		}
		digest := mapValue(node, "digest")
		if digest == nil || strings.TrimSpace(digest.Value) == "" {
			issues = append(issues, fmt.Sprintf("%s: %s must declare a non-empty image digest for repository %q", relPath, currentPath, strings.TrimSpace(repository.Value)))
		}
	})
	return issues, nil
}

func validateDockerfileBaseImages(path string, relPath string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	issues := make([]string, 0)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 || isDigestPinnedImage(fields[1]) {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s:%d: base image must be digest pinned (got %q)", relPath, lineNumber, fields[1]))
	}
	return issues, scanner.Err()
}

func isDigestPinnedImage(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "@sha256:") {
		return true
	}
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "${"), "}")
		parts := strings.SplitN(inner, ":-", 2)
		if len(parts) == 2 {
			return strings.Contains(parts[1], "@sha256:")
		}
	}
	return false
}

func loadYAMLNode(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}
	return root.Content[0], nil
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func visitMappings(node *yaml.Node, currentPath string, fn func(node *yaml.Node, currentPath string)) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		fn(node, currentPath)
		for i := 0; i < len(node.Content); i += 2 {
			childKey := node.Content[i].Value
			childPath := childKey
			if currentPath != "" {
				childPath = currentPath + "." + childKey
			}
			visitMappings(node.Content[i+1], childPath, fn)
		}
		return
	}
	for _, child := range node.Content {
		visitMappings(child, currentPath, fn)
	}
}

func IsLocalOnlyTrackedPath(relPath string) bool {
	switch {
	case relPath == ".env":
		return true
	case relPath == "demo/.env":
		return true
	case strings.HasPrefix(relPath, "demo/") && strings.HasSuffix(relPath, "/.env"):
		return true
	case strings.HasPrefix(relPath, "demo/logs/"):
		return true
	case strings.HasPrefix(relPath, "demo/backups/"):
		return true
	case strings.HasPrefix(relPath, "handoff-packet/"):
		return true
	case strings.HasPrefix(relPath, ".ralph/"):
		return true
	case strings.HasPrefix(relPath, "docs/presentation/slides-internal/"):
		return true
	case strings.HasPrefix(relPath, "docs/presentation/slides-external/") && strings.HasSuffix(relPath, ".png"):
		return true
	case relPath == ".scratchpad.md":
		return true
	default:
		return false
	}
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

func checkContentRules(relPath string, data []byte) ([]Finding, error) {
	findings := make([]Finding, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		for _, rule := range secretContentRules {
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

func findRestrictedLicenseReferences(repoRoot string, policyDoc LicensePolicy) ([]string, error) {
	pathMatchers, err := compileRegexps(flattenLicensePathRegexps(policyDoc.RestrictedComponents))
	if err != nil {
		return nil, err
	}
	contentMatchers, err := compileRegexps(flattenLicenseContentRegexps(policyDoc.RestrictedComponents))
	if err != nil {
		return nil, err
	}
	findings := make([]string, 0)
	err = filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if d.IsDir() {
			if relPath == ".git" || matchesAnyGlob(relPath, policyDoc.ScanScope.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesAnyGlob(relPath, policyDoc.ScanScope.Include) || matchesAnyGlob(relPath, policyDoc.ScanScope.Exclude) {
			return nil
		}
		if matchesAnyRegexp(relPath, pathMatchers) {
			findings = append(findings, relPath)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if matchesAnyRegexp(string(data), contentMatchers) {
			findings = append(findings, relPath)
		}
		return nil
	})
	sort.Strings(findings)
	return findings, err
}

func flattenLicensePathRegexps(components []licenseRestrictedComponent) []string {
	patterns := make([]string, 0)
	for _, component := range components {
		patterns = append(patterns, component.Match.PathRegex...)
	}
	return patterns
}

func flattenLicenseContentRegexps(components []licenseRestrictedComponent) []string {
	patterns := make([]string, 0)
	for _, component := range components {
		patterns = append(patterns, component.Match.ContentRegex...)
	}
	return patterns
}

func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func matchesAnyRegexp(value string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func matchesAnyGlob(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if globMatch(relPath, pattern) {
			return true
		}
	}
	return false
}

func globMatch(relPath string, pattern string) bool {
	const (
		doubleStarDirToken = "<<double-star-dir>>"
		doubleStarToken    = "<<double-star>>"
	)
	normalizedPattern := filepath.ToSlash(pattern)
	normalizedPattern = strings.ReplaceAll(normalizedPattern, "**/", doubleStarDirToken)
	normalizedPattern = strings.ReplaceAll(normalizedPattern, "**", doubleStarToken)
	regexPattern := regexp.QuoteMeta(normalizedPattern)
	regexPattern = strings.ReplaceAll(regexPattern, regexp.QuoteMeta(doubleStarDirToken), `(?:.*/)?`)
	regexPattern = strings.ReplaceAll(regexPattern, regexp.QuoteMeta(doubleStarToken), `.*`)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `[^/]*`)
	regexPattern = strings.ReplaceAll(regexPattern, `\?`, `[^/]`)
	matched, err := regexp.MatchString("^"+regexPattern+"$", relPath)
	return err == nil && matched
}
