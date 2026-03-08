// cmd_artifact_retention.go - Document artifact retention command
//
// Purpose: Enforce retention policy for generated document artifacts
//
// Responsibilities:
//   - Detect stale timestamped handoff evidence directories
//   - Detect stale release-bundle document sets
//   - Provide check mode for CI and apply mode for cleanup
//   - Keep the newest N artifacts per class
//
// Non-scope:
//   - Does NOT regenerate artifacts
//   - Does NOT mutate git history
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
	"strconv"
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

func runArtifactRetentionCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	config := artifactRetentionConfig{
		Mode:         "check",
		KeepEvidence: 1,
		KeepBundles:  1,
		RepoRoot:     detectRepoRootWithContext(ctx),
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check":
			config.Mode = "check"
		case "--apply":
			config.Mode = "apply"
		case "--keep-evidence":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --keep-evidence requires a positive integer")
				return exitcodes.ACPExitUsage
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 {
				fmt.Fprintln(stderr, "Error: --keep-evidence requires a positive integer")
				return exitcodes.ACPExitUsage
			}
			config.KeepEvidence = n
			i++
		case "--keep-bundles":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --keep-bundles requires a positive integer")
				return exitcodes.ACPExitUsage
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 1 {
				fmt.Fprintln(stderr, "Error: --keep-bundles requires a positive integer")
				return exitcodes.ACPExitUsage
			}
			config.KeepBundles = n
			i++
		case "--repo-root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Error: --repo-root requires a path")
				return exitcodes.ACPExitUsage
			}
			config.RepoRoot = args[i+1]
			i++
		case "--help":
			printArtifactRetentionHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			fmt.Fprintf(stderr, "Error: Unknown option '%s'\n", args[i])
			return exitcodes.ACPExitUsage
		}
	}

	out := output.New()
	evidenceRoot := filepath.Join(config.RepoRoot, "handoff-packet", "evidence")
	bundleRoot := filepath.Join(config.RepoRoot, "demo", "logs", "release-bundles")

	// Collect evidence directories
	evidenceOrdered := collectEvidenceDirs(evidenceRoot)

	// Collect bundle bases
	bundleBasesOrdered := collectBundleBases(bundleRoot)

	// Compute stale lists
	evidenceStale := computeStaleEvidence(evidenceOrdered, config.KeepEvidence, evidenceRoot)
	bundleFilesStale := computeStaleBundles(bundleBasesOrdered, config.KeepBundles, bundleRoot)

	// Print summary
	printStaleSummary(stdout, out, config, evidenceStale, bundleFilesStale)

	if config.Mode == "check" {
		if len(evidenceStale) > 0 || len(bundleFilesStale) > 0 {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, out.Fail("Retention check failed."))
			fmt.Fprintln(stdout, "Run: acpctl deploy artifact-retention --apply")
			return exitcodes.ACPExitDomain
		}
		return exitcodes.ACPExitSuccess
	}

	// Apply cleanup
	for _, stale := range evidenceStale {
		deleteEvidenceDir(stale)
	}
	for _, stale := range bundleFilesStale {
		os.Remove(stale)
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, out.Green("Retention cleanup applied successfully."))
	return exitcodes.ACPExitSuccess
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

	// Sort reverse (newest first)
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

	// Sort by modTime descending (newest first)
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
	// Remove all files in the directory
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			os.Remove(path)
		}
		return nil
	})
	// Remove empty directories
	os.RemoveAll(dir)
}

func printArtifactRetentionHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy artifact-retention [OPTIONS]

Enforce cleanup policy for stale document artifacts:
  - handoff-packet/evidence/<timestamp>/
  - demo/logs/release-bundles/*.tar.gz{,.sha256,.asc}

Options:
  --check                 Check only; fail if stale artifacts exist (default)
  --apply                 Delete stale artifacts, keeping newest artifacts
  --keep-evidence N       Number of newest evidence directories to keep (default: 1)
  --keep-bundles N        Number of newest release bundle sets to keep (default: 1)
  --repo-root PATH        Override repository root path
  --help                  Show this help message

Examples:
  # Check workspace retention compliance
  acpctl deploy artifact-retention --check

  # Keep only newest evidence dir and release bundle set
  acpctl deploy artifact-retention --apply

Exit codes:
  0   Success (compliant or cleanup applied)
  1   Stale artifacts found in --check mode
  64  Usage error
`)
}
