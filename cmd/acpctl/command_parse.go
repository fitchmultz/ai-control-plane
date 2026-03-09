// command_parse.go - argv parsing for the acpctl command platform.
//
// Purpose:
//
//	Parse CLI argv into typed command invocations with validated options and
//	arguments.
//
// Responsibilities:
//   - Resolve the active command path from argv tokens.
//   - Parse leaf options, arguments, and trailing passthrough args.
//   - Enforce required options and argument arity.
//
// Scope:
//   - Invocation and leaf-input parsing only.
//
// Usage:
//   - Used by main.go and typed command adapters.
//
// Invariants/Assumptions:
//   - Help tokens short-circuit parsing before backend execution.
//   - Unknown options fail unless the command explicitly allows trailing args.
package main

import (
	"fmt"
	"strings"
)

func parseInvocation(args []string) (commandInvocation, error) {
	spec, err := loadCommandSpec()
	if err != nil {
		return commandInvocation{}, err
	}
	path := []*commandSpec{spec.Root}
	current := spec.Root
	remaining := args
	for len(remaining) > 0 {
		token := remaining[0]
		if isHelpToken(token) {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		next := findChildCommand(current, token)
		if next == nil {
			break
		}
		current = next
		path = append(path, current)
		remaining = remaining[1:]
	}
	if current == spec.Root {
		if len(remaining) == 0 {
			return commandInvocation{Spec: spec.Root, Path: path, HelpOnly: true}, nil
		}
		return commandInvocation{}, &commandLookupError{Kind: "root", Name: remaining[0]}
	}
	if len(current.Children) > 0 {
		if len(remaining) == 0 {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		if isHelpToken(remaining[0]) {
			return commandInvocation{Spec: current, Path: path, HelpOnly: true}, nil
		}
		return commandInvocation{}, &commandLookupError{Kind: "subcommand", Path: commandPathKey(path[1:]), Name: remaining[0]}
	}
	input, helpOnly, err := parseLeafInput(current, remaining)
	if err != nil {
		return commandInvocation{}, err
	}
	return commandInvocation{
		Spec:     current,
		Path:     path,
		Input:    input,
		HelpOnly: helpOnly,
	}, nil
}

func parseLeafInput(spec *commandSpec, args []string) (parsedCommandInput, bool, error) {
	input := parsedCommandInput{flags: make(map[string][]string)}
	optionByLong := make(map[string]commandOptionSpec, len(spec.Options))
	optionByShort := make(map[string]commandOptionSpec, len(spec.Options))
	for _, option := range spec.Options {
		optionByLong[option.Name] = option
		if option.Short != "" {
			optionByShort[option.Short] = option
		}
	}

	argumentIndex := 0
	for index := 0; index < len(args); index++ {
		token := args[index]
		if isHelpToken(token) {
			return parsedCommandInput{}, true, nil
		}
		if spec.AllowTrailingArgs && token == "--" {
			input.trailing = append(input.trailing, args[index+1:]...)
			break
		}
		if strings.HasPrefix(token, "--") {
			name, explicitValue, hasExplicitValue := strings.Cut(strings.TrimPrefix(token, "--"), "=")
			option, ok := optionByLong[name]
			if !ok {
				if spec.AllowTrailingArgs {
					input.trailing = append(input.trailing, args[index:]...)
					break
				}
				return parsedCommandInput{}, false, fmt.Errorf("unknown option: --%s", name)
			}
			value, consumed, err := parseOptionValue(option, token, explicitValue, hasExplicitValue, args, index)
			if err != nil {
				return parsedCommandInput{}, false, err
			}
			input.flags[option.Name] = append(input.flags[option.Name], value)
			index += consumed
			continue
		}
		if strings.HasPrefix(token, "-") && token != "-" {
			shortName, explicitValue, hasExplicitValue := strings.Cut(strings.TrimPrefix(token, "-"), "=")
			option, ok := optionByShort[shortName]
			if !ok {
				if spec.AllowTrailingArgs {
					input.trailing = append(input.trailing, args[index:]...)
					break
				}
				return parsedCommandInput{}, false, fmt.Errorf("unknown option: -%s", shortName)
			}
			value, consumed, err := parseOptionValue(option, token, explicitValue, hasExplicitValue, args, index)
			if err != nil {
				return parsedCommandInput{}, false, err
			}
			input.flags[option.Name] = append(input.flags[option.Name], value)
			index += consumed
			continue
		}
		if argumentIndex < len(spec.Arguments) {
			input.arguments = append(input.arguments, token)
			if !spec.Arguments[argumentIndex].Repeatable {
				argumentIndex++
			}
			continue
		}
		if spec.AllowTrailingArgs {
			input.trailing = append(input.trailing, args[index:]...)
			break
		}
		return parsedCommandInput{}, false, fmt.Errorf("unexpected argument: %s", token)
	}

	for _, option := range spec.Options {
		if option.Required && len(input.flags[option.Name]) == 0 {
			return parsedCommandInput{}, false, fmt.Errorf("missing required option: --%s", option.Name)
		}
		if !option.Repeatable && len(input.flags[option.Name]) > 1 {
			return parsedCommandInput{}, false, fmt.Errorf("option may only be provided once: --%s", option.Name)
		}
	}
	requiredArgs := 0
	for _, argument := range spec.Arguments {
		if argument.Required {
			requiredArgs++
		}
	}
	if len(input.arguments) < requiredArgs {
		missing := spec.Arguments[len(input.arguments)].Name
		return parsedCommandInput{}, false, fmt.Errorf("missing required argument: %s", missing)
	}
	return input, false, nil
}

func parseOptionValue(option commandOptionSpec, token string, explicitValue string, hasExplicitValue bool, args []string, index int) (string, int, error) {
	switch option.Type {
	case optionValueBool:
		if hasExplicitValue {
			value := strings.ToLower(strings.TrimSpace(explicitValue))
			switch value {
			case "true", "1", "yes", "on":
				return "true", 0, nil
			case "false", "0", "no", "off":
				return "false", 0, nil
			default:
				return "", 0, fmt.Errorf("invalid boolean value for %s: %s", token, explicitValue)
			}
		}
		return "true", 0, nil
	default:
		if hasExplicitValue {
			return explicitValue, 0, nil
		}
		if index+1 >= len(args) {
			return "", 0, fmt.Errorf("missing value for %s", token)
		}
		return args[index+1], 1, nil
	}
}
