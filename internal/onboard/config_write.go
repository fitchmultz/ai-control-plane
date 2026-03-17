// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Manage ACP-owned local Codex configuration writes without clobbering
//	unmanaged user configuration.
//
// Responsibilities:
//   - Render ACP-managed Codex config content.
//   - Detect conflicts with unmanaged user config files.
//   - Persist managed config atomically with private permissions.
//   - Lint the resulting config for syntax, contract, and permission validity.
//
// Scope:
//   - Codex config ownership and linting rules only.
//
// Usage:
//   - Called by Run when the wizard is allowed to write managed Codex config.
//
// Invariants/Assumptions:
//   - acpctl only overwrites configs it already manages.
//   - Newly created config directories and files are private to the user.
package onboard

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
	toml "github.com/pelletier/go-toml/v2"
)

const codexManagedHeader = "# Managed by acpctl onboard. Manual edits outside this managed file will block future writes.\n"

var errUnmanagedCodexConfig = errors.New("existing ~/.codex/config.toml is not managed by acpctl")

type codexConfigFile struct {
	Model          string `toml:"model"`
	ModelProvider  string `toml:"model_provider"`
	ModelProviders map[string]struct {
		Name    string `toml:"name"`
		BaseURL string `toml:"base_url"`
		WireAPI string `toml:"wire_api"`
		EnvKey  string `toml:"env_key"`
	} `toml:"model_providers"`
}

func maybeWriteCodexConfig(state runState) (ToolConfigResult, error) {
	if state.Options.Tool != "codex" || state.Options.Mode == "direct" {
		return ToolConfigResult{
			Tool:    state.Options.Tool,
			Skipped: true,
			Summary: "no ACP-managed tool config is required for this tool or mode",
		}, nil
	}
	if !state.Options.WriteConfig {
		return ToolConfigResult{
			Tool:    "codex",
			Skipped: true,
			Summary: "ACP-managed ~/.codex/config.toml write disabled by operator",
		}, nil
	}

	homeDir := strings.TrimSpace(config.NewLoader().Tooling().HomeDir)
	if homeDir == "" {
		return ToolConfigResult{
			Tool:    "codex",
			Summary: "Codex config path could not be resolved",
			Issues: []string{
				"HOME is empty, so acpctl could not determine ~/.codex/config.toml; set HOME and rerun `acpctl onboard codex`",
			},
		}, nil
	}

	configDir := filepath.Join(homeDir, ".codex")
	if err := fsutil.EnsurePrivateDir(configDir); err != nil {
		return ToolConfigResult{}, fmt.Errorf("create codex config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	content := renderManagedCodexConfig(state)
	contentIssues := validateCodexConfigContent([]byte(content), state)
	if len(contentIssues) > 0 {
		return ToolConfigResult{
			Tool:    "codex",
			Path:    configPath,
			Summary: "ACP-managed Codex config contract is invalid",
			Issues:  contentIssues,
		}, nil
	}

	if err := ensureCodexConfigOwnership(configPath, content); err != nil {
		if errors.Is(err, errUnmanagedCodexConfig) {
			return ToolConfigResult{
				Tool:    "codex",
				Path:    configPath,
				Summary: "ACP-managed Codex config write blocked by unmanaged file",
				Issues: []string{
					fmt.Sprintf("%s already exists and is not ACP-managed; move it aside or merge the ACP settings manually, then rerun `acpctl onboard codex`", configPath),
				},
			}, nil
		}
		return ToolConfigResult{}, err
	}

	current, currentErr := os.ReadFile(configPath)
	written := false
	if currentErr != nil && !errors.Is(currentErr, os.ErrNotExist) {
		return ToolConfigResult{}, fmt.Errorf("read current codex config: %w", currentErr)
	}
	if errors.Is(currentErr, os.ErrNotExist) || !bytes.Equal(current, []byte(content)) {
		if err := fsutil.AtomicWritePrivateFile(configPath, []byte(content)); err != nil {
			return ToolConfigResult{}, fmt.Errorf("write codex config: %w", err)
		}
		written = true
	}

	issues, err := validateCodexConfigFile(configPath, state)
	if err != nil {
		return ToolConfigResult{}, err
	}

	summary := fmt.Sprintf("ACP-managed Codex config is valid and loadable: %s", configPath)
	if written {
		summary = fmt.Sprintf("wrote valid ACP-managed Codex config: %s", configPath)
	}
	if len(issues) > 0 {
		summary = fmt.Sprintf("ACP-managed Codex config has issues: %s", configPath)
	}

	return ToolConfigResult{
		Tool:    "codex",
		Path:    configPath,
		Written: written,
		Summary: summary,
		Issues:  issues,
	}, nil
}

func ensureCodexConfigOwnership(path string, desired string) error {
	current, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read existing codex config: %w", err)
	}
	if bytes.Equal(current, []byte(desired)) {
		return nil
	}
	if strings.HasPrefix(string(current), codexManagedHeader) {
		return nil
	}
	return errUnmanagedCodexConfig
}

func validateCodexConfigFile(path string, state runState) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read codex config: %w", err)
	}

	issues := validation.NewIssues()
	issues.Extend(validateCodexConfigContent(data, state))

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat codex config: %w", err)
	}
	if info.Mode().Perm() != fsutil.PrivateFilePerm {
		issues.Addf("%s must use %04o permissions (found %04o); run `chmod 600 %s`", path, fsutil.PrivateFilePerm, info.Mode().Perm(), path)
	}

	dirPath := filepath.Dir(path)
	dirInfo, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("stat codex config directory: %w", err)
	}
	if dirInfo.Mode().Perm() != fsutil.PrivateDirPerm {
		issues.Addf("%s must use %04o permissions (found %04o); run `chmod 700 %s`", dirPath, fsutil.PrivateDirPerm, dirInfo.Mode().Perm(), dirPath)
	}

	return issues.Sorted(), nil
}

