// proc/exec.go - Canonical subprocess execution for operator-facing commands.
//
// Purpose:
//
//	Provide one deadline-aware, cancellable subprocess execution path with
//	consistent stream wiring and error classification.
//
// Responsibilities:
//   - Require a caller context for every subprocess run.
//   - Apply a safe default timeout when the caller has no deadline.
//   - Capture stdout/stderr while optionally streaming them to callers.
//   - Classify timeout, cancel, not-found, start, and exit-status failures.
//
// Scope:
//   - Covers local subprocess execution for CLI and internal operator workflows.
//
// Usage:
//   - Call `Run(ctx, Request{...})` from command handlers and internal helpers.
//   - Call `RunAttached(ctx, Request{...})` for operator-facing interactive
//     subprocesses that must inherit the caller terminal without an implicit
//     fallback timeout.
//
// Invariants/Assumptions:
//   - Nil contexts are rejected.
//   - Empty command names are rejected.
//   - Timeout and cancellation map to repository runtime exit handling.
package proc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

const DefaultTimeout = 30 * time.Second

var (
	ErrExecutableNotFound      = errors.New("executable not found")
	ErrExecutableIsDirectory   = errors.New("executable path is a directory")
	ErrExecutableNotExecutable = errors.New("executable path is not executable")
)

type ErrorKind string

const (
	KindTimeout  ErrorKind = "timeout"
	KindCanceled ErrorKind = "canceled"
	KindNotFound ErrorKind = "not_found"
	KindExit     ErrorKind = "exit"
	KindStart    ErrorKind = "start"
)

type Request struct {
	Name    string
	Args    []string
	Dir     string
	Env     []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Timeout time.Duration
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

type ExecError struct {
	Name     string
	Args     []string
	Kind     ErrorKind
	ExitCode int
	Timeout  time.Duration
	Err      error
}

func (e *ExecError) Error() string {
	switch e.Kind {
	case KindTimeout:
		return fmt.Sprintf("%s timed out after %s", e.command(), e.Timeout)
	case KindCanceled:
		return fmt.Sprintf("%s canceled", e.command())
	case KindNotFound:
		return fmt.Sprintf("%s not found: %v", e.Name, e.Err)
	case KindExit:
		return fmt.Sprintf("%s exited with status %d", e.command(), e.ExitCode)
	default:
		return fmt.Sprintf("%s failed: %v", e.command(), e.Err)
	}
}

func (e *ExecError) Unwrap() error {
	return e.Err
}

func (e *ExecError) command() string {
	name := strings.TrimSpace(e.Name)
	if name == "" {
		name = "subprocess"
	}
	if len(e.Args) == 0 {
		return name
	}
	return name + " " + strings.Join(e.Args, " ")
}

func Run(ctx context.Context, req Request) Result {
	return run(ctx, req, true)
}

// RunAttached executes a subprocess with caller-owned stdin/stdout/stderr and
// without applying the default timeout when the caller does not provide one.
func RunAttached(ctx context.Context, req Request) Result {
	return run(ctx, req, false)
}

func run(ctx context.Context, req Request, applyDefaultTimeout bool) Result {
	if ctx == nil {
		return invalidStartResult(req, errors.New("proc.Run requires a non-nil context"))
	}
	if strings.TrimSpace(req.Name) == "" {
		return invalidStartResult(req, errors.New("proc.Run requires a non-empty command name"))
	}

	runCtx, cancel, effectiveTimeout := withTimeout(ctx, req.Timeout, applyDefaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, req.Name, req.Args...)
	if req.Dir != "" {
		cmd.Dir = req.Dir
	}
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}
	if req.Stdin != nil {
		cmd.Stdin = req.Stdin
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	if req.Stdout != nil {
		cmd.Stdout = io.MultiWriter(req.Stdout, &stdoutBuf)
	} else {
		cmd.Stdout = &stdoutBuf
	}
	if req.Stderr != nil {
		cmd.Stderr = io.MultiWriter(req.Stderr, &stderrBuf)
	} else {
		cmd.Stderr = &stderrBuf
	}

	err := cmd.Run()
	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}
	if err == nil {
		return result
	}

	result.Err = classify(runCtx, req, effectiveTimeout, err)
	if code, ok := ExitCode(result.Err); ok {
		result.ExitCode = code
	} else {
		result.ExitCode = -1
	}
	return result
}

