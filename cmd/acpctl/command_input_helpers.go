// command_input_helpers.go - Shared command input-loading helpers.
//
// Purpose:
//   - Centralize repeated file/stdin JSON payload loading for typed workflows.
//
// Responsibilities:
//   - Read command input from a repository-relative file or stdin.
//   - Enforce consistent usage and runtime error handling semantics.
//
// Scope:
//   - Shared command-layer input helpers only.
//
// Usage:
//   - Used by typed artifact/evaluation commands such as evidence ingest and
//     policy eval.
//
// Invariants/Assumptions:
//   - Commands either receive `--file` or piped stdin content.
//   - Empty stdin is treated as a usage error.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func loadJSONPayload(inputPath string) ([]byte, string, int, error) {
	if strings.TrimSpace(inputPath) != "" {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("read input file: %w", err)
		}
		return data, inputPath, exitcodes.ACPExitSuccess, nil
	}
	info, err := os.Stdin.Stat()
	if err != nil {
		return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("inspect stdin: %w", err)
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return nil, "", exitcodes.ACPExitUsage, fmt.Errorf("provide --file or pipe JSON payload on stdin")
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, "", exitcodes.ACPExitRuntime, fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, "", exitcodes.ACPExitUsage, fmt.Errorf("stdin was empty")
	}
	return data, "stdin", exitcodes.ACPExitSuccess, nil
}
