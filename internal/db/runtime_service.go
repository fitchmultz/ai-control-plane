// Package db provides typed PostgreSQL runtime, readonly, and admin services.
//
// Purpose:
//   - Expose database runtime health and connectivity inspection.
//
// Responsibilities:
//   - Report database reachability, schema readiness, and capacity metadata.
//   - Keep runtime inspection separate from readonly inventory queries.
//
// Scope:
//   - Runtime inspection only.
//
// Usage:
//   - Construct via `NewRuntimeService(connector)` for status/doctor/health flows.
//
// Invariants/Assumptions:
//   - Summary queries remain read-only and bounded.
package db

import (
	"context"
	"fmt"
	"strings"
)

// RuntimeService provides typed runtime health inspection.
type RuntimeService struct {
	connector *Connector
}

// NewRuntimeService creates a typed database runtime service.
func NewRuntimeService(connector *Connector) *RuntimeService {
	return &RuntimeService{connector: connector}
}

// ConfigError returns the connector configuration error, if any.
func (s *RuntimeService) ConfigError() error {
	if s == nil || s.connector == nil {
		return fmt.Errorf("database runtime service requires a connector")
	}
	return s.connector.ConfigError()
}

// IsAccessible reports whether the configured database accepts connections.
func (s *RuntimeService) IsAccessible(ctx context.Context) bool {
	return s.connector.ping(ctx).Healthy
}

// Summary returns typed runtime health details for the database.
func (s *RuntimeService) Summary(ctx context.Context) (Summary, error) {
	if s == nil || s.connector == nil {
		return Summary{}, fmt.Errorf("database runtime service requires a connector")
	}

	summary := Summary{
		Mode:         s.connector.Mode(),
		DatabaseName: s.connector.databaseName(),
		DatabaseUser: s.connector.databaseUser(),
	}
	summary.Ping = s.connector.ping(ctx)
	if !summary.Ping.Healthy {
		return summary, fmt.Errorf("%s", summary.Ping.Error)
	}

	if s.connector.IsEmbedded() {
		containerID, err := s.connector.containerID(ctx)
		if err != nil {
			return summary, err
		}
		summary.ContainerID = containerID
	}

	tableCount, err := s.connector.scalarInt(ctx, coreSchemaTableCountQuery())
	if err != nil {
		return summary, err
	}
	summary.ExpectedTables = tableCount

	version, err := s.connector.scalarString(ctx, "SELECT version();")
	if err != nil {
		return summary, err
	}
	summary.Version = strings.TrimSpace(version)

	size, err := s.connector.scalarString(ctx, fmt.Sprintf("SELECT pg_size_pretty(pg_database_size('%s'));", s.connector.databaseName()))
	if err != nil {
		return summary, err
	}
	summary.Size = strings.TrimSpace(size)

	connections, err := s.connector.scalarInt(ctx, fmt.Sprintf("SELECT COUNT(*) FROM pg_stat_activity WHERE datname = '%s';", s.connector.databaseName()))
	if err != nil {
		return summary, err
	}
	summary.Connections = connections

	return summary, nil
}
