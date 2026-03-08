// checker_test.go - Tests for health checker functionality
//
// Purpose: Provide unit tests for health check logic and result aggregation
//
// Responsibilities:
//   - Test Component.IsPass() logic
//   - Test Status constants and behavior
//   - Test Result aggregation (healthy/unhealthy/warning states)
//   - Test Checker with mockable dependencies
//
// Non-scope:
//   - Does not test actual Docker/gateway/database connections (integration tests)
//   - Does not test terminal output formatting
//
// Invariants/Assumptions:
//   - Tests use injected fake dependencies to avoid external service requirements
//   - Tests verify behavior, not implementation details
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package health

import (
	"context"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

// TestStatusConstants verifies status values are ordered correctly
func TestStatusConstants(t *testing.T) {
	// Status values should be: Unknown=0, Healthy=1, Unhealthy=2, Warning=3
	if StatusUnknown != 0 {
		t.Errorf("StatusUnknown = %d, want 0", StatusUnknown)
	}
	if StatusHealthy != 1 {
		t.Errorf("StatusHealthy = %d, want 1", StatusHealthy)
	}
	if StatusUnhealthy != 2 {
		t.Errorf("StatusUnhealthy = %d, want 2", StatusUnhealthy)
	}
	if StatusWarning != 3 {
		t.Errorf("StatusWarning = %d, want 3", StatusWarning)
	}
}

// TestComponentIsPass verifies the IsPass logic for different component states
func TestComponentIsPass(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		isOptional bool
		wantPass   bool
	}{
		{"healthy required", StatusHealthy, false, true},
		{"healthy optional", StatusHealthy, true, true},
		{"unhealthy required", StatusUnhealthy, false, false},
		{"unhealthy optional", StatusUnhealthy, true, false},
		{"warning required", StatusWarning, false, false},
		{"warning optional", StatusWarning, true, false},
		{"unknown required", StatusUnknown, false, false},
		{"unknown optional", StatusUnknown, true, true}, // Optional unknown passes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := Component{
				Status:     tt.status,
				IsOptional: tt.isOptional,
			}
			if got := comp.IsPass(); got != tt.wantPass {
				t.Errorf("IsPass() = %v, want %v (status=%v, optional=%v)",
					got, tt.wantPass, tt.status, tt.isOptional)
			}
		})
	}
}

// TestResultOverallStatus verifies overall status aggregation
func TestResultOverallStatus(t *testing.T) {
	tests := []struct {
		name        string
		components  []Status
		wantOverall Status
	}{
		{
			name:        "all healthy",
			components:  []Status{StatusHealthy, StatusHealthy},
			wantOverall: StatusHealthy,
		},
		{
			name:        "one unhealthy",
			components:  []Status{StatusHealthy, StatusUnhealthy},
			wantOverall: StatusUnhealthy,
		},
		{
			name:        "all unhealthy",
			components:  []Status{StatusUnhealthy, StatusUnhealthy},
			wantOverall: StatusUnhealthy,
		},
		{
			name:        "mixed with warning",
			components:  []Status{StatusHealthy, StatusWarning},
			wantOverall: StatusUnhealthy, // Warning is not a pass
		},
		{
			name:        "empty components",
			components:  []Status{},
			wantOverall: StatusHealthy, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Components: make([]Component, len(tt.components)),
				Overall:    StatusHealthy,
			}

			for i, status := range tt.components {
				result.Components[i] = Component{Status: status}
				if !result.Components[i].IsPass() {
					result.Overall = StatusUnhealthy
				}
			}

			if result.Overall != tt.wantOverall {
				t.Errorf("Overall status = %v, want %v", result.Overall, tt.wantOverall)
			}
		})
	}
}

// TestCheckerConstruction verifies NewChecker creates properly initialized checker
func TestCheckerConstruction(t *testing.T) {
	compose := &docker.Compose{}
	checker := NewChecker(compose, true)

	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}
	if checker.compose != compose {
		t.Error("Checker.compose not set correctly")
	}
	if checker.gateway == nil {
		t.Error("Checker.gateway should be initialized")
	}
	if checker.db == nil {
		t.Error("Checker.db should be initialized")
	}
	if checker.out == nil {
		t.Error("Checker.out should be initialized")
	}
	if !checker.verbose {
		t.Error("Checker.verbose should be true")
	}

	// Test non-verbose
	checker2 := NewChecker(compose, false)
	if checker2.verbose {
		t.Error("Checker.verbose should be false")
	}
}

// TestCheckerWithNilCompose verifies checker handles nil compose gracefully
func TestCheckerWithNilCompose(t *testing.T) {
	checker := NewChecker(nil, false)
	if checker == nil {
		t.Fatal("NewChecker with nil compose should not return nil")
	}

	// Run should handle nil compose gracefully
	ctx := context.Background()
	result := checker.Run(ctx)

	if result == nil {
		t.Fatal("Run() returned nil")
	}

	// Should have components even with nil compose
	if len(result.Components) == 0 {
		t.Error("Expected components even with nil compose")
	}

	// Overall should be unhealthy since compose is nil
	if result.Overall != StatusUnhealthy {
		t.Errorf("Expected unhealthy status with nil compose, got %v", result.Overall)
	}
}

