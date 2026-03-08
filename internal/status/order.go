// Package status provides aggregated system health status collection.
//
// Purpose:
//   - Define stable component ordering for runtime status rendering.
//
// Responsibilities:
//   - Keep the human-readable component order centralized and reusable.
//
// Scope:
//   - Shared ordering constants only.
//
// Usage:
//   - Used by report writers and runtime inspection composition.
//
// Invariants/Assumptions:
//   - Ordering changes are deliberate because they affect operator output.
package status

// DefaultComponentOrder defines stable human output ordering for runtime components.
var DefaultComponentOrder = []string{"gateway", "database", "keys", "budget", "detections"}
