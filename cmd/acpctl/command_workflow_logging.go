// command_workflow_logging.go - Shared structured workflow logging helpers.
//
// Purpose:
//   - Keep native command handlers aligned on a single structured workflow
//     logging shape without duplicating logger setup logic.
//
// Responsibilities:
//   - Scope command loggers to workflow names.
//   - Emit stable start/complete/failure/warn workflow events.
//
// Scope:
//   - Command-layer workflow event logging only.
//
// Usage:
//   - Used by native command handlers that execute long-running or multi-step
//     workflows while leaving final report rendering on stdout/stderr.
//
// Invariants/Assumptions:
//   - Missing command loggers degrade to a no-op logger.
//   - Workflow events remain stderr-oriented through the seeded command logger.
package main

import (
	"log/slog"

	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

func workflowLogger(runCtx commandRunContext, workflow string, attrs ...any) *slog.Logger {
	base := ensureWorkflowLogger(runCtx)
	args := make([]any, 0, len(attrs)+1)
	args = append(args, slog.String("workflow", workflow))
	args = append(args, attrs...)
	return base.With(args...)
}

func workflowStart(logger *slog.Logger, attrs ...any) {
	logging.WorkflowStart(logger, attrs...)
}

func workflowWarn(logger *slog.Logger, attrs ...any) {
	logging.WorkflowWarn(logger, attrs...)
}

func workflowFailure(logger *slog.Logger, err error, attrs ...any) {
	logging.WorkflowFailure(logger, err, attrs...)
}

func workflowComplete(logger *slog.Logger, attrs ...any) {
	logging.WorkflowComplete(logger, attrs...)
}
