// completion_test.go - Tests for shell completion support
//
// Purpose:
//
//	Provide unit tests for completion.go functionality, ensuring correct
//	behavior for completion generation and dynamic suggestion resolution.
//
// Responsibilities:
//   - Test runCompletionSubcommand for all shells and error cases
//   - Test runHiddenComplete for dynamic completion scenarios
//   - Test parser functions for extracting data from config files
//   - Ensure deterministic, sorted output
//
// Non-scope:
//   - Does not test actual shell integration (done via shell scripts)
//   - Does not test filesystem I/O beyond temp file fixtures
//
// Invariants/Assumptions:
//   - Tests use temporary files to avoid dependency on mutable repo state
//   - Parser tests use inline fixtures for deterministic behavior
package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunCompletionSubcommand_NoArgs(t *testing.T) {
	stdout := os.Stdout
	stderr, err := os.CreateTemp("", "acpctl_test_stderr")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stderr.Name())

	exitCode := runCompletionSubcommand(context.Background(), []string{}, stdout, stderr)

	if exitCode != exitcodes.ACPExitUsage {
		t.Errorf("expected exit code %d for no args, got %d", exitcodes.ACPExitUsage, exitCode)
	}

	stderr.Seek(0, 0)
	buf := new(bytes.Buffer)
	buf.ReadFrom(stderr)
	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("expected usage message in stderr, got: %s", buf.String())
	}
}

