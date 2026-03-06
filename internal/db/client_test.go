// client_test.go - Tests for database client
//
// Purpose: Provide unit tests for database client functionality
//
// Responsibilities:
//   - Test mode detection logic (embedded vs external)
//   - Test external database connection handling
//   - Test environment variable parsing
//
// Non-scope:
//   - Does not test actual database connections (requires PostgreSQL)
//   - Does not test Docker operations (integration tests)
//
// Invariants/Assumptions:
//   - Tests use temporary directories and environment variable manipulation
//   - Tests restore original environment after completion
package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDatabaseMode_ExplicitEnvVar(t *testing.T) {
	// Save and restore original env vars
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
	}()

	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"explicit embedded", "embedded", "embedded"},
		{"explicit EMBEDDED", "EMBEDDED", "embedded"},
		{"explicit external", "external", "external"},
		{"explicit EXTERNAL", "EXTERNAL", "external"},
		{"empty string defaults to embedded", "", "embedded"},
		{"invalid value defaults to embedded", "invalid", "embedded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("DATABASE_URL")
			if tt.mode == "" {
				os.Unsetenv("ACP_DATABASE_MODE")
			} else {
				os.Setenv("ACP_DATABASE_MODE", tt.mode)
			}

			mode := detectDatabaseMode()
			if mode != tt.expected {
				t.Errorf("detectDatabaseMode() = %q, want %q", mode, tt.expected)
			}
		})
	}
}

func TestDetectDatabaseMode_DatabaseURLPresence(t *testing.T) {
	// Save and restore original env vars
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("ACP_REPO_ROOT", origRepoRoot)
	}()

	// Clear explicit mode to confirm DATABASE_URL alone does not flip modes.
	os.Unsetenv("ACP_DATABASE_MODE")

	tests := []struct {
		name     string
		dbURL    string
		expected string
	}{
		{"DATABASE_URL present still defaults to embedded", "postgres://user:pass@localhost/db", "embedded"},
		{"DATABASE_URL empty", "", "embedded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("ACP_REPO_ROOT")
			if tt.dbURL == "" {
				os.Unsetenv("DATABASE_URL")
			} else {
				os.Setenv("DATABASE_URL", tt.dbURL)
			}

			mode := detectDatabaseMode()
			if mode != tt.expected {
				t.Errorf("detectDatabaseMode() = %q, want %q", mode, tt.expected)
			}
		})
	}
}

func TestDetectDatabaseMode_EnvFile(t *testing.T) {
	// Save and restore original env vars
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	origWD, _ := os.Getwd()
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("ACP_REPO_ROOT", origRepoRoot)
		os.Chdir(origWD)
	}()

	// Clear env vars to test .env file detection
	os.Unsetenv("ACP_DATABASE_MODE")
	os.Unsetenv("DATABASE_URL")

	// Create temp directory with demo/.env
	tmpDir, err := os.MkdirTemp("", "db_client_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	demoDir := filepath.Join(tmpDir, "demo")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}

	tests := []struct {
		name       string
		envContent string
		expected   string
	}{
		{
			name:       "explicit external mode in .env",
			envContent: "ACP_DATABASE_MODE=external\n",
			expected:   "external",
		},
		{
			name:       "explicit EXTERNAL mode in .env (case insensitive)",
			envContent: "ACP_DATABASE_MODE=EXTERNAL\n",
			expected:   "external",
		},
		{
			name:       "DATABASE_URL in .env still defaults to embedded",
			envContent: "DATABASE_URL=postgres://user:pass@localhost/db\n",
			expected:   "embedded",
		},
		{
			name:       "embedded mode in .env",
			envContent: "ACP_DATABASE_MODE=embedded\n",
			expected:   "embedded",
		},
		{
			name:       "empty .env defaults to embedded",
			envContent: "",
			expected:   "embedded",
		},
		{
			name:       "DATABASE_URL without value in .env",
			envContent: "DATABASE_URL=\n",
			expected:   "embedded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write .env file
			envFile := filepath.Join(demoDir, ".env")
			if err := os.WriteFile(envFile, []byte(tt.envContent), 0644); err != nil {
				t.Fatalf("failed to write .env file: %v", err)
			}

			// Set ACP_REPO_ROOT
			os.Setenv("ACP_REPO_ROOT", tmpDir)

			mode := detectDatabaseMode()
			if mode != tt.expected {
				t.Errorf("detectDatabaseMode() = %q, want %q for env content:\n%s", mode, tt.expected, tt.envContent)
			}
		})
	}
}

func TestDetectDatabaseMode_Priority(t *testing.T) {
	// Save and restore original env vars
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
	}()

	// ACP_DATABASE_MODE should take priority over DATABASE_URL
	os.Setenv("ACP_DATABASE_MODE", "embedded")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")

	mode := detectDatabaseMode()
	if mode != "embedded" {
		t.Errorf("ACP_DATABASE_MODE should take priority over DATABASE_URL: got %q, want %q", mode, "embedded")
	}
}

