// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Own runtime connector setup for embedded and external PostgreSQL modes.
//
// Responsibilities:
//   - Resolve typed database settings from internal/config.
//   - Manage embedded Docker container lookup and external sql.DB lifecycle.
//   - Execute bounded scalar queries behind typed higher-level services.
//
// Scope:
//   - Shared connector/runtime internals only.
//
// Usage:
//   - Construct via `NewConnector(repoRoot)` and pass to service constructors.
//
// Invariants/Assumptions:
//   - Embedded mode uses the repo-local Compose postgres service.
//   - External mode uses DATABASE_URL and validates connectivity explicitly.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/proc"

	_ "github.com/lib/pq"
)

// Connector owns database runtime configuration and low-level connectivity.
type Connector struct {
	settings  config.DatabaseSettings
	compose   *docker.Compose
	container string
	sqlDB     *sql.DB
}

// NewConnector creates a new database connector.
func NewConnector(repoRoot string) *Connector {
	settings := config.NewLoader().WithRepoRoot(repoRoot).Database(context.Background())
	connector := &Connector{settings: settings}
	if settings.Mode.IsEmbedded() && repoRoot != "" {
		compose, err := docker.NewACPCompose(repoRoot, nil)
		if err == nil {
			connector.compose = compose
		}
	}
	return connector
}

// Close closes the external database connection if one was opened.
func (c *Connector) Close() error {
	if c == nil || c.sqlDB == nil {
		return nil
	}
	return c.sqlDB.Close()
}

// ConfigError returns the configuration error detected during connector creation.
func (c *Connector) ConfigError() error {
	if c == nil {
		return fmt.Errorf("database connector is required")
	}
	return c.settings.AmbiguousErr
}

// Mode returns the effective database mode.
func (c *Connector) Mode() config.DatabaseMode {
	if c == nil {
		return ""
	}
	return c.settings.Mode
}

// IsEmbedded reports whether the connector is in embedded mode.
func (c *Connector) IsEmbedded() bool {
	return c.Mode().IsEmbedded()
}

// IsExternal reports whether the connector is in external mode.
func (c *Connector) IsExternal() bool {
	return c.Mode().IsExternal()
}

func (c *Connector) databaseName() string {
	if c != nil && strings.TrimSpace(c.settings.Name) != "" {
		return c.settings.Name
	}
	return defaultDBName
}

func (c *Connector) databaseUser() string {
	if c != nil && strings.TrimSpace(c.settings.User) != "" {
		return c.settings.User
	}
	return defaultDBUser
}

func (c *Connector) ping(ctx context.Context) Probe {
	start := time.Now()
	if err := c.ConfigError(); err != nil {
		return Probe{
			Operation: "ping",
			Error:     err.Error(),
			Latency:   time.Since(start).Round(time.Millisecond),
		}
	}

	if c.IsExternal() {
		dbConn, err := c.ensureSQLDB(ctx)
		if err != nil {
			return Probe{Operation: "ping", Error: err.Error(), Latency: time.Since(start).Round(time.Millisecond)}
		}
		pingCtx, cancel := withTimeoutContext(ctx, queryTimeout)
		defer cancel()
		err = dbConn.PingContext(pingCtx)
		if err != nil {
			return Probe{
				Operation: "ping",
				Error:     fmt.Sprintf("database ping failed: %v", err),
				Latency:   time.Since(start).Round(time.Millisecond),
			}
		}
		return Probe{Operation: "ping", Healthy: true, Latency: time.Since(start).Round(time.Millisecond)}
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return Probe{Operation: "ping", Error: err.Error(), Latency: time.Since(start).Round(time.Millisecond)}
	}

	runCtx, cancel := withTimeoutContext(ctx, queryTimeout)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
		Name:    "docker",
		Args:    []string{"exec", containerID, "pg_isready", "-U", c.databaseUser(), "-d", c.databaseName()},
		Timeout: queryTimeout,
	})
	if res.Err != nil {
		return Probe{
			Operation: "ping",
			Error:     c.execError("database ping", res),
			Latency:   time.Since(start).Round(time.Millisecond),
		}
	}

	return Probe{Operation: "ping", Healthy: true, Latency: time.Since(start).Round(time.Millisecond)}
}

