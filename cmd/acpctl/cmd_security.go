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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
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

func runSecretsAudit(args []string, stdout *os.File, stderr *os.File) int {
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

	repoRoot := detectRepoRoot()
	trackedFiles, err := listTrackedFiles(repoRoot)
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

func listTrackedFiles(repoRoot string) ([]string, error) {
	if err := ensureExecutable("git"); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	rawPaths := bytes.Split(output, []byte{0})
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

func scanTrackedFiles(repoRoot string, trackedFiles []string) ([]secretFinding, error) {
	findings := make([]secretFinding, 0)
	for _, relPath := range trackedFiles {
		findings = append(findings, checkPathRules(relPath)...)

		absPath := filepath.Join(repoRoot, relPath)
		data, err := os.ReadFile(absPath)
		if err != nil {
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
