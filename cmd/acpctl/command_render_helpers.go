// command_render_helpers.go - Shared command output rendering helpers.
//
// Purpose:
//   - Keep operator-facing native command output concise and consistent for
//     long-running artifact and verification workflows.
//
// Responsibilities:
//   - Render standardized section headers and success banners.
//   - Print aligned summary detail lines and short next-step guidance.
//
// Scope:
//   - Command-layer human output helpers only.
//
// Usage:
//   - Used by artifact-producing commands in cmd/acpctl.
//
// Invariants/Assumptions:
//   - Callers control the wording and order of detail lines.
//   - Helpers preserve the existing stdout/stderr split.
package main

import (
	"fmt"
	"io"

	"github.com/mitchfultz/ai-control-plane/internal/output"
)

func printCommandSection(out io.Writer, printer *output.Output, title string) {
	fmt.Fprintln(out, printer.Bold(title))
}

func printCommandSuccess(out io.Writer, printer *output.Output, title string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, printer.Green(printer.Bold(title)))
}

func printCommandDetail(out io.Writer, label string, value any) {
	fmt.Fprintf(out, "  %-16s %v\n", label+":", value)
}

func printCommandNextStep(out io.Writer, label string, command string) {
	fmt.Fprintf(out, "  %-16s %s\n", label+":", command)
}
