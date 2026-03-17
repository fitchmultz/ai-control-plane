// Package migration provides typed config and database migration primitives.
//
// Purpose:
//   - Define explicit release-edge migration metadata and helpers.
//
// Responsibilities:
//   - Model version-to-version config and database migration steps.
//   - Resolve supported upgrade paths from explicit release edges only.
//   - Apply database SQL steps through a narrow typed executor.
//
// Scope:
//   - Migration catalog modeling and path resolution only.
//
// Usage:
//   - Imported by `internal/upgrade` to build upgrade and rollback plans.
//
// Invariants/Assumptions:
//   - Unsupported upgrade paths must fail until explicit edges are added.
//   - Multi-hop upgrades must execute every intermediate edge in order.
package migration

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
)

// SQLStep describes one explicit database migration statement.
type SQLStep struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	SQL     string `json:"sql"`
}

// ReleaseEdge describes one supported release-to-release upgrade edge.
type ReleaseEdge struct {
	From          string        `json:"from"`
	To            string        `json:"to"`
	Summary       string        `json:"summary"`
	Compatibility []string      `json:"compatibility"`
	Rollback      []string      `json:"rollback"`
	Config        []EnvMutation `json:"config,omitempty"`
	Database      []SQLStep     `json:"database,omitempty"`
}

// Catalog contains all explicitly supported upgrade edges.
type Catalog struct {
	Edges []ReleaseEdge
}

// ResolvePath resolves one explicit supported path through the catalog.
func (c Catalog) ResolvePath(from string, to string) ([]ReleaseEdge, error) {
	if from == to {
		return nil, nil
	}

	type node struct {
		version string
		path    []ReleaseEdge
	}

	queue := []node{{version: from}}
	visited := map[string]struct{}{from: {}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range c.Edges {
			if edge.From != current.version {
				continue
			}
			nextPath := append(append([]ReleaseEdge(nil), current.path...), edge)
			if edge.To == to {
				return nextPath, nil
			}
			if _, seen := visited[edge.To]; seen {
				continue
			}
			visited[edge.To] = struct{}{}
			queue = append(queue, node{version: edge.To, path: nextPath})
		}
	}

	return nil, fmt.Errorf("unsupported upgrade path: %s -> %s; add explicit release edges before shipping this path", from, to)
}

// ApplySQL executes each explicit SQL migration step in order.
func ApplySQL(ctx context.Context, exec db.SQLExecutor, databaseName string, steps []SQLStep) error {
	if exec == nil {
		return fmt.Errorf("database migration executor is required")
	}
	for _, step := range steps {
		if err := exec.Execute(ctx, databaseName, step.SQL); err != nil {
			return fmt.Errorf("apply database migration %s: %w", step.ID, err)
		}
	}
	return nil
}
