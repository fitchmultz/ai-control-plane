// cmd_db_status.go - Database status command implementation.
//
// Purpose:
//   - Own the typed database status workflow and preserve the operator-facing
//     report contract used by docs and runbooks.
//
// Responsibilities:
//   - Read typed runtime and readonly database summaries.
//   - Render stable human-readable status sections for runtime, schema, keys,
//     budgets, and detections.
//   - Fail clearly when the database is unreachable or prerequisites are
//     missing.
//
// Scope:
//   - `acpctl db status` only.
//
// Usage:
//   - Invoked through `acpctl db status` or `make db-status`.
//
// Invariants/Assumptions:
//   - Status remains read-only.
//   - Section names stay aligned with operator documentation.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

const expectedCoreTables = 4

type dbStatusReaders struct {
	Mode     string
	Runtime  db.RuntimeServiceReader
	Readonly db.ReadonlyServiceReader
	Close    func()
}

var openDBStatusReaders = func(repoRoot string) (dbStatusReaders, error) {
	services, err := openDBServices(repoRoot)
	if err != nil {
		return dbStatusReaders{}, err
	}
	return dbStatusReaders{
		Mode:     services.Mode,
		Runtime:  services.Runtime,
		Readonly: services.Readonly,
		Close:    services.Close,
	}, nil
}

func runDBStatus(ctx context.Context, runCtx commandRunContext, _ any) int {
	out := output.New()
	logger := workflowLogger(runCtx, "db_status")
	workflowStart(logger)

	readers, err := openDBStatusReaders(runCtx.RepoRoot)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintln(runCtx.Stderr, out.Fail(err.Error()))
		return exitcodes.ACPExitPrereq
	}
	if readers.Close != nil {
		defer readers.Close()
	}

	summary, err := readers.Runtime.Summary(ctx)
	if err != nil {
		workflowFailure(logger, err)
		message := strings.TrimSpace(err.Error())
		if strings.TrimSpace(summary.Ping.Error) != "" {
			message = strings.TrimSpace(summary.Ping.Error)
		}
		if message == "" {
			message = "database status unavailable"
		}
		fmt.Fprintln(runCtx.Stderr, out.Fail(message))
		return exitcodes.ACPExitPrereq
	}

	keySummary, keyErr := readers.Readonly.KeySummary(ctx)
	budgetSummary, budgetErr := readers.Readonly.BudgetSummary(ctx)
	detectionSummary, detectionErr := readers.Readonly.DetectionSummary(ctx)

	printCommandSection(runCtx.Stdout, out, "1. Runtime Summary")
	printCommandDetail(runCtx.Stdout, "Mode", summary.Mode)
	printCommandDetail(runCtx.Stdout, "Database", summary.DatabaseName)
	printCommandDetail(runCtx.Stdout, "User", summary.DatabaseUser)
	if summary.ContainerID != "" {
		printCommandDetail(runCtx.Stdout, "Container", summary.ContainerID)
	}
	printCommandDetail(runCtx.Stdout, "Connectivity", "reachable")
	printCommandDetail(runCtx.Stdout, "Size", summary.Size)
	printCommandDetail(runCtx.Stdout, "Connections", summary.Connections)

	fmt.Fprintln(runCtx.Stdout)
	printCommandSection(runCtx.Stdout, out, "2. Schema Verification")
	printCommandDetail(runCtx.Stdout, "PostgreSQL", summary.Version)
	printCommandDetail(runCtx.Stdout, "Core tables", fmt.Sprintf("%d/%d", summary.ExpectedTables, expectedCoreTables))
	if summary.ExpectedTables < expectedCoreTables {
		printCommandDetail(runCtx.Stdout, "Status", "schema incomplete; LiteLLM initialization may still be in progress")
	} else {
		printCommandDetail(runCtx.Stdout, "Status", "expected core tables detected")
	}

	fmt.Fprintln(runCtx.Stdout)
	printCommandSection(runCtx.Stdout, out, "3. Virtual Keys")
	if keyErr != nil {
		printCommandDetail(runCtx.Stdout, "Status", fmt.Sprintf("unavailable (%v)", keyErr))
	} else {
		printCommandDetail(runCtx.Stdout, "Total", keySummary.Total)
		printCommandDetail(runCtx.Stdout, "Active", keySummary.Active)
		printCommandDetail(runCtx.Stdout, "Expired", keySummary.Expired)
	}

	fmt.Fprintln(runCtx.Stdout)
	printCommandSection(runCtx.Stdout, out, "4. Budget Usage")
	if budgetErr != nil {
		printCommandDetail(runCtx.Stdout, "Status", fmt.Sprintf("unavailable (%v)", budgetErr))
	} else {
		printCommandDetail(runCtx.Stdout, "Total budgets", budgetSummary.Total)
		printCommandDetail(runCtx.Stdout, "High utilization", budgetSummary.HighUtilization)
		printCommandDetail(runCtx.Stdout, "Exhausted", budgetSummary.Exhausted)
	}

	fmt.Fprintln(runCtx.Stdout)
	printCommandSection(runCtx.Stdout, out, "5. Detection Summary")
	switch {
	case detectionErr != nil:
		printCommandDetail(runCtx.Stdout, "Status", fmt.Sprintf("unavailable (%v)", detectionErr))
	case !detectionSummary.SpendLogsTableExists:
		printCommandDetail(runCtx.Stdout, "Status", "LiteLLM_SpendLogs not present; detection metrics are not available yet")
	default:
		printCommandDetail(runCtx.Stdout, "High severity (24h)", detectionSummary.HighSeverity)
		printCommandDetail(runCtx.Stdout, "Medium severity (24h)", detectionSummary.MediumSeverity)
		printCommandDetail(runCtx.Stdout, "Unique models (24h)", detectionSummary.UniqueModels24h)
		printCommandDetail(runCtx.Stdout, "Entries (24h)", detectionSummary.TotalEntries24h)
	}

	workflowComplete(logger,
		"mode", readers.Mode,
		"core_tables", summary.ExpectedTables,
		"keys_error", keyErr != nil,
		"budget_error", budgetErr != nil,
		"detection_error", detectionErr != nil,
	)
	return exitcodes.ACPExitSuccess
}

func runDBStatusCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"db", "status"}, args, stdout, stderr)
}
