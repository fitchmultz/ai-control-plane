// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose embedded-database admin workflows such as backup, restore,
//     retention-supporting verification, and restore-drill helpers.
//
// Responsibilities:
//   - Enforce embedded-mode-only construction for destructive admin actions.
//   - Execute pg_dump and psql restore with bounded subprocess timeouts.
//   - Support scratch-database restore verification without touching production.
//
// Scope:
//   - Backup, restore, and restore-verification workflows only.
//
// Usage:
//   - Construct via `NewAdminService(connector)` from db ops commands.
//
// Invariants/Assumptions:
//   - Backup and restore are unsupported for external database mode.
package db

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const (
	backupCommandTimeout  = 2 * time.Minute
	restoreCommandTimeout = 5 * time.Minute
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

	runCtx, cancel := withTimeoutContext(ctx, backupCommandTimeout)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
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
		Timeout: backupCommandTimeout,
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

	runCtx, cancel := withTimeoutContext(ctx, restoreCommandTimeout)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
		Name:    "docker",
		Args:    []string{"exec", "-i", containerID, "psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", s.connector.databaseUser(), "-d", "postgres"},
		Stdin:   sqlReader,
		Timeout: restoreCommandTimeout,
	})
	if res.Err != nil {
		return fmt.Errorf("database restore failed: %s", s.connector.execError("database restore", res))
	}
	return nil
}

// RewriteBackupForScratchDatabase rewrites pg_dump database control statements for a scratch database.
func (s *AdminService) RewriteBackupForScratchDatabase(sql string, scratchDatabase string) (string, error) {
	return rewriteBackupDatabaseName(sql, s.connector.databaseName(), scratchDatabase)
}

// DropDatabaseIfExists force-drops a scratch database after terminating active sessions.
func (s *AdminService) DropDatabaseIfExists(ctx context.Context, databaseName string) error {
	containerID, err := s.connector.containerID(ctx)
	if err != nil {
		return err
	}
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return fmt.Errorf("scratch database name is required")
	}

	runCtx, cancel := withTimeoutContext(ctx, restoreCommandTimeout)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
		Name:    "docker",
		Args:    []string{"exec", containerID, "dropdb", "-U", s.connector.databaseUser(), "--if-exists", "--force", databaseName},
		Timeout: restoreCommandTimeout,
	})
	if res.Err != nil {
		return fmt.Errorf("drop scratch database failed: %s", s.connector.execError("drop scratch database", res))
	}
	return nil
}

// VerifyCoreSchema validates that the canonical LiteLLM core schema exists in the target database.
func (s *AdminService) VerifyCoreSchema(ctx context.Context, databaseName string) (RestoreVerification, error) {
	found, err := s.connector.scalarIntInDatabase(ctx, databaseName, coreSchemaTableCountQuery())
	if err != nil {
		return RestoreVerification{}, err
	}

	version, err := s.connector.scalarStringInDatabase(ctx, databaseName, `SELECT version();`)
	if err != nil {
		return RestoreVerification{}, err
	}

	return RestoreVerification{
		DatabaseName:   strings.TrimSpace(databaseName),
		ExpectedTables: ExpectedCoreSchemaTableCount(),
		FoundTables:    found,
		Version:        strings.TrimSpace(version),
	}, nil
}

func rewriteBackupDatabaseName(sql string, sourceDatabase string, targetDatabase string) (string, error) {
	sourceDatabase = strings.TrimSpace(sourceDatabase)
	targetDatabase = strings.TrimSpace(targetDatabase)
	if sourceDatabase == "" {
		return "", fmt.Errorf("source database name is required")
	}
	if targetDatabase == "" {
		return "", fmt.Errorf("target scratch database name is required")
	}

	reader := bufio.NewReader(strings.NewReader(sql))
	var out strings.Builder
	rewroteCreate := false
	rewroteConnect := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("read backup SQL: %w", err)
		}

		hasNewline := strings.HasSuffix(line, "\n")
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")

		rewritten, created, connected := rewriteBackupControlLine(line, targetDatabase)
		rewroteCreate = rewroteCreate || created
		rewroteConnect = rewroteConnect || connected

		out.WriteString(rewritten)
		if hasNewline {
			out.WriteByte('\n')
		}

		if err == io.EOF {
			break
		}
	}

	if !rewroteCreate || !rewroteConnect {
		return "", fmt.Errorf("backup SQL for %q is missing database control statements", sourceDatabase)
	}
	return out.String(), nil
}

func rewriteBackupControlLine(line string, targetDatabase string) (string, bool, bool) {
	switch {
	case strings.HasPrefix(line, "DROP DATABASE IF EXISTS "):
		return replaceDatabaseToken(line, "DROP DATABASE IF EXISTS ", targetDatabase), false, false
	case strings.HasPrefix(line, "DROP DATABASE "):
		rewritten := replaceDatabaseToken(line, "DROP DATABASE ", targetDatabase)
		return strings.Replace(rewritten, "DROP DATABASE ", "DROP DATABASE IF EXISTS ", 1), false, false
	case strings.HasPrefix(line, "CREATE DATABASE "):
		return replaceDatabaseToken(line, "CREATE DATABASE ", targetDatabase), true, false
	case strings.HasPrefix(line, "ALTER DATABASE "):
		return replaceDatabaseToken(line, "ALTER DATABASE ", targetDatabase), false, false
	case strings.HasPrefix(line, `\connect `):
		return replaceConnectTarget(line, targetDatabase), false, true
	default:
		return line, false, false
	}
}

func replaceDatabaseToken(line string, prefix string, targetDatabase string) string {
	rest := strings.TrimPrefix(line, prefix)
	index := strings.IndexAny(rest, " \t;")
	if index < 0 {
		return prefix + targetDatabase
	}
	token := rest[:index]
	suffix := rest[index:]
	if strings.HasPrefix(token, `"`) && strings.HasSuffix(token, `"`) {
		return prefix + quoteSQLIdentifier(targetDatabase) + suffix
	}
	return prefix + targetDatabase + suffix
}

func replaceConnectTarget(line string, targetDatabase string) string {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return line
	}
	parts[1] = targetDatabase
	return strings.Join(parts, " ")
}

func quoteSQLIdentifier(value string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(value), `"`, `""`) + `"`
}