func validateCodexConfigContent(data []byte, state runState) []string {
	var cfg codexConfigFile
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return []string{fmt.Sprintf("Codex config is not valid TOML: %v", err)}
	}

	issues := validation.NewIssues()
	expectedBaseURL := state.Gateway.BaseURL + "/v1"

	if strings.TrimSpace(cfg.Model) != state.Options.Model {
		issues.Addf("Codex config model must be %q (found %q)", state.Options.Model, cfg.Model)
	}
	if strings.TrimSpace(cfg.ModelProvider) != "litellm" {
		issues.Addf("Codex config model_provider must be %q (found %q)", "litellm", cfg.ModelProvider)
	}

	provider, ok := cfg.ModelProviders["litellm"]
	if !ok {
		issues.Add(`Codex config must define [model_providers.litellm]`)
		return issues.Sorted()
	}

	if strings.TrimSpace(provider.Name) != "LiteLLM" {
		issues.Addf("Codex config provider name must be %q (found %q)", "LiteLLM", provider.Name)
	}
	if strings.TrimSpace(provider.BaseURL) != expectedBaseURL {
		issues.Addf("Codex config provider base_url must be %q (found %q)", expectedBaseURL, provider.BaseURL)
	}
	if strings.TrimSpace(provider.WireAPI) != "responses" {
		issues.Addf("Codex config provider wire_api must be %q (found %q)", "responses", provider.WireAPI)
	}
	if strings.TrimSpace(provider.EnvKey) != "OPENAI_API_KEY" {
		issues.Addf("Codex config provider env_key must be %q (found %q)", "OPENAI_API_KEY", provider.EnvKey)
	}

	return issues.Sorted()
}

func renderManagedCodexConfig(state runState) string {
	return codexManagedHeader + fmt.Sprintf(`model = %q
model_provider = %q

[model_providers.litellm]
name = %q
base_url = %q
wire_api = %q
env_key = %q
`, state.Options.Model, "litellm", "LiteLLM", state.Gateway.BaseURL+"/v1", "responses", "OPENAI_API_KEY")
}
