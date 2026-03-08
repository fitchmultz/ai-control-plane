// test_helpers_test.go - Shared helpers for database package unit tests.
//
// Purpose:
//   - Centralize repeatable sqlmock-backed connector setup for focused suites.
//
// Responsibilities:
//   - Build typed external connectors backed by deterministic sqlmock handles.
//   - Provide small helpers for exact SQL expectation registration.
//
// Scope:
//   - Test-only helpers for internal/db unit suites.
//
// Usage:
//   - Imported implicitly by other internal/db `_test.go` files.
//
// Invariants/Assumptions:
//   - Helpers avoid live Docker or PostgreSQL dependencies.
package db

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mitchfultz/ai-control-plane/internal/config"
)

func newExternalSQLMockConnector(t *testing.T) (*Connector, sqlmock.Sqlmock) {
	t.Helper()

	dbConn, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
	})
	t.Cleanup(func() {
		_ = dbConn.Close()
	})

	return &Connector{
		settings: config.DatabaseSettings{
			Mode: config.DatabaseModeExternal,
			URL:  "postgres://user:pass@localhost/db?sslmode=disable",
			Name: "litellm",
			User: "litellm",
		},
		sqlDB: dbConn,
	}, mock
}

func exactQueryRows(columns ...string) *sqlmock.Rows {
	return sqlmock.NewRows(columns)
}

func expectExactQuery(mock sqlmock.Sqlmock, query string, rows *sqlmock.Rows) *sqlmock.ExpectedQuery {
	return mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)
}

func connectorWithDB(settings config.DatabaseSettings, dbConn *sql.DB) *Connector {
	return &Connector{
		settings: settings,
		sqlDB:    dbConn,
	}
}
