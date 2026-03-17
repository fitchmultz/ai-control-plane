// Package hostdeploy centralizes typed host check/apply execution.
//
// Purpose:
//   - Provide one reusable host-first Ansible execution path for CLI commands
//   - and upgrade workflows.
//
// Responsibilities:
//   - Validate the tracked host playbook inputs and local prerequisites.
//   - Execute Ansible syntax-check plus the real check/apply run.
//   - Classify failures into prereq, usage, domain, and runtime buckets.
//
// Scope:
//   - Host deployment orchestration only.
//
// Usage:
//   - Called by `acpctl host check`, `acpctl host apply`, and upgrade
//   - workflows that reuse the host-first convergence contract.
//
// Invariants/Assumptions:
//   - `deploy/ansible/playbooks/gateway_host.yml` is the source of truth.
//   - Remote-path overrides are forwarded via Ansible extra-vars.
package hostdeploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const timeout = 60 * time.Minute

type ErrorKind string

const (
	ErrorKindPrereq  ErrorKind = "prereq"
	ErrorKindUsage   ErrorKind = "usage"
	ErrorKindDomain  ErrorKind = "domain"
	ErrorKindRuntime ErrorKind = "runtime"
)

// Error captures a classified host deployment failure.
type Error struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return "host deployment failed"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "host deployment failed"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Options configures one host deployment execution.
type Options struct {
	Mode                 string
	RepoRoot             string
	Inventory            string
	Limit                string
	RepoPath             string
	EnvFile              string
	TLSMode              string
	PublicURL            string
	WaitForStabilization bool
	RunSmokeTests        bool
	StabilizationSeconds string
	ExtraVars            []string
	Stdout               io.Writer
	Stderr               io.Writer
}

// Execute runs the tracked host deployment workflow in check or apply mode.
func Execute(ctx context.Context, opts Options) error {
	if strings.TrimSpace(opts.Mode) != "check" && strings.TrimSpace(opts.Mode) != "apply" {
		return &Error{Kind: ErrorKindUsage, Message: fmt.Sprintf("unsupported host deploy mode: %s", opts.Mode)}
	}
	if err := proc.ValidateExecutable("ansible-playbook"); err != nil {
		return &Error{Kind: ErrorKindPrereq, Message: fmt.Sprintf("ansible-playbook not found or not executable: %v", err), Cause: err}
	}
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return &Error{Kind: ErrorKindUsage, Message: "repository root is required"}
	}

	playbookPath := filepath.Join(opts.RepoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml")
	ansibleCfg := filepath.Join(opts.RepoRoot, "deploy", "ansible", "ansible.cfg")
	if _, err := os.Stat(opts.Inventory); err != nil {
		return &Error{Kind: ErrorKindUsage, Message: fmt.Sprintf("inventory file not found: %s", opts.Inventory), Cause: err}
	}
	if _, err := os.Stat(playbookPath); err != nil {
		return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("playbook not found: %s", playbookPath), Cause: err}
	}
	if _, err := os.Stat(ansibleCfg); err != nil {
		return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("ansible config not found: %s", ansibleCfg), Cause: err}
	}

	request := proc.Request{
		Name:    "ansible-playbook",
		Dir:     opts.RepoRoot,
		Env:     []string{"ANSIBLE_CONFIG=" + ansibleCfg},
		Stdout:  writerOrDiscard(opts.Stdout),
		Stderr:  writerOrDiscard(opts.Stderr),
		Timeout: timeout,
	}

	args := append(baseArgs(opts.Inventory, playbookPath, opts.Limit), modeArgs(opts.Mode)...)
	for _, item := range extraVars(opts) {
		args = append(args, "--extra-vars", item)
	}
	if err := runProc(ctx, request, append([]string{"--syntax-check"}, args...), "host "+opts.Mode+" syntax-check"); err != nil {
		return err
	}
	return runProc(ctx, request, args, "host "+opts.Mode)
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}

func baseArgs(inventory string, playbookPath string, limit string) []string {
	args := []string{"-i", inventory, playbookPath}
	if strings.TrimSpace(limit) != "" {
		args = append(args, "--limit", limit)
	}
	return args
}

func modeArgs(mode string) []string {
	if strings.TrimSpace(mode) == "check" {
		return []string{"--check"}
	}
	return nil
}

func extraVars(opts Options) []string {
	values := append([]string(nil), opts.ExtraVars...)
	if strings.TrimSpace(opts.RepoPath) != "" {
		values = append(values, "acp_repo_path="+opts.RepoPath)
	}
	if strings.TrimSpace(opts.EnvFile) != "" {
		values = append(values, "acp_env_file="+opts.EnvFile)
	}
	if strings.TrimSpace(opts.TLSMode) != "" {
		values = append(values, "acp_tls_mode="+opts.TLSMode)
	}
	if strings.TrimSpace(opts.PublicURL) != "" {
		values = append(values, "acp_public_url="+opts.PublicURL)
	}
	values = append(values, "acp_wait_for_stabilization="+strconv.FormatBool(opts.WaitForStabilization))
	values = append(values, "acp_run_smoke_tests="+strconv.FormatBool(opts.RunSmokeTests))
	if strings.TrimSpace(opts.StabilizationSeconds) != "" {
		values = append(values, "acp_stabilization_seconds="+opts.StabilizationSeconds)
	}
	return values
}

func runProc(ctx context.Context, request proc.Request, args []string, commandName string) error {
	request.Args = args
	res := proc.Run(ctx, request)
	if res.Err == nil {
		return nil
	}

	var execErr *proc.ExecError
	if errors.As(res.Err, &execErr) {
		switch execErr.Kind {
		case proc.KindNotFound:
			return &Error{Kind: ErrorKindPrereq, Message: "ansible-playbook not found", Cause: res.Err}
		case proc.KindTimeout:
			return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("%s timed out", commandName), Cause: res.Err}
		case proc.KindCanceled:
			return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("%s canceled", commandName), Cause: res.Err}
		case proc.KindExit:
			kind := ErrorKindRuntime
			if strings.Contains(commandName, "syntax-check") {
				kind = ErrorKindDomain
			}
			return &Error{Kind: kind, Message: fmt.Sprintf("%s failed", commandName), Cause: res.Err}
		default:
			return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("%s failed: %v", commandName, res.Err), Cause: res.Err}
		}
	}
	return &Error{Kind: ErrorKindRuntime, Message: fmt.Sprintf("%s failed: %v", commandName, res.Err), Cause: res.Err}
}
