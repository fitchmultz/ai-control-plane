// artifact_run.go - Shared artifact-run lifecycle primitives.
//
// Purpose:
//
//	Centralize generated run-directory lifecycle for readiness evidence and
//	pilot closeout bundles.
//
// Responsibilities:
//   - Create collision-safe timestamped run directories.
//   - Persist generated artifacts and JSON summaries atomically.
//   - Materialize inventory files and latest-run pointers.
//   - Verify generated run directories against their inventories.
//
// Scope:
//   - Generic generated run-directory mechanics only.
//
// Usage:
//   - Imported by `internal/readiness` and `internal/closeout`.
//
// Invariants/Assumptions:
//   - Inventories list slash-normalized paths relative to a run directory.
//   - Latest pointers always contain an absolute run path ending with `\n`.
package artifactrun

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

// Artifact describes one generated artifact to persist under a run directory.
type Artifact struct {
	Path string
	Body []byte
	Perm os.FileMode
}

// Run captures the generated identifier and directory for one artifact run.
type Run struct {
	ID        string
	Directory string
}

// FinalizeOptions configures inventory and latest-pointer persistence.
type FinalizeOptions struct {
	InventoryName  string
	LatestPointers []string
}

// VerifyOptions configures run-directory verification expectations.
type VerifyOptions struct {
	InventoryName string
	RequiredFiles []string
}

// Create allocates a collision-safe timestamped artifact run directory.
func Create(outputRoot string, prefix string, now time.Time) (*Run, error) {
	if strings.TrimSpace(outputRoot) == "" {
		return nil, fmt.Errorf("output root is required")
	}
	if strings.TrimSpace(prefix) == "" {
		return nil, fmt.Errorf("run prefix is required")
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create output root %s: %w", outputRoot, err)
	}

	stamp := now.UTC().Format("20060102T150405.000000000Z")
	for attempt := 0; attempt < 1000; attempt++ {
		suffix := ""
		if attempt > 0 {
			suffix = fmt.Sprintf("-%02d", attempt)
		}
		runID := fmt.Sprintf("%s-%s%s", prefix, stamp, suffix)
		runDir := filepath.Join(outputRoot, runID)
		err := os.Mkdir(runDir, 0o755)
		switch {
		case err == nil:
			return &Run{ID: runID, Directory: runDir}, nil
		case errors.Is(err, os.ErrExist):
			continue
		default:
			return nil, fmt.Errorf("create artifact run directory: %w", err)
		}
	}

	return nil, fmt.Errorf("create artifact run directory: exhausted collision retries for %s", stamp)
}

// WriteArtifacts writes generated artifacts into a run directory atomically.
func WriteArtifacts(runDir string, artifacts []Artifact) error {
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

// WriteJSON marshals a JSON artifact with stable indentation and newline.
func WriteJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	payload = append(payload, '\n')
	if err := fsutil.AtomicWriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write json %s: %w", path, err)
	}
	return nil
}

// Finalize writes the inventory and latest pointers for a completed run.
func Finalize(runDir string, outputRoot string, opts FinalizeOptions) ([]string, error) {
	if strings.TrimSpace(opts.InventoryName) == "" {
		return nil, fmt.Errorf("inventory name is required")
	}

	files, err := generatedRunFiles(runDir)
	if err != nil {
		return nil, fmt.Errorf("walk run directory: %w", err)
	}
	filtered := make([]string, 0, len(files)+1)
	for _, file := range files {
		if file != opts.InventoryName {
			filtered = append(filtered, file)
		}
	}
	files = append(filtered, opts.InventoryName)
	sort.Strings(files)

	inventoryPath := filepath.Join(runDir, opts.InventoryName)
	if err := fsutil.AtomicWriteFile(inventoryPath, []byte(strings.Join(files, "\n")+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write inventory %s: %w", opts.InventoryName, err)
	}

	for _, pointerName := range opts.LatestPointers {
		if err := WriteLatestPointer(outputRoot, pointerName, runDir); err != nil {
			return nil, err
		}
	}

	return files, nil
}

// Verify confirms required generated files and inventory consistency.
func Verify(runDir string, opts VerifyOptions) error {
	for _, required := range opts.RequiredFiles {
		if !FileExists(filepath.Join(runDir, filepath.FromSlash(required))) {
			return fmt.Errorf("missing generated artifact: %s", required)
		}
	}
	if strings.TrimSpace(opts.InventoryName) != "" {
		if err := verifyInventory(runDir, opts.InventoryName); err != nil {
			return err
		}
	}
	return nil
}

// ReadInventory loads one run inventory as relative slash-normalized paths.
func ReadInventory(runDir string, inventoryName string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(runDir, inventoryName))
	if err != nil {
		return nil, fmt.Errorf("read inventory %s: %w", inventoryName, err)
	}
	return filterNonEmpty(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")), nil
}

// WriteLatestPointer updates a latest-run pointer file.
func WriteLatestPointer(outputRoot string, pointerName string, runDir string) error {
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return fmt.Errorf("create output root %s: %w", outputRoot, err)
	}
	if err := fsutil.AtomicWriteFile(filepath.Join(outputRoot, pointerName), []byte(runDir+"\n"), 0o644); err != nil {
		return fmt.Errorf("write latest run pointer %s: %w", pointerName, err)
	}
	return nil
}

// ResolveLatest reads one latest-run pointer file.
func ResolveLatest(outputRoot string, pointerName string) (string, error) {
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

// FileExists reports whether path exists and is a regular file.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func generatedRunFiles(runDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(runDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(runDir, path)
		if err != nil {
			return err
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

func verifyInventory(runDir string, inventoryName string) error {
	expected, err := ReadInventory(runDir, inventoryName)
	if err != nil {
		return err
	}
	actual, err := generatedRunFiles(runDir)
	if err != nil {
		return fmt.Errorf("walk inventory for %s: %w", inventoryName, err)
	}
	if !stringSlicesEqual(expected, actual) {
		return fmt.Errorf("inventory mismatch between %s and filesystem", inventoryName)
	}
	return nil
}

func filterNonEmpty(items []string) []string {
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