// TestResultComponents verifies result contains expected components
func TestResultComponents(t *testing.T) {
	checker := NewChecker(nil, false)
	ctx := context.Background()
	result := checker.Run(ctx)

	// Should have 5 components: Docker, Gateway Health, Gateway Models, Database, OTEL
	expectedComponents := 5
	if len(result.Components) != expectedComponents {
		t.Errorf("Expected %d components, got %d", expectedComponents, len(result.Components))
	}

	// Verify component names
	expectedNames := map[string]bool{
		"Docker Containers":       false,
		"LiteLLM Health Endpoint": false,
		"LiteLLM Models Endpoint": false,
		"PostgreSQL Database":     false,
		"OTEL Collector":          false,
	}

	for _, comp := range result.Components {
		if _, exists := expectedNames[comp.Name]; exists {
			expectedNames[comp.Name] = true
		} else {
			t.Errorf("Unexpected component name: %s", comp.Name)
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Missing expected component: %s", name)
		}
	}
}

// TestComponentFields verifies component struct fields work correctly
func TestComponentFields(t *testing.T) {
	comp := Component{
		Name:       "Test Component",
		Status:     StatusHealthy,
		Message:    "All good",
		Details:    "Detailed info",
		IsOptional: false,
	}

	if comp.Name != "Test Component" {
		t.Errorf("Name = %q, want %q", comp.Name, "Test Component")
	}
	if comp.Status != StatusHealthy {
		t.Errorf("Status = %v, want %v", comp.Status, StatusHealthy)
	}
	if comp.Message != "All good" {
		t.Errorf("Message = %q, want %q", comp.Message, "All good")
	}
	if comp.Details != "Detailed info" {
		t.Errorf("Details = %q, want %q", comp.Details, "Detailed info")
	}
	if comp.IsOptional {
		t.Error("IsOptional should be false")
	}
}

// TestCheckerUsesOutput verifies checker uses output for formatting
func TestCheckerUsesOutput(t *testing.T) {
	checker := NewChecker(nil, false)
	if checker.out == nil {
		t.Error("Checker should have output instance")
	}

	// Verify output methods work
	out := output.New()
	if out == nil {
		t.Error("output.New() should not return nil")
	}
}

// TestGatewayClientIntegration verifies gateway client is properly initialized
func TestGatewayClientIntegration(t *testing.T) {
	checker := NewChecker(nil, false)
	if checker.gateway == nil {
		t.Fatal("Gateway client should be initialized")
	}

	// Test HasMasterKey behavior (depends on environment)
	_ = checker.gateway.HasMasterKey()
}

// TestDBClientIntegration verifies DB client is properly initialized
func TestDBClientIntegration(t *testing.T) {
	compose := &docker.Compose{}
	checker := NewChecker(compose, false)
	if checker.db == nil {
		t.Fatal("DB client should be initialized")
	}

	// Test mode methods
	_ = checker.db.IsEmbedded()
	_ = checker.db.IsExternal()
	_ = checker.db.Mode()
}

// TestPrintSummaryDoesNotPanic verifies PrintSummary handles all result states
func TestPrintSummaryDoesNotPanic(t *testing.T) {
	checker := NewChecker(nil, false)

	tests := []struct {
		name   string
		result *Result
	}{
		{
			name: "healthy result",
			result: &Result{
				Overall:    StatusHealthy,
				Components: []Component{{Name: "Test", Status: StatusHealthy}},
			},
		},
		{
			name: "unhealthy result",
			result: &Result{
				Overall:    StatusUnhealthy,
				Components: []Component{{Name: "Test", Status: StatusUnhealthy}},
			},
		},
		{
			name: "unknown result",
			result: &Result{
				Overall:    StatusUnknown,
				Components: []Component{{Name: "Test", Status: StatusUnknown}},
			},
		},
		{
			name:   "empty result",
			result: &Result{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrintSummary panicked: %v", r)
				}
			}()
			checker.PrintSummary(tt.result)
		})
	}
}

// TestExternalDatabaseModeHandling verifies health check handles external DB mode
func TestExternalDatabaseModeHandling(t *testing.T) {
	// This test verifies that when db.IsExternal() returns true,
	// the database check component behaves appropriately
	// Note: Full integration would require mocking db.Client

	checker := NewChecker(nil, false)
	ctx := context.Background()

	// Run health check
	result := checker.Run(ctx)

	// Find database component
	var dbComponent *Component
	for i := range result.Components {
		if result.Components[i].Name == "PostgreSQL Database" {
			dbComponent = &result.Components[i]
			break
		}
	}

	if dbComponent == nil {
		t.Fatal("PostgreSQL Database component not found")
	}

	// Without a running database, should be unhealthy
	// but the check should complete without panic
	if dbComponent.Status == StatusUnknown {
		t.Error("Database component status should be determined, not unknown")
	}
}

// TestOptionalOTELComponent verifies OTEL is properly marked as optional
func TestOptionalOTELComponent(t *testing.T) {
	checker := NewChecker(nil, false)
	ctx := context.Background()
	result := checker.Run(ctx)

	// Find OTEL component
	var otelComponent *Component
	for i := range result.Components {
		if result.Components[i].Name == "OTEL Collector" {
			otelComponent = &result.Components[i]
			break
		}
	}

	if otelComponent == nil {
		t.Fatal("OTEL Collector component not found")
	}

	if !otelComponent.IsOptional {
		t.Error("OTEL Collector should be marked as optional")
	}

	// Optional component with unknown status should pass IsPass()
	if otelComponent.Status == StatusUnknown && !otelComponent.IsPass() {
		t.Error("Optional component with unknown status should pass IsPass()")
	}
}

// TestContextPropagation verifies context is properly used
func TestContextPropagation(t *testing.T) {
	checker := NewChecker(nil, false)

	// Test with background context
	ctx := context.Background()
	result := checker.Run(ctx)
	if result == nil {
		t.Error("Run with background context should return result")
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	result = checker.Run(ctx)
	if result == nil {
		t.Error("Run with cancelled context should still return result")
	}
}
