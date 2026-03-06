// Package terminal provides terminal color and styling utilities.
//
// Purpose:
//
//	Centralize terminal color definitions for consistent output formatting
//	across all Go commands in the AI Control Plane.
//
// Responsibilities:
//   - Define color/style constants (ANSI codes)
//   - Provide TTY detection for conditional color output
//   - Export common formatting helpers
//
// Non-scope:
//   - Does not format structured output (JSON/YAML)
//   - Does not handle complex layout
//
// Invariants:
//   - Colors are empty strings when output is not a terminal (or NO_COLOR is set)
//   - All constants are defined as string constants (not variables)
//   - Package has no side effects on import
package terminal

import (
	"os"
)

// ANSI color and style codes.
// These are raw escape sequences - use Colors() to get terminal-aware values.
const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiRed     = "\033[31m"
	ansiCyan    = "\033[36m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiClear   = "\033[H\033[2J"
)

// Colors holds terminal color/style values.
// When UseColor is false, all values are empty strings.
type Colors struct {
	UseColor bool
	Reset    string
	Bold     string
	Green    string
	Yellow   string
	Red      string
	Cyan     string
	Blue     string
	Magenta  string
	Clear    string // Clear screen sequence
}

// NewColors returns a Colors struct with values based on terminal detection.
// Respects NO_COLOR environment variable.
func NewColors() Colors {
	if os.Getenv("NO_COLOR") != "" || !IsTerminal(os.Stdout) {
		return Colors{UseColor: false}
	}
	return Colors{
		UseColor: true,
		Reset:    ansiReset,
		Bold:     ansiBold,
		Green:    ansiGreen,
		Yellow:   ansiYellow,
		Red:      ansiRed,
		Cyan:     ansiCyan,
		Blue:     ansiBlue,
		Magenta:  ansiMagenta,
		Clear:    ansiClear,
	}
}

// IsTerminal checks if a file is a terminal device.
func IsTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// StatusFormatter returns formatted status strings with appropriate colors.
type StatusFormatter struct {
	c Colors
}

// NewStatusFormatter creates a formatter using terminal-aware colors.
func NewStatusFormatter() StatusFormatter {
	return StatusFormatter{c: NewColors()}
}

// OK returns a green [OK] or [OK] string.
func (sf StatusFormatter) OK() string {
	if sf.c.UseColor {
		return sf.c.Green + "[OK]" + sf.c.Reset
	}
	return "[OK]"
}

// Warn returns a yellow [WARN] or [WARN] string.
func (sf StatusFormatter) Warn() string {
	if sf.c.UseColor {
		return sf.c.Yellow + "[WARN]" + sf.c.Reset
	}
	return "[WARN]"
}

// Fail returns a red [FAIL] or [FAIL] string.
func (sf StatusFormatter) Fail() string {
	if sf.c.UseColor {
		return sf.c.Red + "[FAIL]" + sf.c.Reset
	}
	return "[FAIL]"
}

// Healthy returns a green "HEALTHY" or "HEALTHY" string.
func (sf StatusFormatter) Healthy() string {
	if sf.c.UseColor {
		return sf.c.Green + "HEALTHY" + sf.c.Reset
	}
	return "HEALTHY"
}

// Warning returns a yellow "WARNING" or "WARNING" string.
func (sf StatusFormatter) Warning() string {
	if sf.c.UseColor {
		return sf.c.Yellow + "WARNING" + sf.c.Reset
	}
	return "WARNING"
}

// Unhealthy returns a red "UNHEALTHY" or "UNHEALTHY" string.
func (sf StatusFormatter) Unhealthy() string {
	if sf.c.UseColor {
		return sf.c.Red + "UNHEALTHY" + sf.c.Reset
	}
	return "UNHEALTHY"
}
