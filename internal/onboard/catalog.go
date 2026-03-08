// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Define the canonical onboarding tool catalog, defaults, and validation
//	rules used by parsing and workflow coordination.
//
// Responsibilities:
//   - Validate supported tool names.
//   - Resolve default auth modes and model aliases.
//   - Enforce tool-specific mode support.
//
// Scope:
//   - Onboarding tool metadata only.
//
// Usage:
//   - Used by ParseArgs and Run before workflow execution.
//
// Invariants/Assumptions:
//   - Tool defaults are declared once and shared across parsing and runtime.
package onboard

import (
	"errors"
	"fmt"
)

var onboardingTools = map[string]toolSpec{
	"codex": {
		Name:           "codex",
		DefaultMode:    "subscription",
		SupportedModes: modeSet("subscription", "api-key", "direct"),
		DefaultModel: func(mode string) string {
			if mode == "subscription" {
				return "chatgpt-gpt5.3-codex"
			}
			return "openai-gpt5.2"
		},
	},
	"claude": {
		Name:           "claude",
		DefaultMode:    "api-key",
		SupportedModes: modeSet("api-key"),
		DefaultModel: func(string) string {
			return "claude-haiku-4-5"
		},
	},
	"opencode": {
		Name:           "opencode",
		DefaultMode:    "api-key",
		SupportedModes: modeSet("api-key"),
		DefaultModel: func(string) string {
			return "openai-gpt5.2"
		},
	},
	"cursor": {
		Name:           "cursor",
		DefaultMode:    "api-key",
		SupportedModes: modeSet("api-key"),
		DefaultModel: func(string) string {
			return "openai-gpt5.2"
		},
	},
	"copilot": {
		Name:           "copilot",
		DefaultMode:    "api-key",
		SupportedModes: modeSet("api-key"),
		DefaultModel: func(string) string {
			return "openai-gpt5.2"
		},
	},
}

func modeSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func lookupToolSpec(tool string) (toolSpec, error) {
	spec, ok := onboardingTools[tool]
	if !ok {
		return toolSpec{}, fmt.Errorf("unsupported tool: %s", tool)
	}
	return spec, nil
}

func resolveDefaults(opts Options) (Options, error) {
	spec, err := lookupToolSpec(opts.Tool)
	if err != nil {
		return Options{}, err
	}
	if opts.Mode == "" {
		opts.Mode = spec.DefaultMode
	}
	if err := validateMode(spec, opts.Mode); err != nil {
		return Options{}, err
	}
	if opts.Model == "" {
		opts.Model = spec.DefaultModel(opts.Mode)
	}
	return opts, nil
}

func validateMode(spec toolSpec, mode string) error {
	if mode == "" {
		return errors.New("mode is required")
	}
	if _, ok := spec.SupportedModes[mode]; ok {
		return nil
	}
	return fmt.Errorf("unsupported mode %q for %s", mode, spec.Name)
}
