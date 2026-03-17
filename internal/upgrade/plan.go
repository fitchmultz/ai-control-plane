// Package upgrade provides typed host-first upgrade and rollback workflows.
//
// Purpose:
//   - Build explicit upgrade plans from tracked release-edge metadata.
//
// Responsibilities:
//   - Resolve only explicitly supported upgrade paths.
//   - Aggregate compatibility, rollback, and operator step metadata.
//   - Publish the tracked default release-edge catalog.
//
// Scope:
//   - Upgrade planning only.
//
// Usage:
//   - Called by CLI plan/check/execute workflows.
//
// Invariants/Assumptions:
//   - Unsupported paths fail until explicit edges are added.
//   - The initial framework release ships with no in-place supported edges.
package upgrade

import (
	"fmt"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/migration"
)

// Plan describes one explicit supported upgrade path.
type Plan struct {
	FromVersion   string                  `json:"from_version"`
	ToVersion     string                  `json:"to_version"`
	Path          []string                `json:"path"`
	Compatibility []string                `json:"compatibility"`
	Rollback      []string                `json:"rollback"`
	Edges         []migration.ReleaseEdge `json:"edges"`
	Steps         []string                `json:"steps"`
}

var defaultCatalog = migration.Catalog{
	Edges: []migration.ReleaseEdge{},
}

// DefaultCatalog returns a defensive copy of the tracked release-edge catalog.
func DefaultCatalog() migration.Catalog {
	out := migration.Catalog{Edges: make([]migration.ReleaseEdge, len(defaultCatalog.Edges))}
	copy(out.Edges, defaultCatalog.Edges)
	return out
}

// ResolveTargetVersion returns an explicit target version or the tracked VERSION value.
func ResolveTargetVersion(repoRoot string, explicit string) string {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		return trimmed
	}
	return bundle.GetDefaultVersion(repoRoot)
}

// BuildPlan resolves one explicit supported upgrade path.
func BuildPlan(from string, to string, catalog migration.Catalog) (*Plan, error) {
	fromVersion, err := ParseVersion(from)
	if err != nil {
		return nil, fmt.Errorf("invalid current version: %w", err)
	}
	toVersion, err := ParseVersion(to)
	if err != nil {
		return nil, fmt.Errorf("invalid target version: %w", err)
	}
	if fromVersion.Compare(toVersion) >= 0 {
		return nil, fmt.Errorf("target version %s must be newer than current version %s", toVersion.Raw, fromVersion.Raw)
	}

	edges, err := catalog.ResolvePath(fromVersion.Raw, toVersion.Raw)
	if err != nil {
		return nil, err
	}

	plan := &Plan{
		FromVersion: fromVersion.Raw,
		ToVersion:   toVersion.Raw,
		Path:        []string{fromVersion.Raw},
		Edges:       append([]migration.ReleaseEdge(nil), edges...),
		Steps: []string{
			"Snapshot the canonical secrets/env file",
			"Create a pre-upgrade embedded database backup",
		},
	}

	compatibilitySeen := map[string]struct{}{}
	rollbackSeen := map[string]struct{}{}

	for _, edge := range edges {
		plan.Path = append(plan.Path, edge.To)
		if len(edge.Config) > 0 {
			plan.Steps = append(plan.Steps, fmt.Sprintf("Apply %d config migration(s) for %s -> %s", len(edge.Config), edge.From, edge.To))
		}
		if len(edge.Database) > 0 {
			plan.Steps = append(plan.Steps, fmt.Sprintf("Apply %d database migration(s) for %s -> %s", len(edge.Database), edge.From, edge.To))
		}
		for _, item := range edge.Compatibility {
			if _, ok := compatibilitySeen[item]; ok {
				continue
			}
			compatibilitySeen[item] = struct{}{}
			plan.Compatibility = append(plan.Compatibility, item)
		}
		for _, item := range edge.Rollback {
			if _, ok := rollbackSeen[item]; ok {
				continue
			}
			rollbackSeen[item] = struct{}{}
			plan.Rollback = append(plan.Rollback, item)
		}
	}

	plan.Steps = append(plan.Steps,
		"Run host-first convergence through acpctl host apply",
		"Validate post-upgrade health, smoke, and DR drill expectations",
		"Persist rollback metadata and upgrade artifacts",
	)
	return plan, nil
}
