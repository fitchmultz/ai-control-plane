// command_surface_parity_test.go - Repository-wide acpctl surface parity tests.
//
// Purpose:
//   - Keep published Make targets, acpctl examples, and generated completions
//     aligned with the live typed command registry.
//
// Responsibilities:
//   - Assert the approved validate/db/key/host/deploy subcommand inventory.
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

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
)

func TestCommandSpec_ApprovedCommandInventory(t *testing.T) {
	expected := map[string][]string{
		"validate": {"lint", "config", "detections", "siem-queries", "policy-rules", "tenant", "public-hygiene", "license", "supply-chain", "secrets-audit", "compose-healthchecks", "headers", "env-access", "security"},
		"policy":   {"eval"},
		"db":       {"status", "backup", "backup-retention", "restore", "off-host-drill", "shell", "dr-drill"},
		"key":      {"gen", "list", "inspect", "rotate", "revoke", "gen-dev", "gen-lead"},
		"cert":     {"list", "inspect", "check", "renew", "renew-auto"},
		"ops":      {"report"},
		"host":     {"preflight", "check", "apply", "failover-drill", "install", "uninstall", "service-status", "service-start", "service-stop", "service-restart"},
		"upgrade":  {"plan", "check", "execute", "rollback"},
		"deploy":   {"release-bundle", "readiness-evidence", "pilot-closeout-bundle", "assessor-packet", "artifact-retention"},
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

	files := append(documentationFiles(t, repoRoot), collectFiles(t, repoRoot, []string{"mk"}, []string{".mk"})...)

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

func TestGeneratedReferenceArtifactsAreCurrent(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	artifacts, err := generatedReferenceArtifacts(repoRoot)
	if err != nil {
		t.Fatalf("generatedReferenceArtifacts() error = %v", err)
	}
	for path, generated := range artifacts {
		current, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if generated != string(current) {
			t.Fatalf("%s is stale; run make generate", mustRelPath(t, repoRoot, path))
		}
	}
}

func TestPublicSurfaceOmitsIncubatingTracks(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	matrix, err := catalog.LoadSupportMatrix(filepath.Join(repoRoot, "docs", "support-matrix.yaml"))
	if err != nil {
		t.Fatalf("LoadSupportMatrix() error = %v", err)
	}

	files := append(publicDocumentationFiles(t, repoRoot, matrix), filepath.Join(repoRoot, "Makefile"))
	var issues []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := strings.ToLower(string(data))
		for _, term := range matrix.IncubatingTerms {
			needle := strings.ToLower(strings.TrimSpace(term))
			if needle == "" || !strings.Contains(content, needle) {
				continue
			}
			issues = append(issues, mustRelPath(t, repoRoot, file)+": incubating term exposed in supported surface: "+needle)
		}
	}

	if len(issues) > 0 {
		t.Fatalf("public supported surface still exposes incubating tracks:\n%s", strings.Join(issues, "\n"))
	}
}

func TestDocumentationLinksStayRepoRelative(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	files := append(documentationFiles(t, repoRoot), filepath.Join(repoRoot, "deploy", "incubating", "README.md"))
	files = append(files,
		filepath.Join(repoRoot, "docs", "deployment", "KUBERNETES_HELM.md"),
		filepath.Join(repoRoot, "docs", "deployment", "TERRAFORM.md"),
	)
	files = dedupeStrings(files)

	linkPattern := regexp.MustCompile(`\]\((/[^)]+)\)`)
	var issues []string
	for _, file := range files {
		for lineNo, line := range readLines(t, file) {
			for _, match := range linkPattern.FindAllStringSubmatch(line, -1) {
				target := match[1]
				issues = append(issues, formatIssue(t, repoRoot, file, lineNo+1, "absolute link target "+target))
			}
			if strings.Contains(line, "/Users/") {
				issues = append(issues, formatIssue(t, repoRoot, file, lineNo+1, "absolute local filesystem path"))
			}
		}
	}

	if len(issues) > 0 {
		t.Fatalf("documentation links must stay repo-relative:\n%s", strings.Join(issues, "\n"))
	}
}

func TestRetiredCommandReferencesStayRemoved(t *testing.T) {
	repoRoot := repoRootForParityTests(t)
	files := append(documentationFiles(t, repoRoot), collectFiles(t, repoRoot, []string{"mk"}, []string{".yaml", ".yml", ".mk"})...)
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
		"demo/scripts/validate_deployment_config.sh",
		"./scripts/acpctl.sh bridge host_deploy",
		"./scripts/acpctl.sh bridge host_install",
		"./scripts/acpctl.sh bridge prepare_secrets_env",
		"./scripts/acpctl.sh bridge prod_smoke_test",
		"./scripts/acpctl.sh demo ",
		"./scripts/acpctl.sh terraform ",
		"./scripts/acpctl.sh helm ",
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
	matrix, err := catalog.LoadSupportMatrix(filepath.Join(repoRoot, "docs", "support-matrix.yaml"))
	if err != nil {
		t.Fatalf("LoadSupportMatrix() error = %v", err)
	}
	files := append(publicDocumentationFiles(t, repoRoot, matrix), referenceDocumentationFiles(t, repoRoot, matrix)...)
	sort.Strings(files)
	return dedupeStrings(files)
}

func publicDocumentationFiles(t *testing.T, repoRoot string, matrix catalog.SupportMatrix) []string {
	t.Helper()
	return resolveTrackedDocPaths(t, repoRoot, matrix.PublicDocs)
}

func referenceDocumentationFiles(t *testing.T, repoRoot string, matrix catalog.SupportMatrix) []string {
	t.Helper()
	return resolveTrackedDocPaths(t, repoRoot, matrix.ReferenceDocs)
}

func resolveTrackedDocPaths(t *testing.T, repoRoot string, paths []string) []string {
	t.Helper()
	files := make([]string, 0, len(paths))
	for _, path := range paths {
		resolved := filepath.Join(repoRoot, filepath.FromSlash(path))
		if _, err := os.Stat(resolved); err == nil {
			files = append(files, resolved)
		}
	}
	sort.Strings(files)
	return dedupeStrings(files)
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

func TestExtractCommandSnippetsFindsFencedAndInlineCommands(t *testing.T) {
	line := "Use `make validate-detections` or `./scripts/acpctl.sh validate detections` before review."
	got := extractCommandSnippets(line, "make", "./scripts/acpctl.sh")
	want := []string{"make validate-detections", "./scripts/acpctl.sh validate detections"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractCommandSnippets() mismatch\nwant: %v\n got: %v", want, got)
	}
}

func TestExtractCommandSnippetsIgnoresCommentLinesWithoutCommandPrefix(t *testing.T) {
	got := extractCommandSnippets("# make validate-detections", "make")
	if len(got) != 0 {
		t.Fatalf("expected commented command to be ignored, got %v", got)
	}
}

func TestPublishedMakeTargetIgnoresWildcardsAndEnvPrefixes(t *testing.T) {
	if _, ok := publishedMakeTarget("FOO=bar make validate-detections"); ok {
		t.Fatal("expected env-prefixed make command to be ignored")
	}
	if _, ok := publishedMakeTarget("make validate-*"); ok {
		t.Fatal("expected wildcard make command to be ignored")
	}
	target, ok := publishedMakeTarget("make validate-detections")
	if !ok || target != "validate-detections" {
		t.Fatalf("expected concrete make target, got %q %v", target, ok)
	}
}

func TestValidatePublishedCommandLineHandlesLeavesAndInvalidSubcommands(t *testing.T) {
	spec, err := loadCommandSpec()
	if err != nil {
		t.Fatalf("loadCommandSpec() error = %v", err)
	}
	if issue, ok := validatePublishedCommandLine(spec.Root, "./scripts/acpctl.sh validate detections --verbose"); ok {
		t.Fatalf("expected valid leaf command, got %q", issue)
	}
	if issue, ok := validatePublishedCommandLine(spec.Root, "./scripts/acpctl.sh validate nope"); !ok || issue != "unknown subcommand validate nope" {
		t.Fatalf("expected invalid subcommand issue, got %q %v", issue, ok)
	}
	if issue, ok := validatePublishedCommandLine(spec.Root, "./scripts/acpctl.sh not-a-root"); !ok || issue != "unknown root command not-a-root" {
		t.Fatalf("expected invalid root issue, got %q %v", issue, ok)
	}
}
