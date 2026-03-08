// bundle_test.go - Tests for release bundle modules.
//
// Purpose: Test parser, planner, builder, and verifier modules
//
// Responsibilities:
//   - Keep this file's behavior focused and deterministic.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid semver", "v1.0.0", false},
		{"valid git sha", "abc1234", false},
		{"valid with dots", "2026.02.23", false},
		{"valid with underscore", "v1_0_0", false},
		{"valid with hyphen", "v1-0-0", false},
		{"empty version", "", true},
		{"with slash", "v1/0", true},
		{"with space", "v1 0", true},
		{"with backslash", "v1\\0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	defaultVersionFn := func(string) string { return "dev" }

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		checkFunc func(*Config) bool
	}{
		{
			name:    "build command",
			args:    []string{"build"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Command == "build" && c.Version == "dev"
			},
		},
		{
			name:    "verify command",
			args:    []string{"verify"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Command == "verify"
			},
		},
		{
			name:    "with version",
			args:    []string{"build", "--version", "v1.0.0"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Version == "v1.0.0"
			},
		},
		{
			name:    "with output dir",
			args:    []string{"build", "--output-dir", "/tmp/bundles"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.OutputDir == "/tmp/bundles"
			},
		},
		{
			name:    "with bundle",
			args:    []string{"verify", "--bundle", "/tmp/bundle.tar.gz"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Bundle == "/tmp/bundle.tar.gz"
			},
		},
		{
			name:    "verbose flag",
			args:    []string{"build", "--verbose"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Verbose == true
			},
		},
		{
			name:    "missing version value",
			args:    []string{"build", "--version"},
			wantErr: true,
		},
		{
			name:    "unknown option",
			args:    []string{"build", "--unknown"},
			wantErr: true,
		},
		{
			name:    "no command",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args, "/tmp/repo", defaultVersionFn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				if !tt.checkFunc(config) {
					t.Errorf("ParseArgs() config check failed: %+v", config)
				}
			}
		})
	}
}

func TestCreatePlan(t *testing.T) {
	config := &Config{
		Version:   "v1.0.0",
		OutputDir: "bundles",
	}

	plan, err := CreatePlan(config, "/repo")
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	if plan.Version != "v1.0.0" {
		t.Errorf("plan.Version = %q, want %q", plan.Version, "v1.0.0")
	}

	if plan.RepoRoot != "/repo" {
		t.Errorf("plan.RepoRoot = %q, want %q", plan.RepoRoot, "/repo")
	}

	expectedBundle := filepath.Join("/repo", "bundles", "ai-control-plane-deploy-v1.0.0.tar.gz")
	if plan.BundlePath != expectedBundle {
		t.Errorf("plan.BundlePath = %q, want %q", plan.BundlePath, expectedBundle)
	}
}

