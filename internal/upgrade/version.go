// Package upgrade provides typed host-first upgrade and rollback workflows.
//
// Purpose:
//   - Parse and compare tracked ACP semantic versions for upgrade planning.
//
// Responsibilities:
//   - Validate semantic version inputs against the tracked VERSION contract.
//   - Reject `dev` for upgrade workflows.
//   - Provide deterministic semantic-version comparison.
//
// Scope:
//   - Version parsing and comparison only.
//
// Usage:
//   - Used by upgrade plan, check, execute, and rollback workflows.
//
// Invariants/Assumptions:
//   - Upgrade workflows require tagged semantic versions, not `dev`.
package upgrade

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
)

// Version captures one parsed semantic version.
type Version struct {
	Raw   string
	Major int
	Minor int
	Patch int
}

// ParseVersion validates and parses a semantic version string.
func ParseVersion(raw string) (Version, error) {
	trimmed := strings.TrimSpace(raw)
	if err := bundle.ValidateVersion(trimmed); err != nil {
		return Version{}, err
	}
	if trimmed == "dev" {
		return Version{}, fmt.Errorf("upgrade workflows require a tagged semantic version, not dev")
	}

	core := trimmed
	if idx := strings.IndexAny(core, "+-"); idx >= 0 {
		core = core[:idx]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}

	values := make([]int, 3)
	for index, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, fmt.Errorf("parse semantic version %q: %w", raw, err)
		}
		values[index] = parsed
	}

	return Version{
		Raw:   trimmed,
		Major: values[0],
		Minor: values[1],
		Patch: values[2],
	}, nil
}

// Compare returns -1, 0, or 1 when v is older, equal, or newer than other.
func (v Version) Compare(other Version) int {
	switch {
	case v.Major != other.Major:
		if v.Major < other.Major {
			return -1
		}
		return 1
	case v.Minor != other.Minor:
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	case v.Patch != other.Patch:
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	default:
		return 0
	}
}
