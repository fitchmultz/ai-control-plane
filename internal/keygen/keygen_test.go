// keygen_test.go - Tests for key generation modules
//
// Purpose: Test parser and validator modules
//
// Responsibilities:
//   - Test argument parsing for all flag combinations
//   - Test alias and role validation
//   - Test role resolution logic
//
// Non-scope:
//   - Does not test actual key generation (requires LiteLLM)
//   - Does not test environment variable handling
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package keygen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		checkFunc func(*Config) bool
	}{
		{
			name:    "alias only",
			args:    []string{"my-key"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.Budget == 10.0 && c.Duration == "30d"
			},
		},
		{
			name:    "with budget",
			args:    []string{"my-key", "--budget", "5.00"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.Budget == 5.0
			},
		},
		{
			name:    "with rpm",
			args:    []string{"my-key", "--rpm", "100"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.RPM == 100
			},
		},
		{
			name:    "with tpm",
			args:    []string{"my-key", "--tpm", "1000"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.TPM == 1000
			},
		},
		{
			name:    "with parallel",
			args:    []string{"my-key", "--parallel", "10"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.Parallel == 10
			},
		},
		{
			name:    "with duration",
			args:    []string{"my-key", "--duration", "7d"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.Duration == "7d"
			},
		},
		{
			name:    "with role",
			args:    []string{"my-key", "--role", "admin"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.Role == "admin"
			},
		},
		{
			name:    "dry run",
			args:    []string{"my-key", "--dry-run"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" && c.DryRun == true
			},
		},
		{
			name:    "all options",
			args:    []string{"my-key", "--budget", "25.00", "--rpm", "200", "--tpm", "2000", "--parallel", "20", "--duration", "90d", "--role", "admin", "--dry-run"},
			wantErr: false,
			checkFunc: func(c *Config) bool {
				return c.Alias == "my-key" &&
					c.Budget == 25.0 &&
					c.RPM == 200 &&
					c.TPM == 2000 &&
					c.Parallel == 20 &&
					c.Duration == "90d" &&
					c.Role == "admin" &&
					c.DryRun == true
			},
		},
		{
			name:    "no alias",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "invalid budget",
			args:    []string{"my-key", "--budget", "invalid"},
			wantErr: true,
		},
		{
			name:    "invalid rpm",
			args:    []string{"my-key", "--rpm", "abc"},
			wantErr: true,
		},
		{
			name:    "unknown option",
			args:    []string{"my-key", "--unknown"},
			wantErr: true,
		},
		{
			name:    "missing budget value",
			args:    []string{"my-key", "--budget"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil && !tt.checkFunc(config) {
				t.Errorf("ParseArgs() config check failed: %+v", config)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{"valid simple", "mykey", false},
		{"valid with dash", "my-key", false},
		{"valid with underscore", "my_key", false},
		{"valid with dot", "my.key", false},
		{"valid mixed", "my-key_123.test", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 65)), true},
		{"with space", "my key", true},
		{"with slash", "my/key", true},
		{"with backslash", "my\\key", true},
		{"with special", "my@key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlias(%q) error = %v, wantErr %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"admin", "admin", false},
		{"team-lead", "team-lead", false},
		{"developer", "developer", false},
		{"auditor", "auditor", false},
		{"empty string", "", false},
		{"invalid", "invalid", true},
		{"ADMIN uppercase", "ADMIN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRole(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
			}
		})
	}
}

func TestGetModelsForRole(t *testing.T) {
	tests := []struct {
		role     string
		expected int
		wantErr  bool
	}{
		{"admin", 3, false},
		{"team-lead", 3, false},
		{"developer", 2, false},
		{"auditor", 0, false},
		{"unknown", 0, true},
		{"", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			models, err := GetModelsForRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetModelsForRole(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(models) != tt.expected {
				t.Errorf("GetModelsForRole(%q) returned %d models, want %d", tt.role, len(models), tt.expected)
			}
		})
	}
}

