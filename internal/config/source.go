// Package config centralizes runtime configuration loading for ACP processes.
//
// Purpose:
//   - Own all environment and repo-local `.env` access behind typed APIs.
//
// Responsibilities:
//   - Resolve process environment values with deterministic precedence rules.
//   - Expose reusable typed parsing helpers for command and internal packages.
//   - Keep repo-local configuration lookup out of leaf packages.
//
// Scope:
//   - Runtime configuration loading and parsing only.
//
// Usage:
//   - Construct a `Loader` and request typed concern-specific config values.
//
// Invariants/Assumptions:
//   - This package is the only Go package allowed to read `os.Getenv` directly.
//   - Process environment always takes precedence over repo-local `demo/.env`.
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/envfile"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

type lookupSource interface {
	Lookup(key string) (string, bool, error)
}

type processEnvSource struct{}

func (processEnvSource) Lookup(key string) (string, bool, error) {
	value, ok := os.LookupEnv(key)
	return value, ok, nil
}

type fileEnvSource struct {
	path string
}

func (s fileEnvSource) Lookup(key string) (string, bool, error) {
	value, ok, err := envfile.LookupFile(s.path, key)
	return value, ok, err
}

// Loader resolves ACP runtime configuration from process env and repo-local files.
type Loader struct {
	process        lookupSource
	repoFile       lookupSource
	repoRoot       string
	repoRootLoaded bool
	repoRootErr    error
}

// NewLoader constructs a runtime loader backed by the current process environment.
func NewLoader() *Loader {
	return &Loader{process: processEnvSource{}}
}

// NewTestLoader constructs a loader backed by explicit process values.
func NewTestLoader(processValues map[string]string, repoRoot string, repoValues map[string]string) *Loader {
	return &Loader{
		process:        staticSource(processValues),
		repoFile:       staticSource(repoValues),
		repoRoot:       repoRoot,
		repoRootLoaded: true,
	}
}

// WithRepoRoot clones the loader with an explicit repository root override.
func (l *Loader) WithRepoRoot(repoRoot string) *Loader {
	if l == nil {
		l = NewLoader()
	}
	clone := *l
	clone.repoRoot = strings.TrimSpace(repoRoot)
	clone.repoRootLoaded = clone.repoRoot != ""
	clone.repoRootErr = nil
	clone.repoFile = nil
	return &clone
}

type staticSource map[string]string

func (s staticSource) Lookup(key string) (string, bool, error) {
	value, ok := s[key]
	return value, ok, nil
}

func (l *Loader) lookup(key string, includeRepo bool) (string, bool, error) {
	if l == nil {
		return "", false, nil
	}
	if l.process == nil {
		l.process = processEnvSource{}
	}
	value, ok, err := l.process.Lookup(key)
	if err != nil {
		return "", false, err
	}
	if ok {
		return value, true, nil
	}
	if !includeRepo {
		return "", false, nil
	}
	source, err := l.repoSource()
	if err != nil {
		return "", false, err
	}
	if source == nil {
		return "", false, nil
	}
	return source.Lookup(key)
}

func (l *Loader) repoSource() (lookupSource, error) {
	if l.repoFile != nil {
		return l.repoFile, nil
	}
	repoRoot, err := l.RepoRoot(context.Background())
	if err != nil || strings.TrimSpace(repoRoot) == "" {
		return nil, err
	}
	l.repoFile = fileEnvSource{path: filepath.Join(repoRoot, "demo", ".env")}
	return l.repoFile, nil
}

// LookupProcess returns a raw process-only value.
func (l *Loader) LookupProcess(key string) (string, bool, error) {
	return l.lookup(key, false)
}

// LookupRepoAware returns a raw value with repo-local fallback.
func (l *Loader) LookupRepoAware(key string) (string, bool, error) {
	return l.lookup(key, true)
}

// String returns a process-only trimmed string.
func (l *Loader) String(key string) string {
	value, _, _ := l.LookupProcess(key)
	return strings.TrimSpace(value)
}

// RepoAwareString returns a trimmed string with repo-local fallback.
func (l *Loader) RepoAwareString(key string) string {
	value, _, _ := l.LookupRepoAware(key)
	return strings.TrimSpace(value)
}

// StringDefault returns a trimmed process-only string or the provided fallback.
func (l *Loader) StringDefault(key, fallback string) string {
	if value := l.String(key); value != "" {
		return value
	}
	return fallback
}

// RepoAwareStringDefault returns a repo-aware trimmed string or fallback.
func (l *Loader) RepoAwareStringDefault(key, fallback string) string {
	if value := l.RepoAwareString(key); value != "" {
		return value
	}
	return fallback
}

// BoolDefault parses a process-only boolean or returns the fallback.
func (l *Loader) BoolDefault(key string, fallback bool) bool {
	value := l.String(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// Float64Ptr parses a nullable float from a process-only key.
func (l *Loader) Float64Ptr(key string) *float64 {
	value := l.String(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// Int64Ptr parses a nullable int64 from a process-only key.
func (l *Loader) Int64Ptr(key string) *int64 {
	value := l.String(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

// RepoRoot resolves the canonical repository root.
func (l *Loader) RepoRoot(ctx context.Context) (string, error) {
	if l != nil && l.repoRootLoaded {
		return l.repoRoot, l.repoRootErr
	}
	if explicit := l.String("ACP_REPO_ROOT"); explicit != "" {
		if l != nil {
			l.repoRoot = explicit
			l.repoRootLoaded = true
		}
		return explicit, nil
	}
	res := proc.Run(ctx, proc.Request{
		Name:    "git",
		Args:    []string{"rev-parse", "--show-toplevel"},
		Timeout: DefaultConnectTimeout,
	})
	if res.Err == nil {
		root := strings.TrimSpace(res.Stdout)
		if l != nil {
			l.repoRoot = root
			l.repoRootLoaded = true
		}
		return root, nil
	}
	wd, err := os.Getwd()
	if l != nil {
		l.repoRoot = wd
		l.repoRootLoaded = true
		l.repoRootErr = err
	}
	return wd, err
}

// RequireRepoRoot resolves the repo root or returns a wrapped error.
func (l *Loader) RequireRepoRoot(ctx context.Context) (string, error) {
	repoRoot, err := l.RepoRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve repo root: %w", err)
	}
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("resolve repo root: empty path")
	}
	return repoRoot, nil
}
