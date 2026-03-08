// Package db provides PostgreSQL database operations.
//
// Purpose:
//
//	Provide one typed ACP database service that works consistently across
//	embedded Docker and external PostgreSQL modes.
//
// Responsibilities:
//   - Resolve the effective database runtime configuration.
//   - Execute bounded database probes and typed summary queries.
//   - Hide mode-specific embedded/external execution details from callers.
//   - Provide backup and restore support for embedded mode.
//
// Non-scope:
//   - Does not manage database migrations.
//   - Does not implement long-lived application-side query workloads.
//
// Invariants/Assumptions:
//   - Embedded mode uses the repo-local Compose postgres service.
//   - External mode uses DATABASE_URL and validates connectivity explicitly.
//
// Scope:
//   - File-local service implementation and typed summaries only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/proc"

	_ "github.com/lib/pq"
)

const (
	defaultDBName          = "litellm"
	defaultDBUser          = "litellm"
	containerLookupTimeout = 5 * time.Second
	queryTimeout           = 10 * time.Second
)

// Probe captures a typed outcome for a single database operation.
type Probe struct {
	Operation string        `json:"operation"`
	Healthy   bool          `json:"healthy"`
	Latency   time.Duration `json:"latency,omitempty"`
	Error     string        `json:"error,omitempty"`
}

// Summary captures typed database runtime health.
type Summary struct {
	Mode           config.DatabaseMode `json:"mode"`
	DatabaseName   string              `json:"database_name"`
	DatabaseUser   string              `json:"database_user"`
	ContainerID    string              `json:"container_id,omitempty"`
	Ping           Probe               `json:"ping"`
	ExpectedTables int                 `json:"expected_tables,omitempty"`
	Version        string              `json:"version,omitempty"`
	Size           string              `json:"size,omitempty"`
	Connections    int                 `json:"connections,omitempty"`
}

// KeySummary captures typed virtual key counts.
type KeySummary struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Expired int `json:"expired"`
}

// BudgetSummary captures typed budget utilization counts.
type BudgetSummary struct {
	Total           int `json:"total"`
	HighUtilization int `json:"high_utilization"`
	Exhausted       int `json:"exhausted"`
}

// DetectionSummary captures typed spend-log finding counts.
type DetectionSummary struct {
	SpendLogsTableExists bool `json:"spend_logs_table_exists"`
	HighSeverity         int  `json:"high_severity"`
	MediumSeverity       int  `json:"medium_severity"`
	UniqueModels24h      int  `json:"unique_models_24h"`
	TotalEntries24h      int  `json:"total_entries_24h"`
}

// Client provides database operations.
type Client struct {
	settings  config.DatabaseSettings
	compose   *docker.Compose
	container string
	sqlDB     *sql.DB
}

// NewClient creates a new database client.
func NewClient(repoRoot string) *Client {
	settings := config.NewLoader().Database(context.Background())
	client := &Client{settings: settings}
	if settings.Mode.IsEmbedded() && repoRoot != "" {
		compose, err := docker.NewCompose(docker.DefaultProjectDir(repoRoot))
		if err == nil {
			client.compose = compose
		}
	}
	return client
}

// Mode returns the effective database mode.
func (c *Client) Mode() config.DatabaseMode {
	return c.settings.Mode
}

// IsEmbedded returns true if using embedded database mode.
func (c *Client) IsEmbedded() bool {
	return c.settings.Mode.IsEmbedded()
}

// IsExternal returns true if using external database mode.
func (c *Client) IsExternal() bool {
	return c.settings.Mode.IsExternal()
}

// ConfigError returns the configuration error detected during client creation, if any.
func (c *Client) ConfigError() error {
	return c.settings.AmbiguousErr
}

// Close closes the external database connection if one was opened.
func (c *Client) Close() error {
	if c.sqlDB != nil {
		return c.sqlDB.Close()
	}
	return nil
}

