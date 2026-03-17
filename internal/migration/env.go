// Package migration provides typed config and database migration primitives.
//
// Purpose:
//   - Define explicit env-file mutation primitives for release-to-release
//   - upgrade edges.
//
// Responsibilities:
//   - Parse canonical KEY=VALUE env files as data only.
//   - Apply rename, set, remove, and required-key mutations deterministically.
//   - Persist migrated env files atomically with private permissions.
//
// Scope:
//   - Canonical secrets/env file mutation helpers only.
//
// Usage:
//   - Called by the typed upgrade workflow when a release edge declares config
//   - mutations.
//
// Invariants/Assumptions:
//   - Only explicit release-declared mutations are applied.
//   - Non-comment malformed env lines fail fast.
package migration

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
)

// EnvMutation describes one explicit env-file migration.
type EnvMutation struct {
	ID      string            `json:"id"`
	Summary string            `json:"summary"`
	Rename  map[string]string `json:"rename,omitempty"`
	Set     map[string]string `json:"set,omitempty"`
	Remove  []string          `json:"remove,omitempty"`
	Require []string          `json:"require,omitempty"`
}

// EnvMutationResult reports the outcome of one env-file migration.
type EnvMutationResult struct {
	Path        string            `json:"path"`
	Changed     bool              `json:"changed"`
	Values      map[string]string `json:"values"`
	RenamedKeys map[string]string `json:"renamed_keys,omitempty"`
	RemovedKeys []string          `json:"removed_keys,omitempty"`
}

// ApplyEnvMutation applies one explicit env mutation, optionally persisting it.
func ApplyEnvMutation(path string, mutation EnvMutation, write bool) (EnvMutationResult, error) {
	values, err := loadEnvMap(path)
	if err != nil {
		return EnvMutationResult{}, err
	}

	result := EnvMutationResult{
		Path:        path,
		Values:      cloneMap(values),
		RenamedKeys: map[string]string{},
	}

	for oldKey, newKey := range mutation.Rename {
		value, ok := result.Values[oldKey]
		if !ok {
			continue
		}
		delete(result.Values, oldKey)
		result.Values[newKey] = value
		result.RenamedKeys[oldKey] = newKey
		result.Changed = true
	}

	for _, key := range mutation.Remove {
		if _, ok := result.Values[key]; ok {
			delete(result.Values, key)
			result.RemovedKeys = append(result.RemovedKeys, key)
			result.Changed = true
		}
	}
	if len(result.RemovedKeys) > 0 {
		sort.Strings(result.RemovedKeys)
	}

	for key, value := range mutation.Set {
		if current, ok := result.Values[key]; !ok || current != value {
			result.Values[key] = value
			result.Changed = true
		}
	}

	for _, key := range mutation.Require {
		if strings.TrimSpace(result.Values[key]) == "" {
			return EnvMutationResult{}, fmt.Errorf("migration %s requires non-empty key %s", mutation.ID, key)
		}
	}

	if write && result.Changed {
		if err := fsutil.AtomicWritePrivateFile(path, renderEnvMap(result.Values)); err != nil {
			return EnvMutationResult{}, fmt.Errorf("write migrated env file: %w", err)
		}
	}

	return result, nil
}

func loadEnvMap(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}

	values := make(map[string]string)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env line %q in %s", line, path)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("blank env key in %s", path)
		}
		values[key] = strings.TrimSpace(value)
	}
	return values, nil
}

func renderEnvMap(values map[string]string) []byte {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(values[key])
		builder.WriteString("\n")
	}
	return []byte(builder.String())
}

func cloneMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
