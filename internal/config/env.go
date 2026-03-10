// Package config centralizes runtime configuration loading for ACP processes.
//
// Purpose:
//   - Own typed env-file access and required-runtime env resolution.
//
// Responsibilities:
//   - Read explicit env files through the shared strict parser.
//   - Resolve repo-local demo/.env metadata from the canonical repo root.
//   - Report required key presence using process-env precedence plus repo fallback.
//
// Scope:
//   - Env-file inspection and presence reporting only.
//
// Usage:
//   - Use `NewEnvFile(path)` for explicit file inspection and `Loader.RequiredRuntimeEnv`
//   - or `Loader.RepoEnvStatus` for repo-aware status.
//
// Invariants/Assumptions:
//   - Process environment takes precedence over repo-local demo/.env values.
//   - Callers outside internal/config never touch envfile parsing directly.
package config

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/mitchfultz/ai-control-plane/internal/envfile"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

// EnvFile provides strict data-only access to a specific env file.
type EnvFile struct {
	path string
}

// EnvFileStatus reports repo-local env file metadata.
type EnvFileStatus struct {
	Path   string
	Exists bool
}

// RequiredEnvStatus reports required key presence with source attribution.
type RequiredEnvStatus struct {
	Found   []string
	Missing []string
	Sources map[string]string
}

// NewEnvFile creates a strict env-file reader for an explicit path.
func NewEnvFile(path string) EnvFile {
	return EnvFile{path: filepath.Clean(textutil.Trim(path))}
}

// Path returns the normalized file path.
func (f EnvFile) Path() string {
	return f.path
}

// Lookup reads a single key from the env file.
func (f EnvFile) Lookup(key string) (string, bool, error) {
	value, ok, err := envfile.LookupFile(f.path, key)
	return textutil.Trim(value), ok, err
}

// RepoEnvStatus reports whether the canonical repo-local demo/.env exists.
func (l *Loader) RepoEnvStatus(ctx context.Context) (EnvFileStatus, error) {
	repoRoot, err := l.RequireRepoRoot(ctx)
	if err != nil {
		return EnvFileStatus{}, err
	}
	path := repopath.DemoEnvPath(repoRoot)
	_, statErr := os.Stat(path)
	return EnvFileStatus{
		Path:   path,
		Exists: statErr == nil,
	}, nil
}

// RequiredRuntimeEnv reports required env-key presence with repo fallback.
func (l *Loader) RequiredRuntimeEnv(keys []string) RequiredEnvStatus {
	status := RequiredEnvStatus{
		Found:   make([]string, 0, len(keys)),
		Missing: make([]string, 0),
		Sources: make(map[string]string, len(keys)),
	}
	for _, key := range keys {
		trimmedKey := textutil.Trim(key)
		if trimmedKey == "" {
			continue
		}
		if value := l.String(trimmedKey); value != "" {
			status.Found = append(status.Found, trimmedKey)
			status.Sources[trimmedKey] = "process"
			continue
		}
		if value := l.RepoAwareString(trimmedKey); value != "" {
			status.Found = append(status.Found, trimmedKey)
			status.Sources[trimmedKey] = "repo"
			continue
		}
		status.Missing = append(status.Missing, trimmedKey)
	}
	sort.Strings(status.Found)
	sort.Strings(status.Missing)
	return status
}