func TestRunCompletionSubcommand_UnsupportedShell(t *testing.T) {
	stdout := os.Stdout
	stderr, err := os.CreateTemp("", "acpctl_test_stderr")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stderr.Name())

	exitCode := runCompletionSubcommand(context.Background(), []string{"powershell"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitUsage {
		t.Errorf("expected exit code %d for unsupported shell, got %d", exitcodes.ACPExitUsage, exitCode)
	}
}

func TestRunCompletionSubcommand_Bash(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())
	stderr := os.Stderr

	exitCode := runCompletionSubcommand(context.Background(), []string{"bash"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Errorf("expected exit code %d for bash, got %d", exitcodes.ACPExitSuccess, exitCode)
	}

	stdout.Seek(0, 0)
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	content := buf.String()

	if !strings.Contains(content, "_acpctl_complete") {
		t.Errorf("expected bash completion function, got: %s", content)
	}
	if !strings.Contains(content, "complete -o default -F _acpctl_complete acpctl") {
		t.Errorf("expected complete command, got: %s", content)
	}
}

func TestRunCompletionSubcommand_Zsh(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())
	stderr := os.Stderr

	exitCode := runCompletionSubcommand(context.Background(), []string{"zsh"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Errorf("expected exit code %d for zsh, got %d", exitcodes.ACPExitSuccess, exitCode)
	}

	stdout.Seek(0, 0)
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	content := buf.String()

	if !strings.Contains(content, "#compdef acpctl") {
		t.Errorf("expected zsh compdef header, got: %s", content)
	}
	if !strings.Contains(content, "_acpctl()") {
		t.Errorf("expected _acpctl function, got: %s", content)
	}
}

func TestRunCompletionSubcommand_Fish(t *testing.T) {
	stdout, err := os.CreateTemp("", "acpctl_test_stdout")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(stdout.Name())
	stderr := os.Stderr

	exitCode := runCompletionSubcommand(context.Background(), []string{"fish"}, stdout, stderr)

	if exitCode != exitcodes.ACPExitSuccess {
		t.Errorf("expected exit code %d for fish, got %d", exitcodes.ACPExitSuccess, exitCode)
	}

	stdout.Seek(0, 0)
	buf := new(bytes.Buffer)
	buf.ReadFrom(stdout)
	content := buf.String()

	if !strings.Contains(content, "function __acpctl_complete") {
		t.Errorf("expected fish function, got: %s", content)
	}
	if !strings.Contains(content, "complete -c acpctl") {
		t.Errorf("expected complete command, got: %s", content)
	}
}

func TestResolveSuggestions_RootCommands(t *testing.T) {
	catalog := completionCatalog{
		rootCommands: []string{"ci", "files", "status", "deploy"},
		groupSubcommands: map[string][]string{
			"deploy": {"up", "down", "health"},
		},
	}

	suggestions := resolveSuggestions([]string{}, "", catalog)

	if len(suggestions) == 0 {
		t.Error("expected non-empty suggestions for empty context")
	}

	// Should include root commands
	found := slices.Contains(suggestions, "ci")
	if !found {
		t.Errorf("expected 'ci' in suggestions, got: %v", suggestions)
	}
}

func TestResolveSuggestions_GroupSubcommands(t *testing.T) {
	catalog := completionCatalog{
		rootCommands: []string{"ci", "deploy"},
		groupSubcommands: map[string][]string{
			"deploy": {"up", "down", "health"},
		},
	}

	// First word is "deploy", suggest subcommands
	suggestions := resolveSuggestions([]string{"deploy"}, "", catalog)

	if len(suggestions) == 0 {
		t.Error("expected non-empty suggestions for group context")
	}

	// Should include deploy subcommands
	hasUp := false
	hasHealth := false
	for _, s := range suggestions {
		if s == "up" {
			hasUp = true
		}
		if s == "health" {
			hasHealth = true
		}
	}
	if !hasUp {
		t.Errorf("expected 'up' in deploy subcommands, got: %v", suggestions)
	}
	if !hasHealth {
		t.Errorf("expected 'health' in deploy subcommands, got: %v", suggestions)
	}
}

func TestResolveSuggestions_KeyPrefixes(t *testing.T) {
	catalog := completionCatalog{
		keyAliases:  []string{"alice", "bob"},
		modelNames:  []string{"claude-sonnet", "gpt-4"},
		scenarioIDs: []string{"1", "2", "3"},
	}

	// Test ALIAS= prefix
	suggestions := resolveSuggestions([]string{}, "ALIAS=", catalog)
	if len(suggestions) != 2 {
		t.Errorf("expected 2 alias suggestions, got %d: %v", len(suggestions), suggestions)
	}
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "ALIAS=") {
			t.Errorf("expected ALIAS= prefix, got: %s", s)
		}
	}

	// Test MODEL= prefix
	suggestions = resolveSuggestions([]string{}, "MODEL=", catalog)
	if len(suggestions) != 2 {
		t.Errorf("expected 2 model suggestions, got %d: %v", len(suggestions), suggestions)
	}
	for _, s := range suggestions {
		if !strings.HasPrefix(s, "MODEL=") {
			t.Errorf("expected MODEL= prefix, got: %s", s)
		}
	}

	// Test SCENARIO= prefix
	suggestions = resolveSuggestions([]string{}, "SCENARIO=", catalog)
	if len(suggestions) != 3 {
		t.Errorf("expected 3 scenario suggestions, got %d: %v", len(suggestions), suggestions)
	}
}

func TestResolveSuggestions_BridgeCommands(t *testing.T) {
	catalog := completionCatalog{
		rootCommands: []string{"bridge", "deploy"},
		groupSubcommands: map[string][]string{
			"bridge": {"host_deploy", "onboard"},
		},
		bridgeNames: []string{"host_deploy", "onboard"},
	}

	suggestions := resolveSuggestions([]string{"bridge"}, "", catalog)

	if len(suggestions) != 2 {
		t.Errorf("expected 2 bridge suggestions, got %d: %v", len(suggestions), suggestions)
	}

	hasHostDeploy := false
	for _, s := range suggestions {
		if s == "host_deploy" {
			hasHostDeploy = true
		}
	}
	if !hasHostDeploy {
		t.Errorf("expected 'host_deploy' in bridge suggestions, got: %v", suggestions)
	}
}

func TestDedupeAndSort(t *testing.T) {
	input := []string{"charlie", "alice", "bob", "alice", "charlie"}
	expected := []string{"alice", "bob", "charlie"}

	result := dedupeAndSort(input)

	if len(result) != len(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, result[i])
		}
	}
}

