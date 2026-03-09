// command_help.go - Help rendering for the acpctl command platform.
//
// Purpose:
//
//	Render usage, arguments, options, examples, and exit-code help from the
//	typed command metadata tree.
//
// Responsibilities:
//   - Build stable usage strings from the current command path.
//   - Render deterministic command help sections and examples.
//
// Scope:
//   - Human help/usage rendering only.
//
// Usage:
//   - Called by main.go, parse errors, and explicit help paths.
//
// Invariants/Assumptions:
//   - Help text is generated from the same command metadata used for parsing.
package main

import (
	"fmt"
	"os"
	"strings"
)

func renderUsage(path []*commandSpec) string {
	current := path[len(path)-1]
	usage := commandPathLabel(path[1:])
	if len(current.Children) > 0 {
		return usage + " <subcommand>"
	}
	if len(current.Options) > 0 {
		usage += " [options]"
	}
	for _, argument := range current.Arguments {
		if argument.Required {
			usage += " <" + argument.Name + ">"
		} else {
			usage += " [" + argument.Name + "]"
		}
		if argument.Repeatable {
			usage += "..."
		}
	}
	if current.AllowTrailingArgs {
		usage += " [args...]"
	}
	return usage
}

func printCommandHelp(out *os.File, path []*commandSpec) {
	current := path[len(path)-1]
	fmt.Fprintf(out, "Usage: %s\n\n", renderUsage(path))
	description := strings.TrimSpace(current.Description)
	if description == "" {
		description = current.Summary
	}
	if description != "" {
		fmt.Fprintln(out, description)
		fmt.Fprintln(out)
	}
	if len(current.Children) > 0 {
		fmt.Fprintln(out, "Commands:")
		for _, child := range current.Children {
			if child.Hidden {
				continue
			}
			fmt.Fprintf(out, "  %-22s %s\n", child.Name, child.Summary)
		}
	} else {
		if len(current.Arguments) > 0 {
			fmt.Fprintln(out, "Arguments:")
			for _, argument := range current.Arguments {
				fmt.Fprintf(out, "  %-22s %s\n", argument.Name, argument.Summary)
			}
			fmt.Fprintln(out)
		}
		if len(current.Options) > 0 {
			fmt.Fprintln(out, "Options:")
			for _, option := range current.Options {
				labels := []string{"--" + option.Name}
				if option.Short != "" {
					labels = append(labels, "-"+option.Short)
				}
				label := strings.Join(labels, ", ")
				if option.Type != optionValueBool {
					valueName := option.ValueName
					if valueName == "" {
						valueName = strings.ToUpper(option.Name)
					}
					label += " " + valueName
				}
				summary := option.Summary
				if option.DefaultText != "" {
					summary += " (default: " + option.DefaultText + ")"
				}
				fmt.Fprintf(out, "  %-22s %s\n", label, summary)
			}
			fmt.Fprintln(out)
		}
	}
	for _, section := range current.Sections {
		fmt.Fprintf(out, "%s:\n", section.Title)
		for _, line := range section.Lines {
			fmt.Fprintf(out, "  %s\n", line)
		}
		fmt.Fprintln(out)
	}
	if len(current.Examples) > 0 {
		fmt.Fprintln(out, "Examples:")
		for _, example := range current.Examples {
			fmt.Fprintf(out, "  %s\n", example)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Exit codes:")
	fmt.Fprintln(out, "  0   Success")
	fmt.Fprintln(out, "  1   Domain non-success")
	fmt.Fprintln(out, "  2   Prerequisites not ready")
	fmt.Fprintln(out, "  3   Runtime/internal error")
	fmt.Fprintln(out, "  64  Usage error")
}
