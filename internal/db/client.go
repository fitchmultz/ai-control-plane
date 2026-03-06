// Package db provides PostgreSQL database operations.
//
// Purpose:
//
//	Provide database connectivity and query execution for the AI Control Plane
//	PostgreSQL database, supporting both embedded and external modes.
//
// Responsibilities:
//   - Database connection string parsing
//   - Query execution via docker exec or direct connection
//   - Schema validation (table existence checks)
//   - Backup and restore operations
//
// Non-scope:
//   - Does not manage database migrations
//   - Does not handle connection pooling for production workloads
//
// Invariants/Assumptions:
//   - Embedded mode uses Docker container
//   - External mode uses DATABASE_URL environment variable
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/docker"

	// PostgreSQL driver for external database mode
	_ "github.com/lib/pq"
)

const (
	defaultDBName              = "litellm"
	defaultDBUser              = "litellm"
	defaultEmbeddedDatabaseURL = "postgresql://litellm:litellm@postgres:5432/litellm"
)

// Client provides database operations
type Client struct {
	mode        string // "embedded" or "external"
	dbName      string
	dbUser      string
	containerID string
	compose     *docker.Compose
	configErr   error
	db          *sql.DB // External database connection (nil for embedded)
}

// NewClient creates a new database client
func NewClient(compose *docker.Compose) *Client {
	mode := detectDatabaseMode()
	return &Client{
		mode:      mode,
		dbName:    getEnvOrDefault("DB_NAME", defaultDBName),
		dbUser:    getEnvOrDefault("DB_USER", defaultDBUser),
		compose:   compose,
		configErr: detectDatabaseConfigError(mode),
	}
}

// Mode returns the database mode (embedded or external)
func (c *Client) Mode() string {
	return c.mode
}

// IsEmbedded returns true if using embedded database mode
func (c *Client) IsEmbedded() bool {
	return c.mode == "embedded"
}

// IsExternal returns true if using external database mode
func (c *Client) IsExternal() bool {
	return c.mode == "external"
}

// ConfigError returns the configuration error detected during client creation, if any.
func (c *Client) ConfigError() error {
	return c.configErr
}

// GetContainerID returns the postgres container ID (embedded mode only)
func (c *Client) GetContainerID(ctx context.Context) (string, error) {
	if c.configErr != nil {
		return "", c.configErr
	}
	if c.containerID != "" {
		return c.containerID, nil
	}
	if c.compose == nil {
		return "", fmt.Errorf("docker compose not available")
	}
	id, err := c.compose.ContainerID(ctx, "postgres")
	if err != nil {
		return "", err
	}
	c.containerID = id
	return id, nil
}

// IsAccessible checks if the database is accepting connections
func (c *Client) IsAccessible(ctx context.Context) bool {
	if c.configErr != nil {
		return false
	}
	if c.IsExternal() {
		// For external, check via DATABASE_URL connectivity
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return false
		}
		// Try a simple query
		_, err := c.Query(ctx, "SELECT 1")
		return err == nil
	}

	// Embedded mode: use pg_isready via docker exec
	containerID, err := c.GetContainerID(ctx)
	if err != nil {
		return false
	}

	cmd := []string{"pg_isready", "-U", c.dbUser, "-d", c.dbName}
	_, err = docker.ExecInContainer(ctx, containerID, cmd...)
	return err == nil
}

// Query executes a SQL query and returns the result
func (c *Client) Query(ctx context.Context, query string) (string, error) {
	if c.configErr != nil {
		return "", c.configErr
	}
	if c.IsExternal() {
		return c.queryExternal(ctx, query)
	}
	return c.queryEmbedded(ctx, query)
}

// queryEmbedded executes a query in embedded mode via docker exec
func (c *Client) queryEmbedded(ctx context.Context, query string) (string, error) {
	containerID, err := c.GetContainerID(ctx)
	if err != nil {
		return "", err
	}

	cmd := []string{
		"psql",
		"-U", c.dbUser,
		"-d", c.dbName,
		"-t", // Tuple only, no headers
		"-c", query,
	}

	return docker.ExecInContainer(ctx, containerID, cmd...)
}

// queryExternal executes a query against an external database
func (c *Client) queryExternal(ctx context.Context, query string) (string, error) {
	if c.db == nil {
		// Lazy initialize connection
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			return "", fmt.Errorf("DATABASE_URL not set for external database mode")
		}
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			return "", fmt.Errorf("failed to open database connection: %w", err)
		}
		// Verify connection
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return "", fmt.Errorf("failed to ping database: %w", err)
		}
		c.db = db
	}

	// Execute query
	var result string
	err := c.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("query failed: %w", err)
	}
	return result, nil
}

