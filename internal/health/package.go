// Package health defines shared health-state primitives across ACP workflows.
//
// Purpose:
//   - Provide one canonical health vocabulary for status, doctor, and future
//     runtime-reporting workflows.
//
// Responsibilities:
//   - Define health levels and severity ordering.
//   - Provide deterministic aggregation helpers.
//
// Scope:
//   - Shared health primitives only.
//
// Usage:
//   - Imported by status, doctor, and other health-oriented packages.
//
// Invariants/Assumptions:
//   - Unhealthy is always the highest severity.
//   - Healthy is always the lowest severity.
package health

type Level string

const (
	LevelHealthy   Level = "healthy"
	LevelUnknown   Level = "unknown"
	LevelWarning   Level = "warning"
	LevelUnhealthy Level = "unhealthy"
)

// Rank returns a stable severity ordering for health levels.
func Rank(level Level) int {
	switch level {
	case LevelHealthy:
		return 0
	case LevelUnknown:
		return 1
	case LevelWarning:
		return 2
	case LevelUnhealthy:
		return 3
	default:
		return 3
	}
}

// Worst returns the highest-severity level from the provided inputs.
func Worst(levels ...Level) Level {
	worst := LevelHealthy
	for _, level := range levels {
		if Rank(level) > Rank(worst) {
			worst = level
		}
	}
	return worst
}
