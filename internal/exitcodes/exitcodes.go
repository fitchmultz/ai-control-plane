// Package exitcodes defines the repository-wide process exit-code contract.
//
// Purpose:
//
//	Provide a single source of truth for shell + Go command exit semantics.
//
// Responsibilities:
//   - Expose named constants for success, domain failure, prereq failure,
//     runtime/internal failure, and usage failure.
//
// Non-scope:
//   - Does not format or print errors.
//   - Does not map arbitrary errors to exit codes.
//
// Invariants/Assumptions:
//   - Values are stable and intentionally aligned with existing shell scripts.
//   - ACPExitUsage matches sysexits EX_USAGE (64).
package exitcodes

const (
	ACPExitSuccess = 0
	ACPExitDomain  = 1
	ACPExitPrereq  = 2
	ACPExitRuntime = 3
	ACPExitUsage   = 64
)
