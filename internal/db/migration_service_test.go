// migration_service_test.go - Tests for typed migration service validation.
//
// Purpose:
//   - Cover nil-guard and input-validation paths for migration execution.
//
// Responsibilities:
//   - Verify construction keeps the provided connector reference.
//   - Reject nil connectors and empty SQL before any database access.
//
// Scope:
//   - Validation-only coverage for internal/db migration helpers.
//
// Usage:
//   - Run via `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Tests remain local-only and do not require a live PostgreSQL instance.
package db

import (
	"context"
	"strings"
	"testing"
)

func TestNewMigrationServiceStoresConnector(t *testing.T) {
	connector := &Connector{}
	service := NewMigrationService(connector)
	if service == nil {
		t.Fatal("expected service")
	}
	if service.connector != connector {
		t.Fatal("expected connector to be retained")
	}
}

func TestMigrationServiceExecuteRejectsMissingConnector(t *testing.T) {
	service := &MigrationService{}
	err := service.Execute(context.Background(), "postgres", "SELECT 1;")
	if err == nil || !strings.Contains(err.Error(), "requires a connector") {
		t.Fatalf("expected missing connector error, got %v", err)
	}
}

func TestMigrationServiceExecuteRejectsEmptySQL(t *testing.T) {
	service := &MigrationService{connector: &Connector{}}
	err := service.Execute(context.Background(), "postgres", "   ")
	if err == nil || !strings.Contains(err.Error(), "migration SQL is required") {
		t.Fatalf("expected empty SQL error, got %v", err)
	}
}
