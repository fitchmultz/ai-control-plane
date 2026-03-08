// command_surface_parity_test.go - Repository-wide acpctl surface parity tests.
//
// Purpose:
//   - Keep published Make targets, acpctl examples, and generated completions
//     aligned with the live typed command registry.
//
// Responsibilities:
//   - Assert the approved validate/db/key/host/terraform subcommand inventory.
//   - Fail when docs or Make wrappers publish dead acpctl command paths.
//   - Fail when docs publish removed Make targets.
//   - Fail when committed completion artifacts drift from generated output.
//
// Scope:
//   - Repository policy coverage for published operator surfaces only.
//
// Usage:
//   - Run via `go test ./cmd/acpctl` or `make validate-acpctl-parity`.
//
// Invariants/Assumptions:
//   - cmd/acpctl remains the single source of truth for live command paths.
//   - Published docs use `./scripts/acpctl.sh` and `make <target>` as canonical entrypoints.
package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestCommandSpec_ApprovedCommandInventory(t *testing.T) {
	expected := map[string][]string{
		"validate":  {"lint", "config", "detections", "siem-queries", "public-hygiene", "license", "supply-chain", "secrets-audit", "compose-healthchecks", "headers", "env-access", "security"},
		"db":        {"status", "backup", "restore", "shell", "dr-drill"},
		"key":       {"gen", "revoke", "gen-dev", "gen-lead"},
		"host":      {"preflight", "check", "apply", "install", "uninstall", "service-status", "service-start", "service-stop", "service-restart", "secrets-refresh"},
		"terraform": {"init", "plan", "apply", "destroy", "fmt", "validate"},
	}

	spec, err := loadCommandSpec()
	if err != nil {
		t.Fatalf("loadCommandSpec() error = %v", err)
	}

	for root, want := range expected {
		node, ok := spec.NodesByPath[root]
		if !ok {
			t.Fatalf("expected root command %q", root)
		}
		got := make([]string, 0, len(node.Children))
		for _, child := range node.Children {
			if child.Hidden {
				continue
			}
			got = append(got, child.Name)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("subcommands for %q mismatch\nwant: %v\n got: %v", root, want, got)
		}
	}
}

func TestPublishedMakeTargetsResolve(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	targets := loadPublishedMakeTargets(t, repoRoot)
	files := append(
		documentationFiles(t, repoRoot),
		collectFiles(t, repoRoot, []string{"deploy/helm/ai-control-plane/templates"}, []string{".yaml", ".yml"})...,
	)

	var issues []string
	for _, file := range files {
		for lineNo, line := range readLines(t, file) {
			for _, snippet := range extractCommandSnippets(line, "make") {
				target, ok := publishedMakeTarget(snippet)
				if !ok {
					continue
				}
				if _, ok := targets[target]; ok {
					continue
				}
				issues = append(issues, formatIssue(t, repoRoot, file, lineNo+1, "unknown make target "+target))
			}
		}
	}

	if len(issues) > 0 {
		t.Fatalf("published make target drift detected:\n%s", strings.Join(issues, "\n"))
	}
}

func TestPublishedACPCTLCommandsResolve(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	spec, err := loadCommandSpec()
	if err != nil {
		t.Fatalf("loadCommandSpec() error = %v", err)
	}

	files := append(documentationFiles(t, repoRoot), collectFiles(t, repoRoot, []string{"deploy/helm/ai-control-plane/templates"}, []string{".yaml", ".yml"})...)
	files = append(files, collectFiles(t, repoRoot, []string{"mk"}, []string{".mk"})...)

	var issues []string
	for _, file := range files {
		for lineNo, line := range readLines(t, file) {
			for _, snippet := range extractCommandSnippets(line, "./scripts/acpctl.sh", "$(ACPCTL_BIN)") {
				if issue, ok := validatePublishedCommandLine(spec.Root, snippet); ok {
					issues = append(issues, formatIssue(t, repoRoot, file, lineNo+1, issue))
				}
			}
		}
	}

	if len(issues) > 0 {
		t.Fatalf("published acpctl command drift detected:\n%s", strings.Join(issues, "\n"))
	}
}

func TestGeneratedCompletionArtifactsAreCurrent(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	artifacts := map[string]string{
		"bash": filepath.Join(repoRoot, "scripts", "completions", "acpctl.bash"),
		"zsh":  filepath.Join(repoRoot, "scripts", "completions", "acpctl.zsh"),
		"fish": filepath.Join(repoRoot, "scripts", "completions", "acpctl.fish"),
	}

	for shell, path := range artifacts {
		generated := captureCompletionScript(t, shell)
		current, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if generated != string(current) {
			t.Fatalf("%s completion artifact is stale; run make completions", shell)
		}
	}
}

