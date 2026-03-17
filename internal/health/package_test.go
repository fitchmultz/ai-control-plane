// package_test.go - Tests for shared health primitives.
//
// Purpose:
//   - Verify severity ranking and aggregation for canonical health levels.
//
// Responsibilities:
//   - Cover deterministic ordering and worst-level aggregation.
//
// Scope:
//   - health package helper behavior only.
//
// Usage:
//   - Run via `go test ./internal/health`.
//
// Invariants/Assumptions:
//   - Unknown ranks above healthy and below warning.
package health

import "testing"

func TestRank(t *testing.T) {
	cases := []struct {
		level Level
		want  int
	}{
		{LevelHealthy, 0},
		{LevelUnknown, 1},
		{LevelWarning, 2},
		{LevelUnhealthy, 3},
		{Level("mystery"), 3},
	}

	for _, tc := range cases {
		if got := Rank(tc.level); got != tc.want {
			t.Fatalf("Rank(%q) = %d, want %d", tc.level, got, tc.want)
		}
	}
}

func TestWorst(t *testing.T) {
	if got := Worst(LevelHealthy, LevelUnknown, LevelWarning); got != LevelWarning {
		t.Fatalf("Worst(...) = %q, want %q", got, LevelWarning)
	}
	if got := Worst(LevelHealthy, LevelUnhealthy); got != LevelUnhealthy {
		t.Fatalf("Worst(...) = %q, want %q", got, LevelUnhealthy)
	}
	if got := Worst(); got != LevelHealthy {
		t.Fatalf("Worst() = %q, want %q", got, LevelHealthy)
	}
}
