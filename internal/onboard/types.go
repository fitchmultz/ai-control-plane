// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Own typed onboarding behavior for CLI and IDE tools that route through the
//	AI Control Plane gateway.
//
// Responsibilities:
//   - Parse onboarding options and resolve tool-specific defaults.
//   - Load required secrets from environment or demo/.env safely.
//   - Generate virtual keys, verify gateway connectivity, and render exports.
//   - Optionally write ACP-managed Codex configuration atomically.
//
// Scope:
//   - Local onboarding orchestration and output rendering only.
//
// Usage:
//   - Called by `acpctl onboard`.
//
// Invariants/Assumptions:
//   - Secrets are redacted unless explicitly requested.
//   - demo/.env is treated as data, never sourced for execution.
package onboard

import (
	"context"
	"io"
	"net/http"
	"time"
)

const (
	DefaultHost   = "127.0.0.1"
	DefaultPort   = "4000"
	DefaultBudget = "10.00"
)

type Options struct {
	RepoRoot     string
	Tool         string
	Mode         string
	Alias        string
	Budget       string
	Model        string
	Host         string
	Port         string
	UseTLS       bool
	Verify       bool
	WriteConfig  bool
	ShowKey      bool
	Stdout       io.Writer
	Stderr       io.Writer
	Now          func() time.Time
	KeyGenerator KeyGenerator
	HTTPClient   *http.Client
}

type Result struct {
	ExitCode int
}

type KeyGenerator interface {
	Generate(ctx context.Context, req KeyRequest) (GeneratedKey, error)
}

type KeyRequest struct {
	Alias  string
	Budget string
	Host   string
	Port   string
}

type GeneratedKey struct {
	Alias string
	Key   string
}

type toolSpec struct {
	Name           string
	DefaultMode    string
	DefaultModel   func(mode string) string
	SupportedModes map[string]struct{}
}

type prerequisites struct {
	MasterKey string
}

type runState struct {
	Options        Options
	Prereqs        prerequisites
	BaseURL        string
	GeneratedAlias string
	KeyValue       string
}
