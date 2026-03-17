// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose a narrow typed database migration execution service for the
//   - upgrade framework.
//
// Responsibilities:
//   - Validate migration service construction.
//   - Execute explicit SQL migration text against a selected database.
//
// Scope:
//   - Typed migration execution only.
//
// Usage:
//   - Construct via `NewMigrationService(connector)` from upgrade workflows.
//
// Invariants/Assumptions:
//   - Migration SQL is declared explicitly by tracked release edges.
//   - Generic ad hoc SQL execution is not exposed on user-facing command paths.
package db

import (
	"context"
	"fmt"
	"strings"
)

// MigrationService executes explicit release-declared SQL migrations.
type MigrationService struct {
	connector *Connector
}

// NewMigrationService creates a migration execution service.
func NewMigrationService(connector *Connector) *MigrationService {
	return &MigrationService{connector: connector}
}

// Execute runs migration SQL against the requested database.
func (s *MigrationService) Execute(ctx context.Context, databaseName string, sqlText string) error {
	if s == nil || s.connector == nil {
		return fmt.Errorf("database migration service requires a connector")
	}
	if strings.TrimSpace(sqlText) == "" {
		return fmt.Errorf("migration SQL is required")
	}
	return s.connector.execSQLInDatabase(ctx, databaseName, sqlText)
}