func TestValidateSourceFiles(t *testing.T) {
	// Create temp directory with some canonical files
	tmpDir, err := os.MkdirTemp("", "release-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create required directory structure and files
	for _, path := range []string{"Makefile", "README.md", "demo/docker-compose.yml"} {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	// Temporarily replace CanonicalPaths
	originalPaths := CanonicalPaths
	CanonicalPaths = []string{"Makefile", "README.md", "demo/docker-compose.yml"}
	defer func() { CanonicalPaths = originalPaths }()

	found, err := ValidateSourceFiles(tmpDir, false)
	if err != nil {
		t.Errorf("ValidateSourceFiles() error = %v", err)
	}
	if len(found) != 3 {
		t.Errorf("ValidateSourceFiles() found %d files, want 3", len(found))
	}
}

func TestValidateSourceFiles_Missing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "release-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Temporarily replace CanonicalPaths with non-existent files
	originalPaths := CanonicalPaths
	CanonicalPaths = []string{"non-existent-file.txt"}
	defer func() { CanonicalPaths = originalPaths }()

	_, err = ValidateSourceFiles(tmpDir, false)
	if err == nil {
		t.Error("ValidateSourceFiles() should error for missing files")
	}
}

func TestGetBundleName(t *testing.T) {
	name := GetBundleName("v1.0.0")
	expected := "ai-control-plane-deploy-v1.0.0.tar.gz"
	if name != expected {
		t.Errorf("GetBundleName() = %q, want %q", name, expected)
	}
}

func TestComputeFileHash(t *testing.T) {
	// Create temp file with known content
	tmpDir, err := os.MkdirTemp("", "hash-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, err := ComputeFileHash(testFile)
	if err != nil {
		t.Errorf("ComputeFileHash() error = %v", err)
	}

	// SHA256 of "hello world"
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("ComputeFileHash() = %q, want %q", hash, expected)
	}
}

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := HumanReadableSize(tt.bytes)
			if got != tt.want {
				t.Errorf("HumanReadableSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestGetDefaultVersion(t *testing.T) {
	// Create temp directory without git
	tmpDir, err := os.MkdirTemp("", "version-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Without git, should return "dev"
	version := GetDefaultVersion(tmpDir)
	if version != "dev" {
		t.Errorf("GetDefaultVersion() without git = %q, want 'dev'", version)
	}
}

func TestBuilder_Build(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "build-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create minimal canonical files
	for _, path := range []string{"Makefile", "README.md"} {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Temporarily replace CanonicalPaths
	originalPaths := CanonicalPaths
	CanonicalPaths = []string{"Makefile", "README.md"}
	defer func() { CanonicalPaths = originalPaths }()

	plan := &Plan{
		Version:    "v1.0.0",
		RepoRoot:   tmpDir,
		OutputDir:  filepath.Join(tmpDir, "output"),
		BundlePath: filepath.Join(tmpDir, "output", "test.tar.gz"),
	}

	builder := NewBuilder(tmpDir, false)
	var output strings.Builder
	err = builder.Build(plan, &output)
	if err != nil {
		t.Errorf("Builder.Build() error = %v", err)
	}

	// Verify bundle was created
	if _, err := os.Stat(plan.BundlePath); err != nil {
		t.Errorf("Bundle file not created: %v", err)
	}

	// Verify sidecar was created
	sidecarPath := plan.BundlePath + ".sha256"
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Errorf("Sidecar file not created: %v", err)
	}
}

func TestVerifier_Verify(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "verify-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create minimal canonical files
	for _, path := range []string{"Makefile", "README.md"} {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	// Temporarily replace CanonicalPaths
	originalPaths := CanonicalPaths
	CanonicalPaths = []string{"Makefile", "README.md"}
	defer func() { CanonicalPaths = originalPaths }()

	// Build a bundle first
	plan := &Plan{
		Version:    "v1.0.0",
		RepoRoot:   tmpDir,
		OutputDir:  filepath.Join(tmpDir, "output"),
		BundlePath: filepath.Join(tmpDir, "output", "test.tar.gz"),
	}

	builder := NewBuilder(tmpDir, false)
	var buildOutput strings.Builder
	if err := builder.Build(plan, &buildOutput); err != nil {
		t.Fatalf("Failed to build bundle: %v", err)
	}

	// Now verify it
	verifier := NewVerifier(false)
	var verifyOutput strings.Builder
	result, err := verifier.Verify(plan.BundlePath, &verifyOutput)
	if err != nil {
		t.Errorf("Verifier.Verify() error = %v", err)
	}

	if result == nil {
		t.Fatal("Verifier.Verify() returned nil result")
	}

	if !result.SidecarValid {
		t.Error("Expected sidecar to be valid")
	}

	if !result.StructureValid {
		t.Error("Expected structure to be valid")
	}

	if !result.PayloadValid {
		t.Error("Expected payload to be valid")
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected errors: %v", result.Errors)
	}
}

func TestVerifier_Verify_MissingBundle(t *testing.T) {
	verifier := NewVerifier(false)
	var output strings.Builder
	_, err := verifier.Verify("/nonexistent/bundle.tar.gz", &output)
	if err == nil {
		t.Error("Verifier.Verify() should error for missing bundle")
	}
}

func TestVerifier_Verify_MissingSidecar(t *testing.T) {
	// Create temp file without sidecar
	tmpDir, err := os.MkdirTemp("", "verify-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bundlePath := filepath.Join(tmpDir, "bundle.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("fake bundle"), 0644); err != nil {
		t.Fatalf("failed to write bundle: %v", err)
	}

	verifier := NewVerifier(false)
	var output strings.Builder
	_, err = verifier.Verify(bundlePath, &output)
	if err == nil {
		t.Error("Verifier.Verify() should error for missing sidecar")
	}
}

func TestCollectRegularFilesPropagatesWalkErrors(t *testing.T) {
	_, err := collectRegularFiles(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected collectRegularFiles to propagate walk error")
	}
}
