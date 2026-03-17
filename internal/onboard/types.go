// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Own typed onboarding behavior for CLI and IDE tools that route through the
//	AI Control Plane gateway.
//
// Responsibilities:
//   - Carry typed onboarding options, results, verification state, and workflow dependencies.
//   - Define stable tool and mode metadata shapes shared across prompts and runtime.
//   - Keep cross-file onboarding contracts centralized.
//
// Scope:
//   - Shared onboarding types only.
//
// Usage:
//   - Called by `acpctl onboard` and supporting onboarding package files.
//
// Invariants/Assumptions:
//   - Secrets are redacted unless explicitly revealed in a one-time output block.
//   - demo/.env is treated as data, never sourced for execution.
package onboard

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
)

const DefaultBudget = "10.00"

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
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Now          func() time.Time
	KeyGenerator KeyGenerator
	HTTPClient   *http.Client
}

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type KeyGenerator interface {
	Generate(ctx context.Context, req KeyRequest) (GeneratedKey, error)
}

type KeyRequest struct {
	Alias   string
	Budget  string
	BaseURL string
}

type GeneratedKey struct {
	Alias string
	Key   string
}

type VerificationStatus string

const (
	VerificationStatusPass VerificationStatus = "pass"
	VerificationStatusFail VerificationStatus = "fail"
	VerificationStatusSkip VerificationStatus = "skip"
)

type VerificationCheck struct {
	Name        string
	Status      VerificationStatus
	Summary     string
	Issues      []string
	Remediation []string
}

type VerificationReport struct {
	Checks []VerificationCheck
	Issues []string
}

func (r VerificationReport) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == VerificationStatusFail {
			return true
		}
	}
	return false
}

type ToolConfigResult struct {
	Tool    string
	Path    string
	Written bool
	Skipped bool
	Summary string
	Issues  []string
}

func (r ToolConfigResult) HasIssues() bool {
	return len(r.Issues) > 0
}

type modeSpec struct {
	Name    string
	Summary string
}

type toolSpec struct {
	Name         string
	Summary      string
	DefaultMode  string
	DefaultAlias func(mode string) string
	DefaultModel func(mode string) string
	Modes        []modeSpec
}

type prerequisites struct {
	MasterKey string
}

type runState struct {
	Options        Options
	Prereqs        prerequisites
	Gateway        config.GatewaySettings
	GeneratedAlias string
	KeyValue       string
	ToolConfig     ToolConfigResult
	Verification   VerificationReport
}
