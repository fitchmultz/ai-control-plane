// request_plan_test.go - Tests for canonical key request planning.
//
// Purpose:
//   - Verify shared key request planning stays deterministic across callers.
//
// Responsibilities:
//   - Cover default duration and model selection.
//   - Cover optional limit propagation.
//   - Cover alias override cloning for retry flows.
//
// Scope:
//   - Shared request-planning helpers only.
//
// Usage:
//   - Run with `go test ./internal/keygen`.
//
// Invariants/Assumptions:
//   - Tests avoid live gateway calls.
package keygen

import "testing"

func TestPlanGenerateRequestAppliesDefaultsAndRoleModels(t *testing.T) {
	t.Setenv("ACP_USER_ROLE", "")

	plan, err := PlanGenerateRequest(GenerateRequestConfig{
		Alias:  "demo-key",
		Budget: 12.5,
	})
	if err != nil {
		t.Fatalf("PlanGenerateRequest() error = %v", err)
	}

	if plan.Role != "developer" {
		t.Fatalf("plan role = %q, want developer", plan.Role)
	}
	if plan.Request.BudgetDuration != "30d" {
		t.Fatalf("duration = %q, want 30d", plan.Request.BudgetDuration)
	}
	if plan.Request.KeyAlias != "demo-key" {
		t.Fatalf("alias = %q, want demo-key", plan.Request.KeyAlias)
	}
	if len(plan.Models) != 2 {
		t.Fatalf("models = %v, want 2 developer models", plan.Models)
	}
	if len(plan.Request.Models) != len(plan.Models) {
		t.Fatalf("request models = %v, want %v", plan.Request.Models, plan.Models)
	}
}

func TestPlanGenerateRequestIncludesOptionalLimits(t *testing.T) {
	plan, err := PlanGenerateRequest(GenerateRequestConfig{
		Alias:    "lead-key",
		Budget:   25,
		Duration: "90d",
		Role:     "team-lead",
		RPM:      100,
		TPM:      2000,
		Parallel: 4,
	})
	if err != nil {
		t.Fatalf("PlanGenerateRequest() error = %v", err)
	}

	if plan.Role != "team-lead" {
		t.Fatalf("plan role = %q, want team-lead", plan.Role)
	}
	if plan.Request.RPMLimit != 100 || plan.Request.TPMLimit != 2000 || plan.Request.MaxParallelRequests != 4 {
		t.Fatalf("limits = %+v, want rpm=100 tpm=2000 parallel=4", plan.Request)
	}
	if got, want := len(plan.Models), 3; got != want {
		t.Fatalf("len(models) = %d, want %d", got, want)
	}
}

func TestPlanGenerateRequestRejectsInvalidInputs(t *testing.T) {
	if _, err := PlanGenerateRequest(GenerateRequestConfig{Alias: "bad alias", Budget: 1}); err == nil {
		t.Fatal("expected alias validation error")
	}
	if _, err := PlanGenerateRequest(GenerateRequestConfig{Alias: "good", Budget: 1, Role: "invalid"}); err == nil {
		t.Fatal("expected role validation error")
	}
}

func TestGenerateRequestPlanWithAliasClonesRequest(t *testing.T) {
	plan, err := PlanGenerateRequest(GenerateRequestConfig{
		Alias:  "demo-key",
		Budget: 5,
		Role:   "developer",
	})
	if err != nil {
		t.Fatalf("PlanGenerateRequest() error = %v", err)
	}

	retryPlan := plan.WithAlias("demo-key-2")
	if retryPlan.Request.KeyAlias != "demo-key-2" {
		t.Fatalf("retry alias = %q, want demo-key-2", retryPlan.Request.KeyAlias)
	}
	if plan.Request.KeyAlias != "demo-key" {
		t.Fatalf("original alias mutated to %q", plan.Request.KeyAlias)
	}
}
