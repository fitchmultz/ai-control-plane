// cmd_artifact_retention.go - Document artifact retention command
//
// Purpose: Enforce retention policy for generated document artifacts.
//
// Responsibilities:
//   - Define the typed artifact-retention command surface.
//   - Detect stale timestamped handoff evidence directories.
//   - Detect stale release-bundle document sets.
//   - Provide check mode for CI and apply mode for cleanup.
//
// Non-scope:
//   - Does not regenerate artifacts.
//   - Does not mutate git history.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

type artifactRetentionConfig struct {
	Mode         string
	KeepEvidence int
	KeepBundles  int
	RepoRoot     string
}

func artifactRetentionCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "artifact-retention",
		Summary:     "Enforce document artifact retention policy",
		Description: "Enforce cleanup policy for stale generated document artifacts.",
		Options: []commandOptionSpec{
			{Name: "check", Summary: "Check only; fail if stale artifacts exist", Type: optionValueBool},
			{Name: "apply", Summary: "Delete stale artifacts", Type: optionValueBool},
			{Name: "keep-evidence", ValueName: "N", Summary: "Number of newest evidence directories to keep", Type: optionValueInt, DefaultText: "1"},
			{Name: "keep-bundles", ValueName: "N", Summary: "Number of newest release bundle sets to keep", Type: optionValueInt, DefaultText: "1"},
			{Name: "repo-root", ValueName: "PATH", Summary: "Override repository root path", Type: optionValueString},
		},
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bindArtifactRetentionOptions,
			NativeRun:  runArtifactRetention,
		},
	}
}

func bindArtifactRetentionOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	config := artifactRetentionConfig{
		Mode:         "check",
		KeepEvidence: 1,
		KeepBundles:  1,
		RepoRoot:     bindCtx.RepoRoot,
	}
	if input.Bool("apply") && input.Bool("check") {
		return nil, fmt.Errorf("--check and --apply cannot be used together")
	}
	if input.Bool("apply") {
		config.Mode = "apply"
	}
	if input.String("keep-evidence") != "" {
		n, err := input.Int("keep-evidence")
		if err != nil || n < 1 {
			return nil, fmt.Errorf("--keep-evidence requires a positive integer")
		}
		config.KeepEvidence = n
	}
	if input.String("keep-bundles") != "" {
		n, err := input.Int("keep-bundles")
		if err != nil || n < 1 {
			return nil, fmt.Errorf("--keep-bundles requires a positive integer")
		}
		config.KeepBundles = n
	}
	if input.String("repo-root") != "" {
		config.RepoRoot = input.String("repo-root")
	}
	return config, nil
}

func runArtifactRetention(_ context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(artifactRetentionConfig)
	out := output.New()
	evidenceRoot := filepath.Join(config.RepoRoot, "handoff-packet", "evidence")
	bundleRoot := filepath.Join(config.RepoRoot, "demo", "logs", "release-bundles")

	evidenceOrdered := collectEvidenceDirs(evidenceRoot)
	bundleBasesOrdered := collectBundleBases(bundleRoot)
	evidenceStale := computeStaleEvidence(evidenceOrdered, config.KeepEvidence, evidenceRoot)
	bundleFilesStale := computeStaleBundles(bundleBasesOrdered, config.KeepBundles, bundleRoot)

	printStaleSummary(runCtx.Stdout, out, config, evidenceStale, bundleFilesStale)

	if config.Mode == "check" {
		if len(evidenceStale) > 0 || len(bundleFilesStale) > 0 {
			fmt.Fprintln(runCtx.Stdout, "")
			fmt.Fprintln(runCtx.Stdout, out.Fail("Retention check failed."))
			fmt.Fprintln(runCtx.Stdout, "Run: acpctl deploy artifact-retention --apply")
			return exitcodes.ACPExitDomain
		}
		return exitcodes.ACPExitSuccess
	}

	for _, stale := range evidenceStale {
		deleteEvidenceDir(stale)
	}
	for _, stale := range bundleFilesStale {
		_ = os.Remove(stale)
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprintln(runCtx.Stdout, out.Green("Retention cleanup applied successfully."))
	return exitcodes.ACPExitSuccess
}

func runArtifactRetentionCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runTypedCommandAdapter(ctx, []string{"deploy", "artifact-retention"}, args, stdout, stderr)
}

func collectEvidenceDirs(evidenceRoot string) []string {
	var ordered []string

	entries, err := os.ReadDir(evidenceRoot)
	if err != nil {
		return ordered
	}

	pattern := regexp.MustCompile(`^\d{8}-\d{6}$`)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if pattern.MatchString(entry.Name()) {
			ordered = append(ordered, entry.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(ordered)))
	return ordered
}

func collectBundleBases(bundleRoot string) []string {
	var ordered []string

	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return ordered
	}

	type bundleInfo struct {
		name    string
		modTime int64
	}
	var bundles []bundleInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".tar.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		bundles = append(bundles, bundleInfo{
			name:    strings.TrimSuffix(entry.Name(), ".tar.gz"),
			modTime: info.ModTime().Unix(),
		})
	}

	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].modTime > bundles[j].modTime
	})

	for _, b := range bundles {
		ordered = append(ordered, b.name)
	}
	return ordered
}

func computeStaleEvidence(ordered []string, keep int, evidenceRoot string) []string {
	var stale []string
	for i := keep; i < len(ordered); i++ {
		stale = append(stale, filepath.Join(evidenceRoot, ordered[i]))
	}
	return stale
}

func computeStaleBundles(ordered []string, keep int, bundleRoot string) []string {
	var stale []string
	staleBases := make(map[string]bool)

	for i := keep; i < len(ordered); i++ {
		staleBases[ordered[i]] = true
	}

	if len(staleBases) == 0 {
		return stale
	}

	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return stale
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".tar.gz") &&
			!strings.HasSuffix(name, ".tar.gz.sha256") &&
			!strings.HasSuffix(name, ".tar.gz.asc") {
			continue
		}

		base := strings.TrimSuffix(name, ".sha256")
		base = strings.TrimSuffix(base, ".asc")
		base = strings.TrimSuffix(base, ".tar.gz")

		if staleBases[base] {
			stale = append(stale, filepath.Join(bundleRoot, name))
		}
	}

	return stale
}

func printStaleSummary(stdout *os.File, out *output.Output, config artifactRetentionConfig, evidenceStale, bundleFilesStale []string) {
	fmt.Fprintln(stdout, out.Bold("Retention policy results"))
	fmt.Fprintf(stdout, "  Repo root: %s\n", config.RepoRoot)
	fmt.Fprintf(stdout, "  Keep evidence dirs: %d\n", config.KeepEvidence)
	fmt.Fprintf(stdout, "  Keep release bundle sets: %d\n", config.KeepBundles)
	fmt.Fprintln(stdout, "")

	if len(evidenceStale) == 0 && len(bundleFilesStale) == 0 {
		fmt.Fprintln(stdout, out.Green("No stale document artifacts found."))
		return
	}

	if len(evidenceStale) > 0 {
		fmt.Fprintln(stdout, out.Yellow("Stale evidence directories:"))
		for _, dir := range evidenceStale {
			fmt.Fprintf(stdout, "  - %s\n", dir)
		}
	}

	if len(bundleFilesStale) > 0 {
		fmt.Fprintln(stdout, out.Yellow("Stale release bundle files:"))
		for _, file := range bundleFilesStale {
			fmt.Fprintf(stdout, "  - %s\n", file)
		}
	}
}

func deleteEvidenceDir(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			_ = os.Remove(path)
		}
		return nil
	})
	_ = os.RemoveAll(dir)
}
