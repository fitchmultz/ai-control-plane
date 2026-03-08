// Package runtimeinspect composes the canonical ACP runtime inspection stack.
//
// Purpose:
//   - Own the shared runtime inspection composition used by CLI workflows.
//
// Responsibilities:
//   - Construct default gateway/database-backed collectors.
//   - Expose a reusable inspector lifecycle around shared dependencies.
//   - Evaluate readiness from the canonical runtime inspection report.
//
// Scope:
//   - Runtime inspection composition and readiness helpers only.
//
// Usage:
//   - Construct via `NewInspector(repoRoot)` and call `Collect(ctx, opts)`.
//
// Invariants/Assumptions:
//   - One inspector instance reuses a single database connector across services.
package runtimeinspect

import (
	"context"
	"slices"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/collectors"
)

// DefaultReadinessComponents defines the canonical runtime readiness gate.
var DefaultReadinessComponents = []string{"gateway", "database"}

// Inspector owns the shared runtime inspection dependencies.
type Inspector struct {
	repoRoot  string
	connector *db.Connector
	gateway   gateway.StatusReader
	runtime   db.RuntimeServiceReader
	readonly  db.ReadonlyServiceReader
}

// Readiness describes whether required runtime components are ready.
type Readiness struct {
	Ready   bool
	Missing []string
	Pending map[string]status.ComponentStatus
}

// NewInspector constructs the canonical runtime inspection stack.
func NewInspector(repoRoot string) *Inspector {
	connector := db.NewConnector(repoRoot)
	return &Inspector{
		repoRoot:  repoRoot,
		connector: connector,
		gateway:   gateway.NewClient(),
		runtime:   db.NewRuntimeService(connector),
		readonly:  db.NewReadonlyService(connector),
	}
}

// Close releases shared dependencies.
func (i *Inspector) Close() error {
	if i == nil || i.connector == nil {
		return nil
	}
	return i.connector.Close()
}

// Collect runs the canonical runtime collectors.
func (i *Inspector) Collect(ctx context.Context, opts status.Options) status.StatusReport {
	return status.CollectAll(ctx, []status.Collector{
		collectors.NewGatewayCollector(i.gateway),
		collectors.NewDatabaseCollector(i.runtime),
		collectors.NewKeysCollector(i.readonly),
		collectors.NewBudgetCollector(i.readonly),
		collectors.NewDetectionsCollector(i.repoRoot, i.readonly),
	}, opts)
}

// EvaluateReadiness returns readiness for the required component names.
func EvaluateReadiness(report status.StatusReport, required []string) Readiness {
	missing := make([]string, 0)
	pending := make(map[string]status.ComponentStatus)
	for _, name := range required {
		component, ok := report.Components[name]
		if !ok || component.Level != status.HealthLevelHealthy {
			missing = append(missing, name)
			if ok {
				pending[name] = component
			}
		}
	}
	slices.Sort(missing)
	return Readiness{
		Ready:   len(missing) == 0,
		Missing: missing,
		Pending: pending,
	}
}
