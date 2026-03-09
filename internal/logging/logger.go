// Package logging provides structured workflow logging for typed operator flows.
//
// Purpose:
//   - Centralize structured stderr-oriented logging for command and workflow
//     execution without replacing final human renderers.
//
// Responsibilities:
//   - Build text or JSON slog loggers with stable repository defaults.
//   - Carry loggers through context so internal workflows can emit events.
//   - Provide small helpers for common structured fields.
//
// Scope:
//   - Workflow/event logging only; final report rendering stays elsewhere.
//
// Usage:
//   - Seed a logger in command dispatch, attach it to context, and retrieve it
//     from internal packages with FromContext.
//
// Invariants/Assumptions:
//   - Loggers default to text output and info level when not specified.
//   - Structured logs target stderr-oriented writers by caller convention.
package logging

import (
	"context"
	"io"
	"log/slog"
)

type contextKey struct{}

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Options struct {
	Writer io.Writer
	Format Format
	Level  slog.Leveler
	Attrs  []slog.Attr
}

func New(opts Options) *slog.Logger {
	writer := opts.Writer
	if writer == nil {
		writer = io.Discard
	}
	level := opts.Level
	if level == nil {
		level = slog.LevelInfo
	}
	handlerOptions := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, handlerOptions)
	default:
		handler = slog.NewTextHandler(writer, handlerOptions)
	}
	logger := slog.New(handler)
	if len(opts.Attrs) == 0 {
		return logger
	}
	args := make([]any, 0, len(opts.Attrs))
	for _, attr := range opts.Attrs {
		args = append(args, attr)
	}
	return logger.With(args...)
}

func Nop() *slog.Logger {
	return New(Options{Writer: io.Discard})
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		logger = Nop()
	}
	return context.WithValue(ctx, contextKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return Nop()
}

func Err(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}
	return slog.String("error", err.Error())
}
