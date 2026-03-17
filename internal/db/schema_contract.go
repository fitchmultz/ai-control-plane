// schema_contract.go - Shared LiteLLM core schema verification contract.
//
// Purpose:
//   - Centralize the expected LiteLLM core-table contract used by runtime
//     inspection and restore-verification workflows.
//
// Responsibilities:
//   - Define the canonical set of required LiteLLM core tables.
//   - Provide shared helpers for counting and verifying that schema contract.
//
// Scope:
//   - Shared schema verification metadata only.
//
// Usage:
//   - Consumed by runtime inspection, status adaptation, and database DR drills.
//
// Invariants/Assumptions:
//   - The supported LiteLLM schema contract currently requires four core tables.
package db

import "strings"

var coreSchemaTables = []string{
	"LiteLLM_VerificationToken",
	"LiteLLM_UserTable",
	"LiteLLM_BudgetTable",
	"LiteLLM_SpendLogs",
}

// RestoreVerification captures typed scratch-restore verification results.
type RestoreVerification struct {
	DatabaseName   string `json:"database_name"`
	ExpectedTables int    `json:"expected_tables"`
	FoundTables    int    `json:"found_tables"`
	Version        string `json:"version,omitempty"`
}

// ExpectedCoreSchemaTableCount returns the canonical required core-table count.
func ExpectedCoreSchemaTableCount() int {
	return len(coreSchemaTables)
}

func coreSchemaTableCountQuery() string {
	quoted := make([]string, 0, len(coreSchemaTables))
	for _, table := range coreSchemaTables {
		quoted = append(quoted, quoteSQLStringLiteral(table))
	}
	return `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN (` + strings.Join(quoted, ", ") + `);
	`
}

func quoteSQLStringLiteral(value string) string {
	return `'` + strings.ReplaceAll(strings.TrimSpace(value), `'`, `''`) + `'`
}
