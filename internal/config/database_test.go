// database_test.go - Coverage for typed database configuration resolution.
//
// Purpose:
//   - Verify database mode normalization, defaults, and ambiguity detection.
//
// Responsibilities:
//   - Cover embedded defaults, explicit external mode, and repo fallback mode.
//   - Cover ambiguous DATABASE_URL handling and default embedded URL exemptions.
//
// Scope:
//   - Database settings resolution only.
//
// Usage:
//   - Run via `go test ./internal/config`.
//
// Invariants/Assumptions:
//   - Tests use explicit loaders to avoid depending on host env state.
package config

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeDatabaseMode(t *testing.T) {
	tests := []struct {
		input    string
		expected DatabaseMode
		ok       bool
	}{
		{input: "embedded", expected: DatabaseModeEmbedded, ok: true},
		{input: " EXTERNAL ", expected: DatabaseModeExternal, ok: true},
		{input: "other", ok: false},
	}

	for _, tt := range tests {
		got, ok := normalizeDatabaseMode(tt.input)
		if got != tt.expected || ok != tt.ok {
			t.Fatalf("normalizeDatabaseMode(%q) = %q, %t", tt.input, got, ok)
		}
	}
}

func TestLoaderDatabaseDefaultsToEmbedded(t *testing.T) {
	settings := NewTestLoader(map[string]string{}, "/repo", nil).Database(context.Background())

	if settings.Mode != DatabaseModeEmbedded {
		t.Fatalf("Mode = %q", settings.Mode)
	}
	if settings.Name != "litellm" || settings.User != "litellm" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
	if settings.RepoEnvPath != "/repo/demo/.env" {
		t.Fatalf("RepoEnvPath = %q", settings.RepoEnvPath)
	}
}

func TestLoaderDatabaseUsesExplicitExternalMode(t *testing.T) {
	settings := NewTestLoader(map[string]string{
		"ACP_DATABASE_MODE": "external",
		"DATABASE_URL":      "postgres://external",
		"DB_NAME":           "custom-db",
		"DB_USER":           "custom-user",
	}, "/repo", nil).Database(context.Background())

	if settings.Mode != DatabaseModeExternal || settings.URL != "postgres://external" {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	if settings.Name != "custom-db" || settings.User != "custom-user" {
		t.Fatalf("unexpected name/user: %+v", settings)
	}
}

func TestLoaderDatabaseUsesRepoFallbackMode(t *testing.T) {
	settings := NewTestLoader(nil, "/repo", map[string]string{
		"ACP_DATABASE_MODE": "external",
		"DATABASE_URL":      "postgres://repo",
	}).Database(context.Background())

	if settings.Mode != DatabaseModeExternal || settings.URL != "postgres://repo" {
		t.Fatalf("unexpected repo fallback settings: %+v", settings)
	}
}

func TestLoaderDatabaseFlagsAmbiguousExternalURL(t *testing.T) {
	settings := NewTestLoader(nil, "/repo", map[string]string{
		"DATABASE_URL": "postgres://repo",
	}).Database(context.Background())

	if settings.AmbiguousErr == nil || !strings.Contains(settings.AmbiguousErr.Error(), "ambiguous database configuration") {
		t.Fatalf("expected ambiguity error, got %v", settings.AmbiguousErr)
	}
}

func TestLoaderDatabaseAllowsDefaultEmbeddedURLWithoutAmbiguity(t *testing.T) {
	settings := NewTestLoader(nil, "/repo", map[string]string{
		"DATABASE_URL": defaultEmbeddedDatabaseURL,
	}).Database(context.Background())

	if settings.AmbiguousErr != nil {
		t.Fatalf("expected default embedded URL to stay unambiguous, got %v", settings.AmbiguousErr)
	}
}
