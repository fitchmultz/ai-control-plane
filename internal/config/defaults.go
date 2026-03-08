// Package config provides default configuration values for the AI Control Plane.
//
// Purpose:
//
//	Centralize default ports, hosts, timeouts, and other configuration constants
//	to eliminate hardcoded values across the codebase.
//
// Responsibilities:
//   - Define default network ports (LiteLLM, PostgreSQL, etc.)
//   - Define default host addresses
//   - Define default timeout values for HTTP clients and health checks
//   - Define polling intervals and retry defaults
//
// Non-scope:
//   - Does not read from environment variables (use os.Getenv at call sites)
//   - Does not provide runtime configuration management
//
// Invariants:
//   - All constants are defined as const (not variables)
//   - Constants follow naming convention: Default* for user-overridable values
//   - Time-related constants use time.Duration type
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package config

import "time"

// Network defaults
const (
	// DefaultLiteLLMPort is the default LiteLLM gateway port
	DefaultLiteLLMPort = 4000
	// DefaultPostgresPort is the default PostgreSQL port
	DefaultPostgresPort = 5432
	// DefaultGatewayHost is the default gateway bind address
	DefaultGatewayHost = "127.0.0.1"
)

// HTTP client timeout defaults
const (
	// DefaultHTTPTimeout is the default HTTP client timeout
	DefaultHTTPTimeout = 5 * time.Second
	// DefaultHealthCheckTimeout is the default timeout for health check requests
	DefaultHealthCheckTimeout = 5 * time.Second
	// DefaultConnectTimeout is the default connection timeout for curl operations
	DefaultConnectTimeout = 2 * time.Second
	// DefaultMaxTime is the default maximum time for curl operations
	DefaultMaxTime = 3 * time.Second
)

// Polling defaults
const (
	// DefaultPollInterval is the default interval between poll attempts
	DefaultPollInterval = 1 * time.Second
	// DefaultPollTimeout is the default maximum time to wait for a condition
	DefaultPollTimeout = 30 * time.Second
	// DefaultBudgetPollTimeout is the default timeout for budget status polling
	DefaultBudgetPollTimeout = 10 * time.Second
	// DefaultAuditPollTimeout is the default timeout for audit log polling
	DefaultAuditPollTimeout = 10 * time.Second
)

// RequiredPorts returns the list of ports that must be available for the demo profile
func RequiredPorts() []int {
	return []int{DefaultLiteLLMPort, DefaultPostgresPort}
}
