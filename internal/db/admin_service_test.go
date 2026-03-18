// admin_service_test.go - Tests for admin restore-verification helpers.
//
// Purpose:
//   - Verify scratch-database rewrite logic for DR drills.
//
// Responsibilities:
//   - Ensure only database control lines are rewritten.
//   - Ensure missing control statements fail fast.
//
// Scope:
//   - Admin helper tests only.
//
// Usage:
//   - Run with `go test ./internal/db`.
//
// Invariants/Assumptions:
//   - Rewrites must not touch regular schema/data statements.
package db

import (
	"strings"
	"testing"
)

func TestRewriteBackupDatabaseName(t *testing.T) {
	t.Parallel()

	sql := strings.Join([]string{
		`DROP DATABASE litellm;`,
		`CREATE DATABASE litellm WITH TEMPLATE = template0 ENCODING = 'UTF8';`,
		`ALTER DATABASE litellm OWNER TO litellm;`,
		`\connect litellm`,
		`CREATE TABLE public.example (id integer);`,
		`COPY public.example (id) FROM stdin;`,
		`1`,
		`\.`,
		``,
	}, "\n")

	rewritten, err := rewriteBackupDatabaseName(sql, "litellm", "acp_dr_drill_20260316_010203")
	if err != nil {
		t.Fatalf("rewriteBackupDatabaseName() error = %v", err)
	}

	for _, want := range []string{
		`DROP DATABASE IF EXISTS acp_dr_drill_20260316_010203;`,
		`CREATE DATABASE acp_dr_drill_20260316_010203 WITH TEMPLATE = template0 ENCODING = 'UTF8';`,
		`ALTER DATABASE acp_dr_drill_20260316_010203 OWNER TO litellm;`,
		`\connect acp_dr_drill_20260316_010203`,
		`CREATE TABLE public.example (id integer);`,
		`COPY public.example (id) FROM stdin;`,
	} {
		if !strings.Contains(rewritten, want) {
			t.Fatalf("rewritten SQL missing %q\n%s", want, rewritten)
		}
	}
}

func TestRewriteBackupDatabaseNameRequiresControlStatements(t *testing.T) {
	t.Parallel()

	_, err := rewriteBackupDatabaseName(`CREATE TABLE public.example (id integer);`, "litellm", "scratch")
	if err == nil {
		t.Fatal("expected missing control statement error, got nil")
	}
}