func TestRetiredCommandReferencesStayRemoved(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	files := append(documentationFiles(t, repoRoot), collectFiles(t, repoRoot, []string{"deploy/helm/ai-control-plane/templates", "mk"}, []string{".yaml", ".yml", ".mk"})...)
	files = append(files, filepath.Join(repoRoot, "Makefile"))
	retiredPatterns := []string{
		"rbac-whoami",
		"rbac-roles",
		"rbac-models",
		"rbac-check",
		"k8s-backup",
		"k8s-backup-verify",
		"k8s-dr-test",
		"k8s-dr-drill",
		"network-contract-check",
		"validate-network-contract",
		"make network-contract",
		"make helm-dr-drill",
		"make helm-db-backup-verify",
	}

	var issues []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(data)
		for _, pattern := range retiredPatterns {
			if strings.Contains(content, pattern) {
				issues = append(issues, mustRelPath(t, repoRoot, file)+": retired reference still present: "+pattern)
			}
		}
	}

	if len(issues) > 0 {
		t.Fatalf("retired command references reappeared:\n%s", strings.Join(issues, "\n"))
	}
}

func repoRootForParityTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("could not find repo root from %s", wd)
		}
		current = parent
	}
}

func loadPublishedMakeTargets(t *testing.T, repoRoot string) map[string]struct{} {
	t.Helper()
	files := append([]string{filepath.Join(repoRoot, "Makefile")}, collectFiles(t, repoRoot, []string{"mk"}, []string{".mk"})...)
	targets := make(map[string]struct{})
	targetPattern := regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*):`)
	phonyPattern := regexp.MustCompile(`^\.PHONY:\s+(.+)$`)

	for _, file := range files {
		for _, line := range readLines(t, file) {
			if match := phonyPattern.FindStringSubmatch(line); match != nil {
				for _, field := range strings.Fields(match[1]) {
					targets[field] = struct{}{}
				}
			}
			if match := targetPattern.FindStringSubmatch(line); match != nil {
				targets[match[1]] = struct{}{}
			}
		}
	}

	return targets
}

func documentationFiles(t *testing.T, repoRoot string) []string {
	t.Helper()
	files := collectFiles(t, repoRoot, []string{"docs", "demo"}, []string{".md"})
	files = append(files, filepath.Join(repoRoot, "README.md"))
	sort.Strings(files)
	return files
}

func collectFiles(t *testing.T, repoRoot string, dirs []string, extensions []string) []string {
	t.Helper()
	allowed := make(map[string]struct{}, len(extensions))
	for _, ext := range extensions {
		allowed[ext] = struct{}{}
	}

	var files []string
	for _, dir := range dirs {
		root := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "vendor", "node_modules", ".tmp-bin":
					return filepath.SkipDir
				}
				return nil
			}
			if len(allowed) > 0 {
				if _, ok := allowed[filepath.Ext(path)]; !ok {
					return nil
				}
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	sort.Strings(files)
	return files
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Split(string(data), "\n")
}

func extractCommandSnippets(line string, prefixes ...string) []string {
	snippets := make([]string, 0)
	trimmed := strings.TrimSpace(line)
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			snippets = append(snippets, trimmed)
		}
	}

	codeSpanPattern := regexp.MustCompile("`([^`]+)`")
	for _, match := range codeSpanPattern.FindAllStringSubmatch(line, -1) {
		content := strings.TrimSpace(match[1])
		for _, prefix := range prefixes {
			if strings.HasPrefix(content, prefix) {
				snippets = append(snippets, content)
				break
			}
		}
	}

	return dedupeStrings(snippets)
}

func publishedMakeTarget(snippet string) (string, bool) {
	fields := strings.Fields(snippet)
	if len(fields) < 2 || fields[0] != "make" || isSkippableCommandToken(fields[1]) {
		return "", false
	}
	if strings.Contains(fields[1], "*") {
		return "", false
	}
	return fields[1], true
}

func validatePublishedCommandLine(root *commandSpec, line string) (string, bool) {
	invokers := []string{"./scripts/acpctl.sh", "$(ACPCTL_BIN)"}
	var tail string
	for _, invoker := range invokers {
		if strings.HasPrefix(line, invoker) {
			tail = strings.TrimSpace(line[len(invoker):])
			break
		}
	}
	if tail == "" {
		return "", false
	}

	fields := strings.Fields(tail)
	if len(fields) == 0 || isSkippableCommandToken(fields[0]) {
		return "", false
	}

	current := root
	path := make([]string, 0, 4)
	for _, token := range fields {
		if isSkippableCommandToken(token) {
			break
		}
		child := findChildCommand(current, token)
		if child == nil {
			if len(path) == 0 {
				return "unknown root command " + token, true
			}
			if len(current.Children) > 0 {
				return "unknown subcommand " + strings.Join(append(append([]string(nil), path...), token), " "), true
			}
			break
		}
		path = append(path, token)
		current = child
	}

	return "", false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func isSkippableCommandToken(token string) bool {
	switch {
	case token == "":
		return true
	case strings.HasPrefix(token, "-"):
		return true
	case strings.HasPrefix(token, "#"):
		return true
	case strings.HasPrefix(token, "<"):
		return true
	case strings.HasPrefix(token, "["):
		return true
	case strings.Contains(token, "..."):
		return true
	default:
		return false
	}
}

func formatIssue(t *testing.T, repoRoot string, path string, lineNo int, message string) string {
	t.Helper()
	return mustRelPath(t, repoRoot, path) + ":" + strconv.Itoa(lineNo) + ": " + message
}

func mustRelPath(t *testing.T, repoRoot string, path string) string {
	t.Helper()
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		t.Fatalf("rel %s: %v", path, err)
	}
	return filepath.ToSlash(rel)
}
