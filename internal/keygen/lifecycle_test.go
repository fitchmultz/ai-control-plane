// lifecycle_test.go - Tests for typed key lifecycle workflows.
//
// Purpose:
//   - Verify inspection and rotation flows stay deterministic.
//
// Responsibilities:
//   - Cover report-month resolution, inspection lookup, and rotation staging.
//   - Verify optional revocation and model-preserving replacement planning.
//
// Scope:
//   - Key lifecycle planning only.
//
// Usage:
//   - Run via `go test ./internal/keygen`.
//
// Invariants/Assumptions:
//   - Tests use fake inventory and usage stores instead of live services.
package keygen

import (
	"context"
	"testing"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/gateway"
)

type fakeInventory struct {
	keys      []gateway.KeyInfo
	generated *gateway.GenerateKeyResponse
	revoked   []string
}

func (f *fakeInventory) ListKeys(context.Context) ([]gateway.KeyInfo, error) {
	return f.keys, nil
}

func (f *fakeInventory) GenerateKey(context.Context, *gateway.GenerateKeyRequest) (*gateway.GenerateKeyResponse, error) {
	if f.generated == nil {
		f.generated = &gateway.GenerateKeyResponse{Key: "sk-new", KeyAlias: "generated"}
	}
	return f.generated, nil
}

func (f *fakeInventory) DeleteKey(_ context.Context, alias string) error {
	f.revoked = append(f.revoked, alias)
	return nil
}

type fakeUsageStore struct {
	usage KeyUsage
}

func (f fakeUsageStore) KeyUsage(context.Context, string, MonthWindow) (KeyUsage, error) {
	return f.usage, nil
}

func TestResolveMonthWindowDefaultsToCurrentMonth(t *testing.T) {
	window, err := ResolveMonthWindow("", time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ResolveMonthWindow() error = %v", err)
	}
	if window.ReportMonth != "2026-03" || window.Start != "2026-03-01" || window.End != "2026-03-31" {
		t.Fatalf("unexpected month window: %+v", window)
	}
}

func TestInspectKey(t *testing.T) {
	inventory := &fakeInventory{
		keys: []gateway.KeyInfo{{KeyAlias: "alice", MaxBudget: 10, Models: []string{"openai-gpt5.2", "claude-haiku-4-5"}}},
	}
	inspection, err := InspectKey(context.Background(), inventory, fakeUsageStore{
		usage: KeyUsage{Alias: "alice", TotalSpend: 4.2, TotalRequests: 7, TotalTokens: 99},
	}, "alice", "2026-02", time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("InspectKey() error = %v", err)
	}
	if inspection.Key.Alias() != "alice" || inspection.Usage.TotalSpend != 4.2 || inspection.Usage.ReportMonth != "2026-02" {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
}

func TestRotateKeyStagesCutoverAndOptionalRevoke(t *testing.T) {
	inventory := &fakeInventory{
		keys: []gateway.KeyInfo{{
			KeyAlias:            "alice",
			MaxBudget:           10,
			BudgetDuration:      "30d",
			Models:              []string{"openai-gpt5.2", "claude-haiku-4-5"},
			RPMLimit:            100,
			TPMLimit:            1000,
			MaxParallelRequests: 3,
		}},
		generated: &gateway.GenerateKeyResponse{Key: "sk-new", KeyAlias: "alice-rotated"},
	}

	result, err := RotateKey(context.Background(), inventory, fakeUsageStore{
		usage: KeyUsage{Alias: "alice", TotalSpend: 8.1},
	}, RotationRequest{
		SourceAlias:      "alice",
		ReplacementAlias: "alice-rotated",
		RevokeOld:        true,
	}, time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}
	if result.Replacement == nil || !result.RevokedOld {
		t.Fatalf("unexpected rotation result: %+v", result)
	}
	if len(inventory.revoked) != 1 || inventory.revoked[0] != "alice" {
		t.Fatalf("unexpected revoked aliases: %+v", inventory.revoked)
	}
	if result.ReplacementPlan.Request.RPMLimit != 100 || result.ReplacementPlan.Request.MaxParallelRequests != 3 {
		t.Fatalf("replacement plan did not preserve limits: %+v", result.ReplacementPlan)
	}
	if got, want := result.ReplacementPlan.Request.Models, inventory.keys[0].Models; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("replacement plan did not preserve models: %+v", result.ReplacementPlan.Request.Models)
	}
}

func TestRotateKeyDryRunUsesTimestampedReplacementAlias(t *testing.T) {
	inventory := &fakeInventory{
		keys: []gateway.KeyInfo{{KeyAlias: "demo", MaxBudget: 5, Models: []string{"openai-gpt5.2"}}},
	}

	result, err := RotateKey(context.Background(), inventory, fakeUsageStore{}, RotationRequest{
		SourceAlias: "demo",
		DryRun:      true,
	}, time.Date(2026, 3, 7, 9, 8, 7, 0, time.UTC))
	if err != nil {
		t.Fatalf("RotateKey() error = %v", err)
	}
	if result.Replacement != nil || result.RevokedOld {
		t.Fatalf("expected dry run result without side effects: %+v", result)
	}
	if result.ReplacementPlan.Request.KeyAlias != "demo-rotated-20260307090807" {
		t.Fatalf("unexpected replacement alias: %q", result.ReplacementPlan.Request.KeyAlias)
	}
}
