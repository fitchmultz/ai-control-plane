// Package release implements typed release and evidence workflows.
//
// Purpose:
//
//	Centralize generated artifact-run directory lifecycle for readiness
//	evidence, pilot closeout bundles, and related release outputs.
//
// Responsibilities:
//   - Create timestamped run directories with stable identifiers.
//   - Persist generated artifacts, inventories, and latest-run pointers.
//   - Verify generated inventory files against the filesystem.
//
// Scope:
//   - Generated run-directory management only.
//
// Usage:
//   - Called by readiness and pilot closeout workflows.
//
// Invariants/Assumptions:
//   - Inventories list repository-relative paths within a run directory.
//   - Latest pointers always end with a trailing newline.
package release

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

type generatedArtifact struct {
	Path string
	Body []byte
	Perm os.FileMode
}

type artifactRun struct {
	RunID        string
	RunDirectory string
}

func createArtifactRun(outputRoot string, prefix string, now time.Time) (*artifactRun, error) {
	runID := fmt.Sprintf("%s-%s", prefix, now.Format("20060102T150405Z"))
	runDir := filepath.Join(outputRoot, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact run directory: %w", err)
	}
	return &artifactRun{RunID: runID, RunDirectory: runDir}, nil
}

func writeGeneratedArtifacts(runDir string, artifacts []generatedArtifact) error {
	for _, artifact := range artifacts {
		target := filepath.Join(runDir, filepath.FromSlash(artifact.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create artifact parent %s: %w", target, err)
		}
		perm := artifact.Perm
		if perm == 0 {
			perm = 0o644
		}
		if err := fsutil.AtomicWriteFile(target, artifact.Body, perm); err != nil {
			return fmt.Errorf("write artifact %s: %w", artifact.Path, err)
		}
	}
	return nil
}

func writeJSONArtifact(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return fsutil.AtomicWriteFile(path, payload, 0o644)
}

func generatedRunFiles(runDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, relErr := filepath.Rel(runDir, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func finalizeRunInventory(runDir string, inventoryName string) ([]string, error) {
	files, err := generatedRunFiles(runDir)
	if err != nil {
		return nil, err
	}
	files = append(files, inventoryName)
	sort.Strings(files)
	inventoryPath := filepath.Join(runDir, inventoryName)
	if err := fsutil.AtomicWriteFile(inventoryPath, []byte(strings.Join(files, "\n")+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write inventory %s: %w", inventoryName, err)
	}
	return files, nil
}

func verifyRunInventory(runDir string, inventoryName string) error {
	inventoryPath := filepath.Join(runDir, inventoryName)
	inventoryData, err := os.ReadFile(inventoryPath)
	if err != nil {
		return fmt.Errorf("read inventory %s: %w", inventoryName, err)
	}
	expected := filterNonEmpty(strings.Split(strings.ReplaceAll(string(inventoryData), "\r\n", "\n"), "\n"))
	actual, err := generatedRunFiles(runDir)
	if err != nil {
		return fmt.Errorf("walk inventory for %s: %w", inventoryName, err)
	}
	if !stringSlicesEqual(expected, actual) {
		return fmt.Errorf("inventory mismatch between %s and filesystem", inventoryName)
	}
	return nil
}

func writeLatestRunPointer(outputRoot string, pointerName string, runDir string) error {
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return fmt.Errorf("create output root %s: %w", outputRoot, err)
	}
	return fsutil.AtomicWriteFile(filepath.Join(outputRoot, pointerName), []byte(runDir+"\n"), 0o644)
}

func resolveLatestRunPointer(outputRoot string, pointerName string) (string, error) {
	data, err := os.ReadFile(filepath.Join(outputRoot, pointerName))
	if err != nil {
		return "", fmt.Errorf("read latest run pointer %s: %w", pointerName, err)
	}
	runDir := strings.TrimSpace(string(data))
	if runDir == "" {
		return "", fmt.Errorf("latest run pointer %s is empty", pointerName)
	}
	return runDir, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