func (c *Connector) ensureSQLDB(ctx context.Context) (*sql.DB, error) {
	if c.sqlDB != nil {
		return c.sqlDB, nil
	}
	if strings.TrimSpace(c.settings.URL) == "" {
		return nil, fmt.Errorf("DATABASE_URL not set for external database mode")
	}
	dbConn, err := sql.Open("postgres", c.settings.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	pingCtx, cancel := withTimeoutContext(ctx, queryTimeout)
	defer cancel()
	if err := dbConn.PingContext(pingCtx); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	c.sqlDB = dbConn
	return c.sqlDB, nil
}

func (c *Connector) containerID(ctx context.Context) (string, error) {
	if c.container != "" {
		return c.container, nil
	}
	if c.compose == nil {
		return "", fmt.Errorf("database runtime requires docker compose for embedded mode")
	}
	lookupCtx, cancel := withTimeoutContext(ctx, containerLookupTimeout)
	defer cancel()
	containerID, err := c.compose.ContainerID(lookupCtx, "postgres")
	if err != nil {
		return "", fmt.Errorf("database container lookup failed: %w", err)
	}
	c.container = strings.TrimSpace(containerID)
	return c.container, nil
}

func (c *Connector) scalarString(ctx context.Context, query string) (string, error) {
	return c.scalarStringInDatabase(ctx, c.databaseName(), query)
}

func (c *Connector) scalarInt(ctx context.Context, query string) (int, error) {
	return c.scalarIntInDatabase(ctx, c.databaseName(), query)
}

func (c *Connector) scalarStringInDatabase(ctx context.Context, databaseName string, query string) (string, error) {
	if err := c.ConfigError(); err != nil {
		return "", err
	}
	resolvedDatabaseName := strings.TrimSpace(databaseName)
	if resolvedDatabaseName == "" {
		resolvedDatabaseName = c.databaseName()
	}
	if c.IsExternal() && resolvedDatabaseName != c.databaseName() {
		return "", fmt.Errorf("cross-database queries are unsupported for external database mode")
	}
	if c.IsExternal() {
		dbConn, err := c.ensureSQLDB(ctx)
		if err != nil {
			return "", err
		}
		var value sql.NullString
		queryCtx, cancel := withTimeoutContext(ctx, queryTimeout)
		defer cancel()
		err = dbConn.QueryRowContext(queryCtx, query).Scan(&value)
		if err != nil {
			return "", fmt.Errorf("database query failed: %w", err)
		}
		return strings.TrimSpace(value.String), nil
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return "", err
	}
	runCtx, cancel := withTimeoutContext(ctx, queryTimeout)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
		Name:    "docker",
		Args:    []string{"exec", containerID, "psql", "-X", "-A", "-t", "-U", c.databaseUser(), "-d", resolvedDatabaseName, "-c", query},
		Timeout: queryTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("database query failed: %s", c.execError("database query", res))
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Connector) scalarIntInDatabase(ctx context.Context, databaseName string, query string) (int, error) {
	value, err := c.scalarStringInDatabase(ctx, databaseName, query)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("database query parse failed: %w", err)
	}
	return parsed, nil
}

func (c *Connector) execSQLInDatabase(ctx context.Context, databaseName string, sqlText string) error {
	if err := c.ConfigError(); err != nil {
		return err
	}

	resolvedDatabaseName := strings.TrimSpace(databaseName)
	if resolvedDatabaseName == "" {
		resolvedDatabaseName = c.databaseName()
	}

	if c.IsExternal() {
		if resolvedDatabaseName != c.databaseName() {
			return fmt.Errorf("cross-database queries are unsupported for external database mode")
		}
		dbConn, err := c.ensureSQLDB(ctx)
		if err != nil {
			return err
		}
		execCtx, cancel := withTimeoutContext(ctx, 5*time.Minute)
		defer cancel()
		if _, err := dbConn.ExecContext(execCtx, sqlText); err != nil {
			return fmt.Errorf("database statement failed: %w", err)
		}
		return nil
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return err
	}
	runCtx, cancel := withTimeoutContext(ctx, 5*time.Minute)
	defer cancel()
	res := proc.Run(runCtx, proc.Request{
		Name: "docker",
		Args: []string{
			"exec", containerID,
			"psql", "-X", "-v", "ON_ERROR_STOP=1",
			"-U", c.databaseUser(),
			"-d", resolvedDatabaseName,
			"-c", sqlText,
		},
		Timeout: 5 * time.Minute,
	})
	if res.Err != nil {
		return fmt.Errorf("database statement failed: %s", c.execError("database statement", res))
	}
	return nil
}

func (c *Connector) execError(prefix string, res proc.Result) string {
	if res.Stderr != "" {
		return fmt.Sprintf("%s: %s", prefix, strings.TrimSpace(res.Stderr))
	}
	if res.Err != nil {
		return fmt.Sprintf("%s: %v", prefix, res.Err)
	}
	return prefix
}

func withTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