func ValidateExecutable(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return fmt.Errorf("%w: empty command", ErrExecutableNotFound)
	}

	if strings.ContainsRune(trimmed, filepath.Separator) {
		info, err := os.Stat(trimmed)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrExecutableNotFound, err)
		}
		if info.IsDir() {
			return fmt.Errorf("%w: %s", ErrExecutableIsDirectory, trimmed)
		}
		if info.Mode()&0o111 == 0 {
			return fmt.Errorf("%w: %s", ErrExecutableNotExecutable, trimmed)
		}
		return nil
	}

	if _, err := exec.LookPath(trimmed); err != nil {
		return fmt.Errorf("%w: %v", ErrExecutableNotFound, err)
	}
	return nil
}

func invalidStartResult(req Request, err error) Result {
	return Result{
		Err: &ExecError{
			Name:     strings.TrimSpace(req.Name),
			Args:     append([]string(nil), req.Args...),
			Kind:     KindStart,
			ExitCode: -1,
			Err:      err,
		},
		ExitCode: -1,
	}
}

func withTimeout(ctx context.Context, requested time.Duration, applyDefaultTimeout bool) (context.Context, context.CancelFunc, time.Duration) {
	if deadline, ok := ctx.Deadline(); ok {
		return ctx, func() {}, time.Until(deadline)
	}
	if requested <= 0 && applyDefaultTimeout {
		requested = DefaultTimeout
	}
	if requested <= 0 {
		return ctx, func() {}, 0
	}
	child, cancel := context.WithTimeout(ctx, requested)
	return child, cancel, requested
}

func classify(ctx context.Context, req Request, effectiveTimeout time.Duration, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &ExecError{
			Name:     req.Name,
			Args:     append([]string(nil), req.Args...),
			Kind:     KindTimeout,
			ExitCode: -1,
			Timeout:  effectiveTimeout,
			Err:      context.DeadlineExceeded,
		}
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return &ExecError{
			Name:     req.Name,
			Args:     append([]string(nil), req.Args...),
			Kind:     KindCanceled,
			ExitCode: -1,
			Err:      context.Canceled,
		}
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return &ExecError{
			Name:     req.Name,
			Args:     append([]string(nil), req.Args...),
			Kind:     KindNotFound,
			ExitCode: 127,
			Err:      err,
		}
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return &ExecError{
			Name:     req.Name,
			Args:     append([]string(nil), req.Args...),
			Kind:     KindExit,
			ExitCode: exitErr.ExitCode(),
			Err:      err,
		}
	}

	return &ExecError{
		Name:     req.Name,
		Args:     append([]string(nil), req.Args...),
		Kind:     KindStart,
		ExitCode: -1,
		Err:      err,
	}
}

func IsTimeout(err error) bool {
	return hasKind(err, KindTimeout)
}

func IsCanceled(err error) bool {
	return hasKind(err, KindCanceled)
}

func IsNotFound(err error) bool {
	return hasKind(err, KindNotFound)
}

func IsStart(err error) bool {
	return hasKind(err, KindStart)
}

func IsExit(err error) bool {
	return hasKind(err, KindExit)
}

func ExitCode(err error) (int, bool) {
	var execErr *ExecError
	if errors.As(err, &execErr) && execErr.ExitCode >= 0 {
		return execErr.ExitCode, true
	}
	return 0, false
}

func ACPExitCode(err error) int {
	if err == nil {
		return exitcodes.ACPExitSuccess
	}

	switch {
	case IsNotFound(err):
		return exitcodes.ACPExitPrereq
	case IsTimeout(err), IsCanceled(err):
		return exitcodes.ACPExitRuntime
	}

	code, ok := ExitCode(err)
	if !ok {
		return exitcodes.ACPExitRuntime
	}

	switch code {
	case exitcodes.ACPExitDomain, exitcodes.ACPExitPrereq, exitcodes.ACPExitRuntime, exitcodes.ACPExitUsage:
		return code
	case 127:
		return exitcodes.ACPExitPrereq
	default:
		return exitcodes.ACPExitRuntime
	}
}

func hasKind(err error, want ErrorKind) bool {
	var execErr *ExecError
	return errors.As(err, &execErr) && execErr.Kind == want
}