func TestNewClient_ModeDetection(t *testing.T) {
	// Save and restore original env vars
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
	}()

	// Test embedded mode
	os.Setenv("ACP_DATABASE_MODE", "embedded")
	os.Unsetenv("DATABASE_URL")
	client := NewClient(nil)
	if !client.IsEmbedded() {
		t.Error("Expected embedded mode")
	}
	if client.IsExternal() {
		t.Error("Should not be external mode")
	}

	// Test external mode
	os.Setenv("ACP_DATABASE_MODE", "external")
	client = NewClient(nil)
	if !client.IsExternal() {
		t.Error("Expected external mode")
	}
	if client.IsEmbedded() {
		t.Error("Should not be embedded mode")
	}
}

func TestNewClient_ConfigErrorOnAmbiguousDatabaseURL(t *testing.T) {
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("ACP_REPO_ROOT", origRepoRoot)
	}()

	tmpDir, err := os.MkdirTemp("", "db_client_ambiguous")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "demo"), 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "demo", ".env"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	os.Setenv("ACP_REPO_ROOT", tmpDir)
	os.Unsetenv("ACP_DATABASE_MODE")
	os.Setenv("DATABASE_URL", "postgresql://custom-user:custom-pass@db.example.com:5432/litellm")

	client := NewClient(nil)
	if client.ConfigError() == nil {
		t.Fatal("expected configuration error for custom DATABASE_URL without ACP_DATABASE_MODE")
	}
	if _, err := client.Query(t.Context(), "SELECT 1"); err == nil {
		t.Fatal("expected Query to fail when configuration is ambiguous")
	}
}

func TestNewClient_DefaultEmbeddedURLRemainsValidWithoutExplicitMode(t *testing.T) {
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("ACP_REPO_ROOT", origRepoRoot)
	}()

	tmpDir, err := os.MkdirTemp("", "db_client_default")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "demo"), 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "demo", ".env"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	os.Setenv("ACP_REPO_ROOT", tmpDir)
	os.Unsetenv("ACP_DATABASE_MODE")
	os.Setenv("DATABASE_URL", defaultEmbeddedDatabaseURL)

	client := NewClient(nil)
	if client.ConfigError() != nil {
		t.Fatalf("expected no configuration error for embedded default DATABASE_URL, got %v", client.ConfigError())
	}
}

func TestNewClient_ConfigErrorOnAmbiguousDatabaseURLInRepoEnv(t *testing.T) {
	origMode := os.Getenv("ACP_DATABASE_MODE")
	origDBURL := os.Getenv("DATABASE_URL")
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	defer func() {
		os.Setenv("ACP_DATABASE_MODE", origMode)
		os.Setenv("DATABASE_URL", origDBURL)
		os.Setenv("ACP_REPO_ROOT", origRepoRoot)
	}()

	tmpDir, err := os.MkdirTemp("", "db_client_repo_env")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "demo"), 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "demo", ".env"), []byte("DATABASE_URL=postgresql://custom-user:custom-pass@db.example.com:5432/litellm\n"), 0644); err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	os.Setenv("ACP_REPO_ROOT", tmpDir)
	os.Unsetenv("ACP_DATABASE_MODE")
	os.Unsetenv("DATABASE_URL")

	client := NewClient(nil)
	if client.ConfigError() == nil {
		t.Fatal("expected configuration error for repo .env DATABASE_URL without ACP_DATABASE_MODE")
	}
}

func TestClient_ModeMethods(t *testing.T) {
	client := &Client{mode: "embedded"}
	if !client.IsEmbedded() {
		t.Error("IsEmbedded() should return true for embedded mode")
	}
	if client.IsExternal() {
		t.Error("IsExternal() should return false for embedded mode")
	}
	if client.Mode() != "embedded" {
		t.Errorf("Mode() = %q, want %q", client.Mode(), "embedded")
	}

	client = &Client{mode: "external"}
	if !client.IsExternal() {
		t.Error("IsExternal() should return true for external mode")
	}
	if client.IsEmbedded() {
		t.Error("IsEmbedded() should return false for external mode")
	}
	if client.Mode() != "external" {
		t.Errorf("Mode() = %q, want %q", client.Mode(), "external")
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save and restore original env var
	origVal := os.Getenv("TEST_VAR_12345")
	defer os.Setenv("TEST_VAR_12345", origVal)

	// Test with unset variable
	os.Unsetenv("TEST_VAR_12345")
	result := getEnvOrDefault("TEST_VAR_12345", "default")
	if result != "default" {
		t.Errorf("getEnvOrDefault() = %q, want %q", result, "default")
	}

	// Test with set variable
	os.Setenv("TEST_VAR_12345", "custom")
	result = getEnvOrDefault("TEST_VAR_12345", "default")
	if result != "custom" {
		t.Errorf("getEnvOrDefault() = %q, want %q", result, "custom")
	}

	// Test with empty string (should return default)
	os.Setenv("TEST_VAR_12345", "")
	result = getEnvOrDefault("TEST_VAR_12345", "default")
	if result != "default" {
		t.Errorf("getEnvOrDefault() = %q, want %q (empty string should use default)", result, "default")
	}
}

func TestClient_Close(t *testing.T) {
	// Test Close with nil db (embedded mode)
	client := &Client{mode: "embedded", db: nil}
	if err := client.Close(); err != nil {
		t.Errorf("Close() with nil db should not error, got: %v", err)
	}

	// Note: Testing Close() with actual external connection would require
	// a real PostgreSQL instance, which is beyond unit test scope
}