func TestResolveRole(t *testing.T) {
	// Save and restore env
	origRole := os.Getenv("ACP_USER_ROLE")
	defer os.Setenv("ACP_USER_ROLE", origRole)

	tests := []struct {
		name         string
		explicitRole string
		envRole      string
		expected     string
	}{
		{"explicit takes priority", "admin", "developer", "admin"},
		{"env variable used", "", "team-lead", "team-lead"},
		{"default when empty", "", "", "developer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ACP_USER_ROLE", tt.envRole)
			result, err := ResolveRole(tt.explicitRole)
			if err != nil {
				t.Fatalf("ResolveRole(%q) error = %v", tt.explicitRole, err)
			}
			if result != tt.expected {
				t.Errorf("ResolveRole(%q) with env=%q = %q, want %q",
					tt.explicitRole, tt.envRole, result, tt.expected)
			}
		})
	}
}

func TestCheckPrerequisites(t *testing.T) {
	// Save and restore env
	origKey := os.Getenv("LITELLM_MASTER_KEY")
	defer os.Setenv("LITELLM_MASTER_KEY", origKey)
	origRepoRoot := os.Getenv("ACP_REPO_ROOT")
	defer os.Setenv("ACP_REPO_ROOT", origRepoRoot)

	t.Run("master key required and present", func(t *testing.T) {
		os.Setenv("ACP_REPO_ROOT", t.TempDir())
		os.Setenv("LITELLM_MASTER_KEY", "test-key")
		err := CheckPrerequisites(true)
		if err != nil {
			t.Errorf("CheckPrerequisites(true) with key set error = %v", err)
		}
	})

	t.Run("master key required but missing", func(t *testing.T) {
		os.Setenv("ACP_REPO_ROOT", t.TempDir())
		os.Unsetenv("LITELLM_MASTER_KEY")
		err := CheckPrerequisites(true)
		if err == nil {
			t.Error("CheckPrerequisites(true) without key should error")
		}
	})

	t.Run("master key not required", func(t *testing.T) {
		os.Setenv("ACP_REPO_ROOT", t.TempDir())
		os.Unsetenv("LITELLM_MASTER_KEY")
		err := CheckPrerequisites(false)
		if err != nil {
			t.Errorf("CheckPrerequisites(false) without key error = %v", err)
		}
	})

	t.Run("master key repo fallback counts as configured", func(t *testing.T) {
		repoRoot := t.TempDir()
		envPath := filepath.Join(repoRoot, "demo", ".env")
		if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(envPath, []byte("LITELLM_MASTER_KEY=repo-key\n"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		origRepoRoot := os.Getenv("ACP_REPO_ROOT")
		defer os.Setenv("ACP_REPO_ROOT", origRepoRoot)
		os.Setenv("ACP_REPO_ROOT", repoRoot)
		os.Unsetenv("LITELLM_MASTER_KEY")

		err := CheckPrerequisites(true)
		if err != nil {
			t.Fatalf("CheckPrerequisites(true) with repo fallback error = %v", err)
		}
	})
}

func TestValidRoles(t *testing.T) {
	roles := ValidRoles()
	expected := []string{"admin", "auditor", "developer", "team-lead"}
	if len(roles) != len(expected) {
		t.Errorf("ValidRoles() returned %d roles, want %d", len(roles), len(expected))
	}
	for i, r := range expected {
		if roles[i] != r {
			t.Errorf("ValidRoles()[%d] = %q, want %q", i, roles[i], r)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.Budget != 10.00 {
		t.Errorf("DefaultConfig().Budget = %v, want 10.00", config.Budget)
	}
	if config.Duration != "30d" {
		t.Errorf("DefaultConfig().Duration = %q, want '30d'", config.Duration)
	}
	if config.DryRun != false {
		t.Errorf("DefaultConfig().DryRun = %v, want false", config.DryRun)
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "test", Message: "error message"}
	expected := "test: error message"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}
