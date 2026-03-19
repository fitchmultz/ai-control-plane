// workflow.go - Canonical structured workflow logging helpers.
//
// Purpose:
//   - Provide one shared structured logging contract for workflow-oriented Go
//     packages and command handlers.
//
// Responsibilities:
//   - Build workflow-scoped loggers from context.
//   - Emit stable workflow start, warn, failure, and completion events.
//   - Keep error-field formatting consistent across packages.
//
// Scope:
//   - Shared workflow event helpers only.
//
// Usage:
//   - Used by command-layer and internal workflow packages that need stable
//     structured workflow events.
//
// Invariants/Assumptions:
//   - Missing context loggers degrade to a no-op logger.
//   - Workflow events use the fixed event names workflow.start,
//     workflow.warn, workflow.failed, and workflow.complete.
package logging

import (
	"context"
	"log/slog"
)

// WorkflowLogger scopes the context logger to workflow-specific attributes.
func WorkflowLogger(ctx context.Context, attrs ...any) *slog.Logger {
	return FromContext(ctx).With(attrs...)
}

// WorkflowStart emits the canonical workflow start event.
func WorkflowStart(logger *slog.Logger, attrs ...any) {
	logger.Info("workflow.start", attrs...)
}

// WorkflowWarn emits the canonical workflow warning event.
func WorkflowWarn(logger *slog.Logger, attrs ...any) {
	logger.Warn("workflow.warn", attrs...)
}

// WorkflowFailure emits the canonical workflow failure event.
func WorkflowFailure(logger *slog.Logger, err error, attrs ...any) {
	args := make([]any, 0, len(attrs)+1)
	args = append(args, Err(err))
	args = append(args, attrs...)
	logger.Error("workflow.failed", args...)
}

// WorkflowComplete emits the canonical workflow completion event.
func WorkflowComplete(logger *slog.Logger, attrs ...any) {
	logger.Info("workflow.complete", attrs...)
}
