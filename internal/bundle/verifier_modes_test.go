// verifier_modes_test.go - Permission-focused coverage for verifier extraction.
//
// Purpose:
//   - Verify bundle verification extraction clamps local artifacts to private modes.
//
// Responsibilities:
//   - Assert extracted directories use private permissions.
//   - Assert extracted files use private permissions regardless of tar metadata.
//
// Scope:
//   - Verifier extraction filesystem behavior only.
//
// Usage:
//   - Run via `go test ./internal/bundle`.
//
// Invariants/Assumptions:
//   - Tests run on filesystems that report owner/group/other mode bits.
package bundle

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

func TestVerifierExtractTarballUsesPrivateModes(t *testing.T) {
	t.Parallel()

	tarballPath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	writeTarball(t, tarballPath, []tarEntry{
		{name: "payload/", mode: 0o777, typ: tar.TypeDir},
		{name: "payload/tool.sh", mode: 0o755, typ: tar.TypeReg, body: "#!/bin/sh\necho ok\n"},
	})

	destDir := filepath.Join(t.TempDir(), "extract")
	if err := NewVerifier(false).extractTarball(tarballPath, destDir); err != nil {
		t.Fatalf("extractTarball() error = %v", err)
	}

	assertPerm(t, filepath.Join(destDir, "payload"), fsutil.PrivateDirPerm)
	filePath := filepath.Join(destDir, "payload", "tool.sh")
	assertPerm(t, filePath, fsutil.PrivateFilePerm)

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", filePath, err)
	}
	if got, want := string(data), "#!/bin/sh\necho ok\n"; got != want {
		t.Fatalf("extracted content = %q, want %q", got, want)
	}
}

type tarEntry struct {
	name string
	body string
	mode int64
	typ  byte
}

func writeTarball(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%s) error = %v", path, err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     entry.mode,
			Typeflag: entry.typ,
			Size:     int64(len(entry.body)),
		}
		if entry.typ == tar.TypeDir {
			header.Size = 0
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader(%s) error = %v", entry.name, err)
		}
		if entry.typ == tar.TypeReg {
			if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
				t.Fatalf("Write(%s) error = %v", entry.name, err)
			}
		}
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %04o, want %04o", path, got, want)
	}
}
