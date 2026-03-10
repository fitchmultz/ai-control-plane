// cmd_bind_helpers.go - Shared command binder helpers.
//
// Purpose:
//   - Centralize small command-layer binder patterns reused across multiple
//     command specs.
//
// Responsibilities:
//   - Provide reusable no-option binders for native commands.
//   - Compose small native command spec definitions from shared defaults.
//   - Keep repeated binder and backend boilerplate out of leaf command definitions.
//
// Scope:
//   - Command-layer binder helpers only.
//
// Usage:
//   - Used by command specs that do not need parsed options or need to return a
//     fixed typed payload.
//
// Invariants/Assumptions:
//   - Helpers remain behaviorally trivial and side-effect free.
package main

import "context"

type nativeCommandConfig struct {
	Name              string
	Summary           string
	Description       string
	Examples          []string
	Options           []commandOptionSpec
	Arguments         []commandArgumentSpec
	Sections          []commandHelpSection
	Children          []*commandSpec
	Hidden            bool
	AllowTrailingArgs bool
	Bind              func(commandBindContext, parsedCommandInput) (any, error)
	Run               func(context.Context, commandRunContext, any) int
}

func bindNoOptions(_ commandBindContext, _ parsedCommandInput) (any, error) {
	return struct{}{}, nil
}

func bindStaticOptions[T any](value T) func(commandBindContext, parsedCommandInput) (any, error) {
	return func(_ commandBindContext, _ parsedCommandInput) (any, error) {
		return value, nil
	}
}

func bindParsed[T any](fn func(parsedCommandInput) (T, error)) func(commandBindContext, parsedCommandInput) (any, error) {
	return func(_ commandBindContext, input parsedCommandInput) (any, error) {
		value, err := fn(input)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func bindParsedValue[T any](fn func(parsedCommandInput) T) func(commandBindContext, parsedCommandInput) (any, error) {
	return func(_ commandBindContext, input parsedCommandInput) (any, error) {
		return fn(input), nil
	}
}

func bindRepoParsed[T any](fn func(commandBindContext, parsedCommandInput) (T, error)) func(commandBindContext, parsedCommandInput) (any, error) {
	return func(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
		value, err := fn(bindCtx, input)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}

func newNativeCommandSpec(config nativeCommandConfig) *commandSpec {
	description := config.Description
	if description == "" {
		description = config.Summary + "."
	}
	bind := config.Bind
	if bind == nil {
		bind = bindNoOptions
	}
	return &commandSpec{
		Name:              config.Name,
		Summary:           config.Summary,
		Description:       description,
		Examples:          append([]string(nil), config.Examples...),
		Hidden:            config.Hidden,
		Options:           append([]commandOptionSpec(nil), config.Options...),
		Arguments:         append([]commandArgumentSpec(nil), config.Arguments...),
		Sections:          append([]commandHelpSection(nil), config.Sections...),
		Children:          append([]*commandSpec(nil), config.Children...),
		AllowTrailingArgs: config.AllowTrailingArgs,
		Backend: commandBackend{
			Kind:       commandBackendNative,
			NativeBind: bind,
			NativeRun:  config.Run,
		},
	}
}

func newNativeLeafCommandSpec(name string, summary string, run func(context.Context, commandRunContext, any) int) *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:    name,
		Summary: summary,
		Run:     run,
	})
}
