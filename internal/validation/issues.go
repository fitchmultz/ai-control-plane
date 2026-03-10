// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Provide a shared issue accumulator for repository validators.
//
// Responsibilities:
//   - Collect non-empty validation findings without repetitive slice plumbing.
//   - Provide deterministic sorted output for CLI and tests.
//   - Keep validator implementations focused on rule logic.
//
// Scope:
//   - Shared issue accumulation helpers only.
//
// Usage:
//   - Used by validation, security, and contract validators that emit []string findings.
//
// Invariants/Assumptions:
//   - Blank findings are ignored.
//   - Sorted output always returns a cloned slice.
package validation

import (
	"fmt"
	"sort"

	"github.com/mitchfultz/ai-control-plane/internal/textutil"
)

// Issues accumulates validator findings with deterministic rendering helpers.
type Issues struct {
	items []string
}

// NewIssues constructs an accumulator with an optional initial capacity hint.
func NewIssues(capacity ...int) Issues {
	size := 0
	if len(capacity) > 0 && capacity[0] > 0 {
		size = capacity[0]
	}
	return Issues{items: make([]string, 0, size)}
}

// Add records a non-empty issue.
func (i *Issues) Add(issue string) {
	if textutil.IsBlank(issue) {
		return
	}
	i.items = append(i.items, issue)
}

// Addf formats and records a non-empty issue.
func (i *Issues) Addf(format string, args ...any) {
	i.Add(fmt.Sprintf(format, args...))
}

// Extend records every non-empty issue from the provided slice.
func (i *Issues) Extend(issues []string) {
	for _, issue := range issues {
		i.Add(issue)
	}
}

// Len reports the number of accumulated issues.
func (i Issues) Len() int {
	return len(i.items)
}

// ToSlice returns an unsorted copy of the accumulated issues.
func (i Issues) ToSlice() []string {
	return append([]string(nil), i.items...)
}

// Sorted returns a sorted copy of the accumulated issues.
func (i Issues) Sorted() []string {
	out := i.ToSlice()
	sort.Strings(out)
	return out
}