// Ping checks if the configured database accepts connections.
func (c *Client) Ping(ctx context.Context) Probe {
	start := time.Now()
	if c.settings.AmbiguousErr != nil {
		return Probe{
			Operation: "ping",
			Error:     c.settings.AmbiguousErr.Error(),
			Latency:   time.Since(start).Round(time.Millisecond),
		}
	}

	if c.IsExternal() {
		dbConn, err := c.ensureSQLDB(ctx)
		if err != nil {
			return Probe{Operation: "ping", Error: err.Error(), Latency: time.Since(start).Round(time.Millisecond)}
		}
		err = dbConn.PingContext(withTimeoutContext(ctx, queryTimeout))
		if err != nil {
			return Probe{Operation: "ping", Error: fmt.Sprintf("database ping failed: %v", err), Latency: time.Since(start).Round(time.Millisecond)}
		}
		return Probe{Operation: "ping", Healthy: true, Latency: time.Since(start).Round(time.Millisecond)}
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return Probe{Operation: "ping", Error: err.Error(), Latency: time.Since(start).Round(time.Millisecond)}
	}

	res := proc.Run(withTimeoutContext(ctx, queryTimeout), proc.Request{
		Name:    "docker",
		Args:    []string{"exec", containerID, "pg_isready", "-U", c.user(), "-d", c.name()},
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

// IsAccessible reports whether the database accepts connections.
func (c *Client) IsAccessible(ctx context.Context) bool {
	return c.Ping(ctx).Healthy
}

// Summary returns typed runtime health details for the database.
func (c *Client) Summary(ctx context.Context) (Summary, error) {
	summary := Summary{
		Mode:         c.Mode(),
		DatabaseName: c.name(),
		DatabaseUser: c.user(),
	}
	summary.Ping = c.Ping(ctx)
	if !summary.Ping.Healthy {
		return summary, fmt.Errorf("%s", summary.Ping.Error)
	}

	if c.IsEmbedded() {
		containerID, err := c.containerID(ctx)
		if err != nil {
			return summary, err
		}
		summary.ContainerID = containerID
	}

	tableCount, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('LiteLLM_VerificationToken', 'LiteLLM_UserTable', 'LiteLLM_BudgetTable', 'LiteLLM_SpendLogs');
	`)
	if err != nil {
		return summary, err
	}
	summary.ExpectedTables = tableCount

	version, err := c.queryString(ctx, "SELECT version();")
	if err != nil {
		return summary, err
	}
	summary.Version = strings.TrimSpace(version)

	size, err := c.queryString(ctx, fmt.Sprintf("SELECT pg_size_pretty(pg_database_size('%s'));", c.name()))
	if err != nil {
		return summary, err
	}
	summary.Size = strings.TrimSpace(size)

	connections, err := c.queryInt(ctx, fmt.Sprintf("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = '%s';", c.name()))
	if err != nil {
		return summary, err
	}
	summary.Connections = connections

	return summary, nil
}

// TablesExist checks if the expected LiteLLM tables exist.
func (c *Client) TablesExist(ctx context.Context) bool {
	summary, err := c.Summary(ctx)
	return err == nil && summary.ExpectedTables == 4
}

// Version returns the PostgreSQL version.
func (c *Client) Version(ctx context.Context) (string, error) {
	return c.queryString(ctx, "SELECT version();")
}

// KeySummary returns typed virtual key counts.
func (c *Client) KeySummary(ctx context.Context) (KeySummary, error) {
	total, err := c.queryInt(ctx, `SELECT COUNT(*) FROM "LiteLLM_VerificationToken";`)
	if err != nil {
		return KeySummary{}, err
	}
	active, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_VerificationToken"
		WHERE expires IS NULL OR expires > NOW();
	`)
	if err != nil {
		return KeySummary{}, err
	}
	return KeySummary{
		Total:   total,
		Active:  active,
		Expired: total - active,
	}, nil
}

// BudgetSummary returns typed budget utilization counts.
func (c *Client) BudgetSummary(ctx context.Context) (BudgetSummary, error) {
	total, err := c.queryInt(ctx, `SELECT COUNT(*) FROM "LiteLLM_BudgetTable";`)
	if err != nil {
		return BudgetSummary{}, err
	}
	high, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE max_budget > 0 AND (budget::float / max_budget::float * 100) <= 20;
	`)
	if err != nil {
		return BudgetSummary{}, err
	}
	exhausted, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_BudgetTable"
		WHERE budget <= 0;
	`)
	if err != nil {
		return BudgetSummary{}, err
	}
	return BudgetSummary{
		Total:           total,
		HighUtilization: high,
		Exhausted:       exhausted,
	}, nil
}

// DetectionSummary returns typed recent detection findings.
func (c *Client) DetectionSummary(ctx context.Context) (DetectionSummary, error) {
	tableCount, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'LiteLLM_SpendLogs';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	if tableCount == 0 {
		return DetectionSummary{}, nil
	}

	high, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	medium, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE spend > 5.0 AND spend <= 10.0
		AND "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	uniqueModels, err := c.queryInt(ctx, `
		SELECT COUNT(DISTINCT model) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	totalEntries, err := c.queryInt(ctx, `
		SELECT COUNT(*) FROM "LiteLLM_SpendLogs"
		WHERE "startTime" > NOW() - INTERVAL '24 hours';
	`)
	if err != nil {
		return DetectionSummary{}, err
	}
	return DetectionSummary{
		SpendLogsTableExists: true,
		HighSeverity:         high,
		MediumSeverity:       medium,
		UniqueModels24h:      uniqueModels,
		TotalEntries24h:      totalEntries,
	}, nil
}

// Query executes a SQL query and returns a trimmed scalar result.
func (c *Client) Query(ctx context.Context, query string) (string, error) {
	return c.queryString(ctx, query)
}

// Backup creates a database backup.
func (c *Client) Backup(ctx context.Context) (string, error) {
	if c.settings.AmbiguousErr != nil {
		return "", c.settings.AmbiguousErr
	}
	if c.IsExternal() {
		return "", fmt.Errorf("backup not supported for external database mode")
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return "", err
	}

	res := proc.Run(withTimeoutContext(ctx, 30*time.Second), proc.Request{
		Name: "docker",
		Args: []string{
			"exec", containerID,
			"pg_dump",
			"-U", c.user(),
			"-d", c.name(),
			"-c",
			"-C",
			"-E", "UTF8",
			"--no-owner",
			"--no-acl",
		},
		Timeout: 30 * time.Second,
	})
	if res.Err != nil {
		return "", fmt.Errorf("database backup failed: %s", c.execError("database backup", res))
	}
	return res.Stdout, nil
}

// Restore restores a database from a streamed SQL reader.
func (c *Client) Restore(ctx context.Context, sqlReader io.Reader) error {
	if c.settings.AmbiguousErr != nil {
		return c.settings.AmbiguousErr
	}
	if c.IsExternal() {
		return fmt.Errorf("restore not supported for external database mode")
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return err
	}

	res := proc.Run(withTimeoutContext(ctx, 30*time.Second), proc.Request{
		Name:    "docker",
		Args:    []string{"exec", "-i", containerID, "psql", "-X", "-v", "ON_ERROR_STOP=1", "-U", c.user(), "-d", "postgres"},
		Stdin:   sqlReader,
		Timeout: 30 * time.Second,
	})
	if res.Err != nil {
		return fmt.Errorf("database restore failed: %s", c.execError("database restore", res))
	}
	return nil
}

func (c *Client) name() string {
	if strings.TrimSpace(c.settings.Name) != "" {
		return c.settings.Name
	}
	return defaultDBName
}

func (c *Client) user() string {
	if strings.TrimSpace(c.settings.User) != "" {
		return c.settings.User
	}
	return defaultDBUser
}

func (c *Client) containerID(ctx context.Context) (string, error) {
	if c.container != "" {
		return c.container, nil
	}
	if c.compose == nil {
		return "", fmt.Errorf("database runtime requires docker compose for embedded mode")
	}
	lookupCtx := withTimeoutContext(ctx, containerLookupTimeout)
	containerID, err := c.compose.ContainerID(lookupCtx, "postgres")
	if err != nil {
		return "", fmt.Errorf("database container lookup failed: %w", err)
	}
	c.container = strings.TrimSpace(containerID)
	return c.container, nil
}

func (c *Client) ensureSQLDB(ctx context.Context) (*sql.DB, error) {
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
	if err := dbConn.PingContext(withTimeoutContext(ctx, queryTimeout)); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	c.sqlDB = dbConn
	return c.sqlDB, nil
}

func (c *Client) queryString(ctx context.Context, query string) (string, error) {
	if c.settings.AmbiguousErr != nil {
		return "", c.settings.AmbiguousErr
	}
	if c.IsExternal() {
		dbConn, err := c.ensureSQLDB(ctx)
		if err != nil {
			return "", err
		}
		var value sql.NullString
		err = dbConn.QueryRowContext(withTimeoutContext(ctx, queryTimeout), query).Scan(&value)
		if err != nil {
			return "", fmt.Errorf("database query failed: %w", err)
		}
		return strings.TrimSpace(value.String), nil
	}

	containerID, err := c.containerID(ctx)
	if err != nil {
		return "", err
	}
	res := proc.Run(withTimeoutContext(ctx, queryTimeout), proc.Request{
		Name:    "docker",
		Args:    []string{"exec", containerID, "psql", "-X", "-A", "-t", "-U", c.user(), "-d", c.name(), "-c", query},
		Timeout: queryTimeout,
	})
	if res.Err != nil {
		return "", fmt.Errorf("database query failed: %s", c.execError("database query", res))
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (c *Client) queryInt(ctx context.Context, query string) (int, error) {
	value, err := c.queryString(ctx, query)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("database query parse failed: %w", err)
	}
	return parsed, nil
}

func (c *Client) execError(prefix string, res proc.Result) string {
	if res.Stderr != "" {
		return fmt.Sprintf("%s: %s", prefix, strings.TrimSpace(res.Stderr))
	}
	if res.Err != nil {
		return fmt.Sprintf("%s: %v", prefix, res.Err)
	}
	return prefix
}

func withTimeoutContext(ctx context.Context, timeout time.Duration) context.Context {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx
	}
	child, _ := context.WithTimeout(ctx, timeout)
	return child
}

func detectDatabaseMode() string {
	return config.NewLoader().Database(context.Background()).Mode.String()
}

func getEnvOrDefault(key, defaultValue string) string {
	return config.NewLoader().StringDefault(key, defaultValue)
}
