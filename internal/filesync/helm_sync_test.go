// helm_sync_test validates typed Helm file synchronization behavior.
//
// Purpose:
//
//	Ensure mapping fidelity and file-copy behavior match legacy shell sync
//	semantics for Helm chart file mirroring.
//
// Responsibilities:
//   - Verify HelmMappings remain exactly aligned with expected paths.
//   - Verify mapped files are copied and destination directories are created.
//   - Verify missing source files return a deterministic failure.
//
// Non-scope:
//   - Does not execute the acpctl CLI wrapper.
//
// Invariants/Assumptions:
//   - Tests run on temporary filesystem state only.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package filesync

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestHelmMappingsExact(t *testing.T) {
	t.Parallel()

	expected := []Mapping{
		{Source: "demo/scripts/chargeback_report.sh", Destination: "deploy/helm/ai-control-plane/files/chargeback_report.sh"},
		{Source: "demo/scripts/lib/chargeback_db.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_db.sh"},
		{Source: "demo/scripts/lib/chargeback_analysis.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_analysis.sh"},
		{Source: "demo/scripts/lib/chargeback_render.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_render.sh"},
		{Source: "demo/scripts/lib/chargeback_io.sh", Destination: "deploy/helm/ai-control-plane/files/lib/chargeback_io.sh"},
	}

	if !reflect.DeepEqual(expected, HelmMappings) {
		t.Fatalf("HelmMappings drifted from contract\nexpected: %#v\nactual:   %#v", expected, HelmMappings)
	}
}

func TestSyncHelmFilesCopiesMappings(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	for _, mapping := range HelmMappings {
		srcPath := filepath.Join(repoRoot, filepath.FromSlash(mapping.Source))
		mustWriteExecutableFile(t, srcPath, []byte("content for "+mapping.Source+"\n"))
	}

	var out bytes.Buffer
	if err := SyncHelmFiles(SyncOptions{RepoRoot: repoRoot, Writer: &out}); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	if !strings.Contains(out.String(), "Synchronizing Helm chart files...") {
		t.Fatalf("expected sync output header, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "Synchronization complete") {
		t.Fatalf("expected sync completion output, got: %q", out.String())
	}

	for _, mapping := range HelmMappings {
		srcPath := filepath.Join(repoRoot, filepath.FromSlash(mapping.Source))
		dstPath := filepath.Join(repoRoot, filepath.FromSlash(mapping.Destination))

		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read source %s: %v", mapping.Source, err)
		}
		dstBytes, err := os.ReadFile(dstPath)
		if err != nil {
			t.Fatalf("read destination %s: %v", mapping.Destination, err)
		}
		if !bytes.Equal(srcBytes, dstBytes) {
			t.Fatalf("destination mismatch for %s", mapping.Destination)
		}
	}
}

func TestSyncHelmFilesMissingSourceFails(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	err := SyncHelmFiles(SyncOptions{RepoRoot: repoRoot})
	if err == nil {
		t.Fatal("expected missing source error")
	}
	if !strings.Contains(err.Error(), HelmMappings[0].Source) {
		t.Fatalf("expected error to mention first mapping source; got: %v", err)
	}
}

func TestSyncHelmFilesRequiresRepoRoot(t *testing.T) {
	t.Parallel()

	err := SyncHelmFiles(SyncOptions{RepoRoot: "  "})
	if err == nil {
		t.Fatal("expected repo root validation error")
	}
	if !strings.Contains(err.Error(), "repository root is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustWriteExecutableFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
