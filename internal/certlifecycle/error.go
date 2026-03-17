// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Classify certificate lifecycle failures for CLI and health exit-code mapping.
//
// Responsibilities:
//   - Wrap domain, prerequisite, and runtime certificate failures consistently.
//   - Expose typed predicates for command-layer error handling.
//
// Scope:
//   - Certificate lifecycle error typing only.
//
// Usage:
//   - Returned by `Check`, `Renew`, and host timer installation workflows.
//
// Invariants/Assumptions:
//   - Domain failures represent unsupported or unhealthy certificate state.
//   - Prerequisite failures represent missing Docker/runtime dependencies.
package certlifecycle

import "errors"

type ErrorKind string

const (
	ErrorKindDomain  ErrorKind = "domain"
	ErrorKindPrereq  ErrorKind = "prereq"
	ErrorKindRuntime ErrorKind = "runtime"
)

// Error wraps one classified certificate lifecycle failure.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return "certificate lifecycle failed"
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrap(kind ErrorKind, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	return &Error{Kind: kind, Err: err}
}

// IsKind reports whether err or one of its wrappers is a certificate Error of the requested kind.
func IsKind(err error, kind ErrorKind) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind == kind
}