func TestExtractModelNames(t *testing.T) {
	// Create a temporary directory with a test litellm.yaml
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "demo", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := `---
model_list:
  - model_name: test-model-1
    litellm_params:
      model: provider/model-1
  - model_name: test-model-2
    litellm_params:
      model: provider/model-2
  - model_name: duplicate-model
  - model_name: duplicate-model
`
	litellmPath := filepath.Join(configDir, "litellm.yaml")
	if err := os.WriteFile(litellmPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write litellm.yaml: %v", err)
	}

	models := extractModelNames(tmpDir)

	if len(models) != 3 {
		t.Errorf("expected 3 unique models, got %d: %v", len(models), models)
	}

	expected := []string{"duplicate-model", "test-model-1", "test-model-2"}
	for i, v := range expected {
		if models[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, models[i])
		}
	}
}

func TestExtractScenarioIDs(t *testing.T) {
	// Create a temporary directory with test scenario files
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scenarioDir := filepath.Join(tmpDir, "local", "scripts", "demo_scenarios")
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	// Create test scenario files
	scenarios := []string{
		"scenario_1_test.sh",
		"scenario_5_another.sh",
		"scenario_12_multiple_digits.sh",
		"not_a_scenario.sh",
	}

	for _, name := range scenarios {
		path := filepath.Join(scenarioDir, name)
		if err := os.WriteFile(path, []byte("#!/bin/bash"), 0755); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	ids := extractScenarioIDs(tmpDir)

	if len(ids) != 3 {
		t.Errorf("expected 3 scenario IDs, got %d: %v", len(ids), ids)
	}

	expected := []string{"1", "12", "5"}
	for i, v := range expected {
		if ids[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, ids[i])
		}
	}
}

func TestExtractPresetNames(t *testing.T) {
	// Create a temporary directory with a test demo_presets.yaml
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "demo", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := `presets:
  test-preset-1:
    name: "Test Preset 1"
    description: "Test"
    scenarios:
      - 1
  another-preset:
    name: "Another Preset"
    timeout_minutes: 10
  z-last-preset:
    name: "Z Last"

settings:
  default_timeout_minutes: 5
`
	presetsPath := filepath.Join(configDir, "demo_presets.yaml")
	if err := os.WriteFile(presetsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write demo_presets.yaml: %v", err)
	}

	presets := extractPresetNames(tmpDir)

	if len(presets) != 3 {
		t.Errorf("expected 3 presets, got %d: %v", len(presets), presets)
	}

	expected := []string{"another-preset", "test-preset-1", "z-last-preset"}
	for i, v := range expected {
		if presets[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, presets[i])
		}
	}
}

func TestExtractConfigKeys(t *testing.T) {
	// Create a temporary directory with test config files
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "demo", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	content := `---
top_level_key:
  nested: value
another_key:
  - item1
  - item2
`
	testConfigPath := filepath.Join(configDir, "test_config.yaml")
	if err := os.WriteFile(testConfigPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test_config.yaml: %v", err)
	}

	keys := extractConfigKeys(tmpDir)

	if len(keys) != 2 {
		t.Errorf("expected 2 config keys, got %d: %v", len(keys), keys)
	}

	// Should be sorted
	if keys[0] != "another_key" {
		t.Errorf("expected 'another_key' first (alphabetically), got: %s", keys[0])
	}
	if keys[1] != "top_level_key" {
		t.Errorf("expected 'top_level_key' second, got: %s", keys[1])
	}
}

