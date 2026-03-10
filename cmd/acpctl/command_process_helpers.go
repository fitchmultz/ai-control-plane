// command_process_helpers.go - Shared subprocess validation and failure helpers.
//
// Purpose:
//   - Keep command-layer subprocess adapters aligned on one executable
//     validation path and one process-failure classification flow.
//
// Responsibilities:
//   - Classify canonical `internal/proc` failures into operator-facing messages.
//   - Preserve stable ACP exit-code behavior for delegated subprocess adapters.
//
// Scope:
//   - acpctl command-layer helpers for process-backed commands only.
//
// Usage:
//   - Used by bridge, delegated, and other command adapters that run external
//     tools through `internal/proc`.
//
// Invariants/Assumptions:
//   - Messages remain caller-defined so each command keeps its operator wording.
//   - Exit-code overrides are only applied for explicit classified failures.
package main

import (
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

type procFailureMessages struct {
	NotFound         string
	Timeout          string
	Canceled         string
	StartFormat      string
	Exit             string
	ExitCodeOverride int
	Fallback         string
}

func classifyProcFailure(err error, messages procFailureMessages) (string, int) {
	code := proc.ACPExitCode(err)

	switch {
	case proc.IsNotFound(err) && messages.NotFound != "":
		return messages.NotFound, code
	case proc.IsTimeout(err) && messages.Timeout != "":
		return messages.Timeout, code
	case proc.IsCanceled(err) && messages.Canceled != "":
		return messages.Canceled, code
	case proc.IsStart(err) && messages.StartFormat != "":
		return fmt.Sprintf(messages.StartFormat, err), code
	case proc.IsExit(err) && messages.Exit != "":
		if messages.ExitCodeOverride != 0 {
			code = messages.ExitCodeOverride
		}
		return messages.Exit, code
	case messages.Fallback != "":
		return messages.Fallback, code
	default:
		return err.Error(), code
	}
}
