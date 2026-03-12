// Package output provides terminal output utilities including colors and symbols.
//
// Purpose:
//
//	Provide consistent, terminal-aware colored output and symbols for all
//	CLI commands and scripts.
//
// Responsibilities:
//   - Detect terminal capability for color output
//   - Define color constants and symbol constants
//   - Provide formatted output helpers
//
// Non-scope:
//   - Does not handle logging (see log package)
//   - Does not handle complex UI (see TUI packages)
//
// Invariants/Assumptions:
//   - Colors are disabled when stdout is not a terminal (unless forced)
//   - Symbols are consistent across all CLI output
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
package output

import (
	"fmt"
	"os"
)

// Color codes
const (
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorBold    = "\033[1m"
	colorReset   = "\033[0m"
)

// Symbols
const (
	symbolPass = "✓"
	symbolFail = "✗"
	symbolWarn = "⚠"
	symbolInfo = "ℹ"
)

// Output handles terminal-aware output
type Output struct {
	useColor bool
}

// New creates a new Output instance with auto-detected terminal capability
func New() *Output {
	return &Output{
		useColor: isTerminal(),
	}
}

// NewForceColor creates a new Output instance with forced color output
func NewForceColor() *Output {
	return &Output{useColor: true}
}

// NewNoColor creates a new Output instance without color output
func NewNoColor() *Output {
	return &Output{useColor: false}
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// Color returns the color code if colors are enabled
func (o *Output) Color(code string) string {
	if o.useColor {
		return code
	}
	return ""
}

func (o *Output) wrap(code string, text string) string {
	return fmt.Sprintf("%s%s%s", o.Color(code), text, o.Color(colorReset))
}

func (o *Output) symbolLine(symbol string, text string, style func(string) string) string {
	styledSymbol := style(symbol)
	if text == "" {
		return styledSymbol
	}
	return fmt.Sprintf("%s %s", styledSymbol, text)
}

// Red returns red colored text
func (o *Output) Red(text string) string {
	return o.wrap(colorRed, text)
}

// Green returns green colored text
func (o *Output) Green(text string) string {
	return o.wrap(colorGreen, text)
}

// Yellow returns yellow colored text
func (o *Output) Yellow(text string) string {
	return o.wrap(colorYellow, text)
}

// Blue returns blue colored text
func (o *Output) Blue(text string) string {
	return o.wrap(colorBlue, text)
}

// Bold returns bold text
func (o *Output) Bold(text string) string {
	return o.wrap(colorBold, text)
}

// Pass returns the pass symbol with optional text
func (o *Output) Pass(text string) string {
	return o.symbolLine(symbolPass, text, o.Green)
}

// Fail returns the fail symbol with optional text
func (o *Output) Fail(text string) string {
	return o.symbolLine(symbolFail, text, o.Red)
}

// Warn returns the warning symbol with optional text
func (o *Output) Warn(text string) string {
	return o.symbolLine(symbolWarn, text, o.Yellow)
}

// Info returns the info symbol with optional text
func (o *Output) Info(text string) string {
	return o.symbolLine(symbolInfo, text, o.Blue)
}

// SectionHeader prints a section header
func (o *Output) SectionHeader(title string) {
	fmt.Printf("\n%s\n\n", o.Bold(title))
}

// InfoLine prints an info line
func (o *Output) InfoLine(text string) {
	fmt.Printf("  %s %s\n", o.Info(""), text)
}

// Success prints a success message
func (o *Output) Success(text string) {
	fmt.Printf("  %s\n", o.Pass(text))
}

// Error prints an error message
func (o *Output) Error(text string) {
	fmt.Printf("  %s\n", o.Fail(text))
}

// Warning prints a warning message
func (o *Output) Warning(text string) {
	fmt.Printf("  %s\n", o.Warn(text))
}

// Printf prints formatted text
func (o *Output) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// Println prints a line
func (o *Output) Println(text string) {
	fmt.Println(text)
}

// Global instance for convenience
var std = New()

// Standard output functions
func Red(text string) string            { return std.Red(text) }
func Green(text string) string          { return std.Green(text) }
func Yellow(text string) string         { return std.Yellow(text) }
func Blue(text string) string           { return std.Blue(text) }
func Bold(text string) string           { return std.Bold(text) }
func Pass(text string) string           { return std.Pass(text) }
func Fail(text string) string           { return std.Fail(text) }
func Warn(text string) string           { return std.Warn(text) }
func Info(text string) string           { return std.Info(text) }
func SectionHeader(title string)        { std.SectionHeader(title) }
func InfoLine(text string)              { std.InfoLine(text) }
func Success(text string)               { std.Success(text) }
func Error(text string)                 { std.Error(text) }
func Warning(text string)               { std.Warning(text) }
func Printf(format string, args ...any) { std.Printf(format, args...) }
func Println(text string)               { std.Println(text) }
