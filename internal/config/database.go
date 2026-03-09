// Package config centralizes runtime configuration loading for ACP processes.
//
// Purpose:
//   - Expose typed database and repository-aware configuration.
//
// Responsibilities:
//   - Normalize database mode resolution and ambiguity detection.
//   - Provide repo-aware access to required database env keys.
//   - Keep repo-root and `.env` fallback rules out of callers.
//
// Scope:
//   - Database/runtime config resolution only.
//
// Usage:
//   - Call `Loader.Database(ctx)` before constructing database-facing helpers.
//
// Invariants/Assumptions:
//   - `ACP_DATABASE_MODE` is the only explicit switch between embedded/external modes.
//   - `DATABASE_URL` alone must not silently change database mode.
package config

import (
	"context"
	"fmt"
	"strings"

	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

const defaultEmbeddedDatabaseURL = "postgresql://litellm:litellm@postgres:5432/litellm"

// DatabaseMode identifies the effective database runtime mode.
type DatabaseMode string

const (
	DatabaseModeEmbedded DatabaseMode = "embedded"
	DatabaseModeExternal DatabaseMode = "external"
)

// String returns the serialized database mode value.
func (m DatabaseMode) String() string {
	return string(m)
}

// IsEmbedded reports whether the mode uses the repo-local PostgreSQL container.
func (m DatabaseMode) IsEmbedded() bool {
	return m == DatabaseModeEmbedded
}

// IsExternal reports whether the mode uses an external PostgreSQL instance.
func (m DatabaseMode) IsExternal() bool {
	return m == DatabaseModeExternal
}

// DatabaseSettings captures typed database runtime configuration.
type DatabaseSettings struct {
	Mode         DatabaseMode
	Name         string
	User         string
	URL          string
	RepoRoot     string
	RepoEnvPath  string
	AmbiguousErr error
}

// Database returns the effective typed database settings.
func (l *Loader) Database(ctx context.Context) DatabaseSettings {
	repoRoot, repoRootErr := l.RepoRoot(ctx)
	settings := DatabaseSettings{
		Mode:     DatabaseModeEmbedded,
		Name:     l.StringDefault("DB_NAME", "litellm"),
		User:     l.StringDefault("DB_USER", "litellm"),
		URL:      l.RepoAwareString("DATABASE_URL"),
		RepoRoot: repoRoot,
	}
	if repoRootErr == nil && strings.TrimSpace(repoRoot) != "" {
		settings.RepoEnvPath = repopath.DemoEnvPath(repoRoot)
	}
	if mode, ok := normalizeDatabaseMode(l.String("ACP_DATABASE_MODE")); ok {
		settings.Mode = mode
	} else if mode, ok := normalizeDatabaseMode(l.RepoAwareString("ACP_DATABASE_MODE")); ok {
		settings.Mode = mode
	}
	if settings.Mode.IsEmbedded() && strings.TrimSpace(settings.URL) != "" && settings.URL != defaultEmbeddedDatabaseURL {
		explicitMode := l.String("ACP_DATABASE_MODE")
		if explicitMode == "" {
			explicitMode = l.RepoAwareString("ACP_DATABASE_MODE")
		}
		if explicitMode == "" {
			settings.AmbiguousErr = fmt.Errorf("ambiguous database configuration: DATABASE_URL is set but ACP_DATABASE_MODE is not; set ACP_DATABASE_MODE=external for external PostgreSQL or ACP_DATABASE_MODE=embedded for the local demo stack")
		}
	}
	return settings
}

func normalizeDatabaseMode(value string) (DatabaseMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "embedded":
		return DatabaseModeEmbedded, true
	case "external":
		return DatabaseModeExternal, true
	default:
		return "", false
	}
}