func TestExtractKeyAliases(t *testing.T) {
	// Create a temporary directory with test scenario scripts
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	scenarioDir := filepath.Join(tmpDir, "local", "scripts", "demo_scenarios")
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	content := `#!/bin/bash
SCENARIO_KEY_ALIAS="test-alias-1"
KEY_ALIAS="another-alias"
SCENARIO_KEY_ALIAS="duplicate-alias"
SCENARIO_KEY_ALIAS="duplicate-alias"
`
	scenarioPath := filepath.Join(scenarioDir, "scenario_1_test.sh")
	if err := os.WriteFile(scenarioPath, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	aliases := extractKeyAliases(tmpDir)

	if len(aliases) != 3 {
		t.Errorf("expected 3 unique aliases, got %d: %v", len(aliases), aliases)
	}

	expected := []string{"another-alias", "duplicate-alias", "test-alias-1"}
	for i, v := range expected {
		if aliases[i] != v {
			t.Errorf("expected %s at index %d, got %s", v, i, aliases[i])
		}
	}
}

func TestBuildCompletionCatalog(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "acpctl_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, "demo", "config")
	scenarioDir := filepath.Join(tmpDir, "local", "scripts", "demo_scenarios")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(scenarioDir, 0755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	// Create litellm.yaml
	litellmContent := `---
model_list:
  - model_name: test-model
    litellm_params:
      model: provider/test
`
	if err := os.WriteFile(filepath.Join(configDir, "litellm.yaml"), []byte(litellmContent), 0644); err != nil {
		t.Fatalf("failed to write litellm.yaml: %v", err)
	}

	// Create demo_presets.yaml
	presetsContent := `presets:
  test-preset:
    name: "Test"
`
	if err := os.WriteFile(filepath.Join(configDir, "demo_presets.yaml"), []byte(presetsContent), 0644); err != nil {
		t.Fatalf("failed to write demo_presets.yaml: %v", err)
	}

	// Create scenario file
	scenarioContent := `#!/bin/bash
SCENARIO_KEY_ALIAS=test-key
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "scenario_1_test.sh"), []byte(scenarioContent), 0755); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	catalog := buildCompletionCatalog(tmpDir)

	// Check root commands
	if len(catalog.rootCommands) == 0 {
		t.Error("expected non-empty root commands")
	}
	if !slices.Contains(catalog.rootCommands, "health") {
		t.Fatalf("expected health in root commands, got: %v", catalog.rootCommands)
	}
	if !slices.Contains(catalog.rootCommands, "helm") {
		t.Fatalf("expected helm in root commands, got: %v", catalog.rootCommands)
	}

	// Check group subcommands
	if len(catalog.groupSubcommands) == 0 {
		t.Error("expected non-empty group subcommands")
	}
	if !slices.Contains(catalog.groupSubcommands["ci"], "wait") {
		t.Fatalf("expected ci wait in catalog, got: %v", catalog.groupSubcommands["ci"])
	}
	if !slices.Contains(catalog.groupSubcommands["benchmark"], "baseline") {
		t.Fatalf("expected benchmark baseline in catalog, got: %v", catalog.groupSubcommands["benchmark"])
	}
	if !slices.Contains(catalog.groupSubcommands["bridge"], "host_preflight") {
		t.Fatalf("expected bridge host_preflight in catalog, got: %v", catalog.groupSubcommands["bridge"])
	}
	if !slices.Contains(catalog.groupSubcommands["helm"], "smoke") {
		t.Fatalf("expected helm smoke in catalog, got: %v", catalog.groupSubcommands["helm"])
	}
	if !slices.Contains(catalog.groupSubcommands["deploy"], "artifact-retention") {
		t.Fatalf("expected deploy artifact-retention in catalog, got: %v", catalog.groupSubcommands["deploy"])
	}
	if !slices.Contains(catalog.groupSubcommands["validate"], "compose-healthchecks") {
		t.Fatalf("expected validate compose-healthchecks in catalog, got: %v", catalog.groupSubcommands["validate"])
	}

	// Check bridge names
	if len(catalog.bridgeNames) == 0 {
		t.Error("expected non-empty bridge names")
	}

	// Check extracted values
	if len(catalog.modelNames) != 1 || catalog.modelNames[0] != "test-model" {
		t.Errorf("expected ['test-model'], got: %v", catalog.modelNames)
	}

	if len(catalog.presetNames) != 1 || catalog.presetNames[0] != "test-preset" {
		t.Errorf("expected ['test-preset'], got: %v", catalog.presetNames)
	}

	if len(catalog.keyAliases) != 1 || catalog.keyAliases[0] != "test-key" {
		t.Errorf("expected ['test-key'], got: %v", catalog.keyAliases)
	}

	if len(catalog.scenarioIDs) != 1 || catalog.scenarioIDs[0] != "1" {
		t.Errorf("expected ['1'], got: %v", catalog.scenarioIDs)
	}
}

func TestBuildCompletionCatalog_ExcludesStaleCommands(t *testing.T) {
	catalog := buildCompletionCatalog(t.TempDir())

	forbidden := map[string][]string{
		"key":  {"rbac-whoami", "rbac-roles"},
		"host": {"upgrade-status"},
		"demo": {"status"},
	}

	for group, names := range forbidden {
		for _, name := range names {
			if slices.Contains(catalog.groupSubcommands[group], name) {
				t.Fatalf("unexpected stale subcommand %q in %q: %v", name, group, catalog.groupSubcommands[group])
			}
		}
	}
}

func TestGeneratedCompletionScriptsFollowCatalog(t *testing.T) {
	catalog := buildCompletionCatalog(t.TempDir())

	for _, shell := range []string{"bash", "zsh", "fish"} {
		content := captureCompletionScript(t, shell)

		for _, root := range catalog.rootCommands {
			if !strings.Contains(content, root) {
				t.Fatalf("%s completion missing root command %q", shell, root)
			}
		}

		for _, group := range []string{"ci", "completion", "bridge", "deploy", "validate", "helm"} {
			for _, subcommand := range catalog.groupSubcommands[group] {
				if !strings.Contains(content, subcommand) {
					t.Fatalf("%s completion missing %s subcommand %q", shell, group, subcommand)
				}
			}
		}

		for _, stale := range []string{"rbac-whoami", "rbac-roles", "upgrade-status"} {
			if strings.Contains(content, stale) {
				t.Fatalf("%s completion unexpectedly contains stale subcommand %q", shell, stale)
			}
		}
	}
}

func TestNativeHelpSurfacesFollowRegistry(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		renderHelp func(*os.File)
	}{
		{name: "ci", command: "ci", renderHelp: printCIHelp},
		{name: "files", command: "files", renderHelp: printFilesHelp},
		{name: "benchmark", command: "benchmark", renderHelp: printBenchmarkHelp},
		{name: "completion", command: "completion", renderHelp: printCompletionHelp},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := captureHelpOutput(t, tc.renderHelp)
			command := mustLookupNativeCommand(tc.command)
			for _, subcommand := range command.Subcommands {
				if !strings.Contains(content, subcommand.Name) {
					t.Fatalf("%s help missing subcommand %q", tc.name, subcommand.Name)
				}
				if !strings.Contains(content, subcommand.Description) {
					t.Fatalf("%s help missing description for %q", tc.name, subcommand.Name)
				}
			}
		})
	}
}

func captureCompletionScript(t *testing.T, shell string) string {
	t.Helper()

	stdout, err := os.CreateTemp("", "acpctl_completion_stdout")
	if err != nil {
		t.Fatalf("failed to create temp stdout: %v", err)
	}
	defer os.Remove(stdout.Name())

	stderr, err := os.CreateTemp("", "acpctl_completion_stderr")
	if err != nil {
		t.Fatalf("failed to create temp stderr: %v", err)
	}
	defer os.Remove(stderr.Name())

	if exitCode := runCompletionSubcommand(context.Background(), []string{shell}, stdout, stderr); exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success generating %s completion, got %d", shell, exitCode)
	}

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatalf("failed to seek stdout: %v", err)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(stdout); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	return buf.String()
}

func captureHelpOutput(t *testing.T, renderHelp func(*os.File)) string {
	t.Helper()

	stdout, err := os.CreateTemp("", "acpctl_help_stdout")
	if err != nil {
		t.Fatalf("failed to create temp help stdout: %v", err)
	}
	defer os.Remove(stdout.Name())

	renderHelp(stdout)

	if _, err := stdout.Seek(0, 0); err != nil {
		t.Fatalf("failed to seek help stdout: %v", err)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(stdout); err != nil {
		t.Fatalf("failed to read help stdout: %v", err)
	}

	return buf.String()
}