// Close closes the database connection (for external mode)
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// TablesExist checks if the expected LiteLLM tables exist
func (c *Client) TablesExist(ctx context.Context) bool {
	query := `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('LiteLLM_VerificationToken', 'LiteLLM_UserTable', 'LiteLLM_BudgetTable', 'LiteLLM_SpendLogs');
	`
	result, err := c.Query(ctx, query)
	if err != nil {
		return false
	}

	// Parse result - should be at least 4 if all tables exist
	result = strings.TrimSpace(result)
	return result == "4" || result == "4\n"
}

// Version returns the PostgreSQL version
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.Query(ctx, "SELECT version();")
}

// Backup creates a database backup
func (c *Client) Backup(ctx context.Context) (string, error) {
	if c.configErr != nil {
		return "", c.configErr
	}
	if c.IsExternal() {
		return "", fmt.Errorf("backup not supported for external database mode")
	}

	containerID, err := c.GetContainerID(ctx)
	if err != nil {
		return "", err
	}

	cmd := []string{
		"pg_dump",
		"-U", c.dbUser,
		"-d", c.dbName,
		"-c", // Include SQL commands to create the database
		"-C", // Create database before restoring
		"-E", "UTF8",
		"--no-owner",
		"--no-acl",
	}

	return docker.ExecInContainer(ctx, containerID, cmd...)
}

// Restore restores a database from SQL
func (c *Client) Restore(ctx context.Context, sql string) error {
	if c.configErr != nil {
		return c.configErr
	}
	if c.IsExternal() {
		return fmt.Errorf("restore not supported for external database mode")
	}

	containerID, err := c.GetContainerID(ctx)
	if err != nil {
		return err
	}

	// Use psql to restore
	cmd := []string{
		"psql",
		"-U", c.dbUser,
		"-d", c.dbName,
	}

	// We need to pipe the SQL to psql
	// For now, this is a simplified version
	_, err = docker.ExecInContainer(ctx, containerID, append(cmd, "-c", sql)...)
	return err
}

// detectDatabaseMode detects the database mode from explicit configuration.
// Priority: 1) ACP_DATABASE_MODE env var, 2) ACP_DATABASE_MODE in demo/.env, 3) default embedded.
//
// DATABASE_URL alone is not sufficient to infer external mode because the
// default embedded demo stack also defines DATABASE_URL for LiteLLM.
func detectDatabaseMode() string {
	if mode, ok := explicitDatabaseMode(); ok {
		return mode
	}
	return "embedded"
}

func detectDatabaseConfigError(mode string) error {
	if mode != "embedded" {
		return nil
	}
	if _, ok := explicitDatabaseMode(); ok {
		return nil
	}
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		if repoValue, ok := repoEnvValue("DATABASE_URL"); ok {
			dbURL = strings.TrimSpace(repoValue)
		}
	}
	if dbURL == "" || dbURL == defaultEmbeddedDatabaseURL {
		return nil
	}
	return fmt.Errorf("ambiguous database configuration: DATABASE_URL is set but ACP_DATABASE_MODE is not; set ACP_DATABASE_MODE=external for external PostgreSQL or ACP_DATABASE_MODE=embedded for the local demo stack")
}

func explicitDatabaseMode() (string, bool) {
	if mode, ok := normalizeDatabaseMode(os.Getenv("ACP_DATABASE_MODE")); ok {
		return mode, true
	}
	if repoValue, ok := repoEnvValue("ACP_DATABASE_MODE"); ok {
		return normalizeDatabaseMode(repoValue)
	}
	return "", false
}

func repoEnvValue(key string) (string, bool) {
	repoRoot := resolveRepoRoot()
	if repoRoot == "" {
		return "", false
	}
	return envFileValue(repoRoot+"/demo/.env", key)
}

func envFileValue(path, key string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lineKey, value, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(lineKey), key) {
			continue
		}
		return strings.TrimSpace(value), true
	}
	return "", false
}

func normalizeDatabaseMode(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "embedded":
		return "embedded", true
	case "external":
		return "external", true
	default:
		return "", false
	}
}

func resolveRepoRoot() string {
	repoRoot := os.Getenv("ACP_REPO_ROOT")
	if repoRoot != "" {
		return repoRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// getEnvOrDefault returns the value of an environment variable or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
