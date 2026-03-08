// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose embedded-database admin workflows such as backup and restore.
//
// Responsibilities:
//   - Enforce embedded-mode-only construction for destructive admin actions.
//   - Execute pg_dump and psql restore with bounded subprocess timeouts.
//
// Scope:
//   - Backup and restore workflows only.
//
// Usage:
//   - Construct via `NewAdminService(connector)` from db ops commands.
//
// Invariants/Assumptions:
//   - Backup and restore are unsupported for external database mode.
package db

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

// AdminService provides embedded-database backup and restore workflows.
type AdminService struct {
	connector *Connector
}

// NewAdminService creates an embedded-only admin service.
func NewAdminService(connector *Connector) (*AdminService, error) {
	if connector == nil {
		return nil, fmt.Errorf("database admin service requires a connector")
	}
	if err := connector.ConfigError(); err != nil {
		return nil, err
	}
	if connector.IsExternal() {
		return nil, fmt.Errorf("backup and restore are not supported for external database mode")
	}
	return &AdminService{connector: connector}, nil
}

// Backup creates a database backup.
func (s *AdminService) Backup(ctx context.Context) (string, error) {
	containerID, err := s.connector.containerID(ctx)
	if err != nil {
		return "", err
	}

	res := proc.Run(withTimeoutContext(ctx, 30*time.Second), proc.Request{
		Name: "docker",
		Args: []string{
			"exec", containerID,
			"pg_dump",
			"-U", s.connector.databaseUser(),
			"-d", s.connector.databaseName(),
			"-c",
			"-C",
			"-E", "UTF8",
			"--no-owner",
			"--no-acl",
		},
		Timeout: 30 * time.Second,
	})
	if res.Err != nil {
		return "", fmt.Errorf("database backup failed: %s", s.connector.execError("database backup", res))
	}
	return res.Stdout, nil
}

// Restore restores a database from a streamed SQL reader.
func (s *AdminService) Restore(ctx context.Context, sqlReader io.Reader) error {
	containerID, err := s.connector.containerID(ctx)
	if err != nil {
		return err
	}

	res := proc.Run(withTimeoutContext(ctx, 30*time.Second), proc.Request{
		Name:    "docker",
		Args:    []string{"exec", "-i", containerID, "psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", s.connector.databaseUser(), "-d", "postgres"},
		Stdin:   sqlReader,
		Timeout: 30 * time.Second,
	})
	if res.Err != nil {
		return fmt.Errorf("database restore failed: %s", s.connector.execError("database restore", res))
	}
	return nil
}
