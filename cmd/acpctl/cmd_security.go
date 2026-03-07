// cmd_security.go - Security validation command implementations
//
// Purpose: Provide deterministic native security validations for the CLI.
// Responsibilities:
//   - Audit tracked repository files for likely secrets.
//   - Keep command help and exit codes aligned with actual enforcement.
//   - Honor repo-specific placeholder allowances for committed examples.
// Scope:
//   - Native `acpctl validate secrets-audit` behavior.
// Usage:
//   - Invoked through `acpctl validate secrets-audit`.
// Invariants/Assumptions:
//   - Scan scope is limited to tracked git files for deterministic results.
//   - High-confidence rules are preferred over broad keyword heuristics.

package main

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

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

type secretFinding struct {
	Path    string
	Line    int
	RuleID  string
	Message string
}

type secretPathRule struct {
	ID      string
	Message string
	Match   func(relPath string) bool
}

type secretContentRule struct {
	ID      string
	Message string
	Pattern *regexp.Regexp
}

var secretPathRules = []secretPathRule{
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

var secretContentRules = []secretContentRule{
	{
		ID:      "private-key-block",
		Message: "private key material",
		Pattern: regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
	},
	{
		ID:      "aws-access-key-id",
		Message: "AWS access key ID",
		Pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	},
	{
		ID:      "github-token",
		Message: "GitHub token",
		Pattern: regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`),
	},
	{
		ID:      "slack-token",
		Message: "Slack token",
		Pattern: regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{20,}\b`),
	},
	{
		ID:      "google-api-key",
		Message: "Google API key",
		Pattern: regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{20,}\b`),
	},
	{
		ID:      "openai-style-key",
		Message: "OpenAI-style API key",
		Pattern: regexp.MustCompile(`\bsk-[A-Za-z0-9][A-Za-z0-9_-]{20,}\b`),
	},
}

type supplyChainPolicyDocument struct {
	PolicyID       string                 `json:"policy_id"`
	Allowlist      []map[string]any       `json:"allowlist"`
	SeverityPolicy supplyChainSeverityDoc `json:"severity_policy"`
}

type supplyChainSeverityDoc struct {
	FailOn    []string       `json:"fail_on"`
	MaxCounts map[string]int `json:"max_counts"`
}

type licensePolicyDocument struct {
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

func runSecretsAudit(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printSecretsAuditHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printSecretsAuditHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}

	out := output.New()
	fmt.Fprintln(stdout, out.Bold("=== Secrets Audit ==="))
	fmt.Fprintln(stdout, "Scanning tracked files for likely public-repo secret leaks...")

	repoRoot := detectRepoRootWithContext(ctx)
	trackedFiles, err := listTrackedFiles(ctx, repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Secrets audit could not enumerate tracked files: %v\n"), err)
		return exitcodes.ACPExitPrereq
	}

	findings, err := scanTrackedFiles(repoRoot, trackedFiles)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Secrets audit failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if len(findings) > 0 {
		for _, finding := range findings {
			if finding.Line > 0 {
				fmt.Fprintf(stdout, "%s:%d [%s] %s\n", finding.Path, finding.Line, finding.RuleID, finding.Message)
				continue
			}
			fmt.Fprintf(stdout, "%s [%s] %s\n", finding.Path, finding.RuleID, finding.Message)
		}
		fmt.Fprintln(stderr, out.Fail("Secrets audit found tracked-file security issues"))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, out.Green("Secrets audit passed"))
	return exitcodes.ACPExitSuccess
}

func runValidatePublicHygiene(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printPublicHygieneHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printPublicHygieneHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}

	repoRoot := detectRepoRootWithContext(ctx)
	trackedFiles, err := listTrackedFiles(ctx, repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitPrereq
	}

	var violations []string
	for _, relPath := range trackedFiles {
		if !isLocalOnlyTrackedPath(relPath) {
			continue
		}
		if strings.HasSuffix(relPath, "/.gitkeep") || strings.HasSuffix(relPath, "/.gitignore") {
			continue
		}
		violations = append(violations, relPath)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		fmt.Fprintln(stderr, "Local-only files are tracked and block public release:")
		for _, violation := range violations {
			fmt.Fprintln(stderr, violation)
		}
		fmt.Fprintln(stderr, "Remove from git index (git rm --cached ...) and keep in .gitignore.")
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, "Public-release tracked file hygiene passed")
	return exitcodes.ACPExitSuccess
}

func runValidateLicense(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printLicenseHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printLicenseHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}

	repoRoot := detectRepoRootWithContext(ctx)
	policyPath := filepath.Join(repoRoot, "docs", "policy", "THIRD_PARTY_LICENSE_MATRIX.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: missing policy file: %s\n", policyPath)
		return exitcodes.ACPExitDomain
	}
	var policy licensePolicyDocument
	if err := json.Unmarshal(data, &policy); err != nil {
		fmt.Fprintf(stderr, "Error: parse %s: %v\n", policyPath, err)
		return exitcodes.ACPExitRuntime
	}
	if policy.SchemaVersion == nil || policy.PolicyID == nil || len(policy.ScanScope.Include) == 0 || policy.RestrictedComponents == nil {
		fmt.Fprintln(stderr, "Error: license policy JSON missing required fields")
		return exitcodes.ACPExitDomain
	}

	findings, err := findRestrictedLicenseReferences(repoRoot, policy)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(findings) > 0 {
		fmt.Fprintln(stderr, "Restricted LiteLLM enterprise references detected outside docs:")
		for _, finding := range findings {
			fmt.Fprintln(stderr, finding)
		}
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, "License boundary check passed")
	return exitcodes.ACPExitSuccess
}

func runValidateSupplyChain(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	for _, arg := range args {
		if isHelpToken(arg) {
			printSupplyChainHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
		if arg != "" {
			fmt.Fprintf(stderr, "Error: unknown option: %s\n", arg)
			printSupplyChainHelp(stderr)
			return exitcodes.ACPExitUsage
		}
	}

	repoRoot := detectRepoRootWithContext(ctx)
	policyPath := filepath.Join(repoRoot, "demo", "config", "supply_chain_vulnerability_policy.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		fmt.Fprintf(stderr, "Error: missing policy file: %s\n", policyPath)
		return exitcodes.ACPExitDomain
	}
	var policy supplyChainPolicyDocument
	if err := json.Unmarshal(data, &policy); err != nil {
		fmt.Fprintf(stderr, "Error: parse %s: %v\n", policyPath, err)
		return exitcodes.ACPExitRuntime
	}
	if policy.PolicyID == "" || policy.Allowlist == nil || policy.SeverityPolicy.FailOn == nil {
		fmt.Fprintln(stderr, "Error: supply-chain policy JSON missing required fields")
		return exitcodes.ACPExitDomain
	}

	violations, err := findNonDigestPinnedImages(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if len(violations) > 0 {
		fmt.Fprintln(stderr, "Found non-digest-pinned image reference(s) in demo/docker-compose*.yml:")
		for _, violation := range violations {
			fmt.Fprintln(stderr, violation)
		}
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintln(stdout, "Supply-chain policy and digest pinning baseline passed")
	return exitcodes.ACPExitSuccess
}

func listTrackedFiles(ctx context.Context, repoRoot string) ([]string, error) {
	if err := ensureExecutable("git"); err != nil {
		return nil, err
	}

	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    []string{"ls-files", "-z"},
		Dir:     repoRoot,
		Timeout: repoRootDetectTimeout,
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

func isLocalOnlyTrackedPath(relPath string) bool {
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

func findRestrictedLicenseReferences(repoRoot string, policy licensePolicyDocument) ([]string, error) {
	pathMatchers, err := compileRegexps(flattenLicensePathRegexps(policy.RestrictedComponents))
	if err != nil {
		return nil, err
	}
	contentMatchers, err := compileRegexps(flattenLicenseContentRegexps(policy.RestrictedComponents))
	if err != nil {
		return nil, err
	}

	var findings []string
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
			if shouldSkipLicenseDir(relPath, policy.ScanScope.Exclude) {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesAnyGlob(relPath, policy.ScanScope.Include) || matchesAnyGlob(relPath, policy.ScanScope.Exclude) {
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
	var patterns []string
	for _, component := range components {
		patterns = append(patterns, component.Match.PathRegex...)
	}
	return patterns
}

func flattenLicenseContentRegexps(components []licenseRestrictedComponent) []string {
	var patterns []string
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
			return nil, fmt.Errorf("compile restricted-component pattern %q: %w", pattern, err)
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

func shouldSkipLicenseDir(relPath string, excludePatterns []string) bool {
	if relPath == ".git" {
		return true
	}
	return matchesAnyGlob(relPath, excludePatterns)
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

func findNonDigestPinnedImages(repoRoot string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(repoRoot, "demo", "docker-compose*.yml"))
	if err != nil {
		return nil, err
	}
	var violations []string
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			return nil, err
		}
		relPath, err := filepath.Rel(repoRoot, match)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()
			trimmedLine := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedLine, "#") {
				continue
			}
			if !strings.HasPrefix(trimmedLine, "image:") {
				continue
			}
			if strings.Contains(line, "@sha256:") {
				continue
			}
			violations = append(violations, fmt.Sprintf("%s:%d:%s", filepath.ToSlash(relPath), lineNumber, strings.TrimSpace(line)))
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	sort.Strings(violations)
	return violations, nil
}

func printPublicHygieneHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate public-hygiene

Fail when local-only files are tracked by git.
`)
}

func printLicenseHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate license

Validate the third-party license policy contract and restricted reference boundary.
`)
}

func printSupplyChainHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate supply-chain

Validate supply-chain policy structure and digest pinning for compose images.
`)
}

func scanTrackedFiles(repoRoot string, trackedFiles []string) ([]secretFinding, error) {
	findings := make([]secretFinding, 0)
	for _, relPath := range trackedFiles {
		findings = append(findings, checkPathRules(relPath)...)

		absPath := filepath.Join(repoRoot, relPath)
		data, err := os.ReadFile(absPath)
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

func checkPathRules(relPath string) []secretFinding {
	findings := make([]secretFinding, 0)
	for _, rule := range secretPathRules {
		if !rule.Match(relPath) {
			continue
		}
		findings = append(findings, secretFinding{
			Path:    relPath,
			RuleID:  rule.ID,
			Message: rule.Message,
		})
	}
	return findings
}

func checkContentRules(relPath string, data []byte) ([]secretFinding, error) {
	findings := make([]secretFinding, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		for _, rule := range secretContentRules {
			matches := rule.Pattern.FindAllString(line, -1)
			for _, match := range matches {
				if isAllowedSecretExample(relPath, line, match) {
					continue
				}
				findings = append(findings, secretFinding{
					Path:    relPath,
					Line:    lineNumber,
					RuleID:  rule.ID,
					Message: rule.Message,
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return findings, nil
}

func shouldScanContent(relPath string, data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	if !utf8.Valid(data) {
		return false
	}

	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".ico", ".woff", ".woff2", ".ttf", ".eot":
		return false
	default:
		return true
	}
}

func isAllowedSecretExample(relPath string, line string, match string) bool {
	normalized := strings.ToLower(strings.TrimSpace(match))
	if normalized == "sk-litellm-master-change-me" || normalized == "sk-litellm-salt-change-me" {
		return true
	}

	if !isPlaceholderContext(relPath, line) {
		return false
	}

	if strings.Contains(normalized, "change-me") ||
		strings.Contains(normalized, "example") ||
		strings.Contains(normalized, "-demo-") ||
		strings.Contains(normalized, "-test-") ||
		strings.Contains(normalized, "your-") ||
		strings.Contains(normalized, "developer") ||
		strings.Contains(normalized, "virtual-key") ||
		strings.Contains(normalized, "placeholder") {
		return true
	}

	return strings.Contains(line, "PLACEHOLDER") || strings.Contains(line, "Format:")
}

func isExampleTemplatePath(relPath string) bool {
	base := strings.ToLower(filepath.Base(relPath))
	return strings.HasSuffix(base, ".example") ||
		strings.HasSuffix(base, ".sample") ||
		strings.HasSuffix(base, ".template") ||
		strings.HasSuffix(base, ".tmpl")
}

func isPlaceholderContext(relPath string, line string) bool {
	normalizedPath := strings.ToLower(relPath)
	return isExampleTemplatePath(relPath) ||
		strings.HasSuffix(normalizedPath, ".md") ||
		strings.HasSuffix(normalizedPath, ".yaml") ||
		strings.HasSuffix(normalizedPath, ".yml") ||
		strings.HasSuffix(normalizedPath, "_test.go") ||
		strings.Contains(normalizedPath, "/tests/") ||
		strings.Contains(normalizedPath, "/examples/") ||
		strings.Contains(strings.ToLower(line), "placeholder") ||
		strings.Contains(strings.ToLower(line), "demo key")
}

func printSecretsAuditHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl validate secrets-audit [OPTIONS]

Run a deterministic tracked-file secrets audit for public-repo safety.

Behavior:
  - Enumerates tracked files with git ls-files
  - Flags tracked .env/private-key material and high-confidence secret patterns
  - Allows documented placeholder values in committed example/template files

Options:
  --help, -h        Show this help message

Exit codes:
  0   Audit passed
  1   Tracked-file secret risk found
  2   Prerequisites not ready (for example: git unavailable)
  3   Runtime/internal error
  64  Usage error
`)
}
