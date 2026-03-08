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
//
// Scope:
//   - Codex config ownership rules only.
//
// Usage:
//   - Called by Run when `--write-config` is enabled for Codex.
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
)

const codexManagedHeader = "# Managed by acpctl onboard. Manual edits outside this managed file will block future writes.\n"

var errUnmanagedCodexConfig = errors.New("existing ~/.codex/config.toml is not managed by acpctl")

func maybeWriteCodexConfig(state runState) error {
	if !state.Options.WriteConfig || state.Options.Tool != "codex" || state.Options.Mode == "direct" {
		return nil
	}
	configDir := filepath.Join(config.NewLoader().Tooling().HomeDir, ".codex")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create codex config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	content := renderManagedCodexConfig(state)
	if err := ensureCodexConfigOwnership(configPath, content); err != nil {
		if errors.Is(err, errUnmanagedCodexConfig) {
			return fmt.Errorf("%w; move it aside or convert it manually before using --write-config", err)
		}
		return err
	}
	if err := fsutil.AtomicWriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write codex config: %w", err)
	}
	fprintf(state.Options.Stdout, "INFO: Wrote %s\n", configPath)
	return nil
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

func renderManagedCodexConfig(state runState) string {
	return codexManagedHeader + fmt.Sprintf(`model = %q
model_provider = %q

[model_providers.litellm]
name = %q
base_url = %q
wire_api = %q
env_key = %q
`, state.Options.Model, "litellm", "LiteLLM", state.BaseURL+"/v1", "responses", "OPENAI_API_KEY")
}
