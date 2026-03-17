// Package upgrade provides typed host-first upgrade and rollback workflows.
//
// Purpose:
//   - Classify upgrade workflow failures for CLI exit-code mapping.
//
// Responsibilities:
//   - Wrap domain, prereq, and runtime upgrade failures consistently.
//   - Expose typed predicates for command-layer error handling.
//
// Scope:
//   - Upgrade error typing only.
//
// Usage:
//   - Returned by `Check`, `Execute`, and `Rollback`.
//
// Invariants/Assumptions:
//   - Domain failures represent unsupported paths or contract violations.
//   - Prereq failures represent missing local dependencies.
package upgrade

import "errors"

type ErrorKind string

const (
	ErrorKindDomain  ErrorKind = "domain"
	ErrorKindPrereq  ErrorKind = "prereq"
	ErrorKindRuntime ErrorKind = "runtime"
)

// Error wraps one classified upgrade failure.
type Error struct {
	Kind ErrorKind
	Err  error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return "upgrade failed"
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

// IsKind reports whether err or one of its wrappers is an upgrade Error of the requested kind.
func IsKind(err error, kind ErrorKind) bool {
	var typed *Error
	if !errors.As(err, &typed) {
		return false
	}
	return typed.Kind == kind
}
