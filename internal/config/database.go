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
)

const defaultEmbeddedDatabaseURL = "postgresql://litellm:litellm@postgres:5432/litellm"

// DatabaseSettings captures typed database runtime configuration.
type DatabaseSettings struct {
	Mode         string
	Name         string
	User         string
	URL          string
	RepoRoot     string
	RepoEnvPath  string
	AmbiguousErr error
}

// Database returns the effective typed database settings.
func (l *Loader) Database(ctx context.Context) DatabaseSettings {
	repoRoot, _ := l.RepoRoot(ctx)
	settings := DatabaseSettings{
		Mode:        "embedded",
		Name:        l.StringDefault("DB_NAME", "litellm"),
		User:        l.StringDefault("DB_USER", "litellm"),
		URL:         l.RepoAwareString("DATABASE_URL"),
		RepoRoot:    repoRoot,
		RepoEnvPath: repoRoot + "/demo/.env",
	}
	if mode, ok := normalizeDatabaseMode(l.String("ACP_DATABASE_MODE")); ok {
		settings.Mode = mode
	} else if mode, ok := normalizeDatabaseMode(l.RepoAwareString("ACP_DATABASE_MODE")); ok {
		settings.Mode = mode
	}
	if settings.Mode == "embedded" && strings.TrimSpace(settings.URL) != "" && settings.URL != defaultEmbeddedDatabaseURL {
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

func normalizeDatabaseMode(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "embedded":
		return "embedded", true
	case "external":
		return "external", true
	default:
		return "", false
	}
}
