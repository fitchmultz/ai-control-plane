// exec_test.go - Deterministic tests for the canonical subprocess executor.
//
// Purpose:
//
//	Verify subprocess execution captures streams and classifies timeout,
//	cancellation, not-found, and exit failures deterministically.
//
// Responsibilities:
//   - Re-exec the Go test binary as a helper subprocess.
//   - Assert stdout/stderr capture and stream mirroring.
//   - Assert exit-code and timeout/cancel classification.
//
// Scope:
//   - Covers the internal/proc package only.
//
// Usage:
//   - Run via `go test ./internal/proc`.
//
// Invariants/Assumptions:
//   - Tests do not depend on host binaries.
//   - Helper-process execution uses the current test binary.
package proc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestProcHelperProcess(t *testing.T) {
	t.Helper()
	sep := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		return
	}

	mode := os.Args[sep+1]
	switch mode {
	case "stdout-stderr":
		fmt.Fprint(os.Stdout, os.Args[sep+2])
		fmt.Fprint(os.Stderr, os.Args[sep+3])
		os.Exit(0)
	case "exit":
		code, _ := strconv.Atoi(os.Args[sep+2])
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(code)
	case "sleep":
		d, _ := time.ParseDuration(os.Args[sep+2])
		time.Sleep(d)
		os.Exit(0)
	default:
		os.Exit(99)
	}
}

func helperRequest(mode string, args ...string) Request {
	return Request{
		Name: os.Args[0],
		Args: append([]string{"-test.run=TestProcHelperProcess", "--", mode}, args...),
	}
}

func TestRunCapturesStdoutAndStderr(t *testing.T) {
	var streamedOut bytes.Buffer
	var streamedErr bytes.Buffer

	res := Run(context.Background(), Request{
		Name:   os.Args[0],
		Args:   helperRequest("stdout-stderr", "hello stdout", "hello stderr").Args,
		Stdout: &streamedOut,
		Stderr: &streamedErr,
	})
	if res.Err != nil {
		t.Fatalf("Run() error = %v", res.Err)
	}
	if res.Stdout != "hello stdout" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if res.Stderr != "hello stderr" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if streamedOut.String() != "hello stdout" {
		t.Fatalf("streamed stdout = %q", streamedOut.String())
	}
	if streamedErr.String() != "hello stderr" {
		t.Fatalf("streamed stderr = %q", streamedErr.String())
	}
}

func TestRunClassifiesExit(t *testing.T) {
	res := Run(context.Background(), helperRequest("exit", "7"))
	if !IsExit(res.Err) {
		t.Fatalf("expected exit classification, got %v", res.Err)
	}
	if code, ok := ExitCode(res.Err); !ok || code != 7 {
		t.Fatalf("exit code = %d, %v; want 7, true", code, ok)
	}
}

func TestRunClassifiesTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	res := Run(ctx, helperRequest("sleep", "1s"))
	if !IsTimeout(res.Err) {
		t.Fatalf("expected timeout classification, got %v", res.Err)
	}
}

func TestRunClassifiesCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := Run(ctx, helperRequest("sleep", "1s"))
	if !IsCanceled(res.Err) {
		t.Fatalf("expected cancel classification, got %v", res.Err)
	}
}

func TestACPExitCodeMappings(t *testing.T) {
	if code := ACPExitCode(nil); code != 0 {
		t.Fatalf("ACPExitCode(nil) = %d, want 0", code)
	}

	res := Run(context.Background(), helperRequest("exit", "64"))
	if code := ACPExitCode(res.Err); code != 64 {
		t.Fatalf("ACPExitCode(exit 64) = %d, want 64", code)
	}

	notFound := Run(context.Background(), Request{Name: filepathDoesNotExist()})
	if code := ACPExitCode(notFound.Err); code != 2 {
		t.Fatalf("ACPExitCode(not found) = %d, want 2", code)
	}
}

func filepathDoesNotExist() string {
	return "/definitely/missing/acpctl-proc-helper"
}
