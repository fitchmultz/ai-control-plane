// plan_test.go - Tests for explicit upgrade planning.
//
// Purpose:
//   - Verify typed upgrade path resolution and guardrails.
//
// Responsibilities:
//   - Cover explicit multi-hop path resolution.
//   - Reject unsupported paths and undeclared default-catalog upgrades.
//
// Scope:
//   - Upgrade planning tests only.
//
// Usage:
//   - Run via `go test ./internal/upgrade`.
//
// Invariants/Assumptions:
//   - Unsupported paths must fail until explicit edges are added.
package upgrade

import (
	"reflect"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/migration"
)

func TestBuildPlanResolvesExplicitMultiHopPath(t *testing.T) {
	catalog := migration.Catalog{
		Edges: []migration.ReleaseEdge{
			{From: "0.1.0", To: "0.2.0", Summary: "first hop"},
			{From: "0.2.0", To: "0.3.0", Summary: "second hop"},
		},
	}

	plan, err := BuildPlan("0.1.0", "0.3.0", catalog)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	want := []string{"0.1.0", "0.2.0", "0.3.0"}
	if !reflect.DeepEqual(plan.Path, want) {
		t.Fatalf("plan.Path = %v, want %v", plan.Path, want)
	}
	if len(plan.Edges) != 2 {
		t.Fatalf("len(plan.Edges) = %d, want 2", len(plan.Edges))
	}
}

func TestBuildPlanRejectsUnsupportedPath(t *testing.T) {
	catalog := migration.Catalog{
		Edges: []migration.ReleaseEdge{{From: "0.1.0", To: "0.2.0", Summary: "first hop"}},
	}

	if _, err := BuildPlan("0.1.0", "0.3.0", catalog); err == nil {
		t.Fatal("expected unsupported path error")
	}
}

func TestDefaultCatalogRejectsUndeclaredPaths(t *testing.T) {
	if _, err := BuildPlan("0.0.9", "0.1.0", DefaultCatalog()); err == nil {
		t.Fatal("expected undeclared default-catalog path to fail")
	}
}
