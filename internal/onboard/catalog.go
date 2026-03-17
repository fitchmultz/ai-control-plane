// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Define the canonical onboarding tool catalog, defaults, and validation
//	rules used by prompts and workflow coordination.
//
// Responsibilities:
//   - Validate supported tool names.
//   - Resolve default auth modes, aliases, and model aliases.
//   - Enforce tool-specific mode support and deterministic prompt ordering.
//
// Scope:
//   - Onboarding tool metadata only.
//
// Usage:
//   - Used by the wizard and workflow before execution.
//
// Invariants/Assumptions:
//   - Tool defaults are declared once and shared across prompting and runtime.
//   - Interactive ordering must remain deterministic.
package onboard

import (
	"errors"
	"fmt"
)

var onboardingToolOrder = []string{"codex", "claude", "opencode", "cursor"}

var onboardingTools = map[string]toolSpec{
	"codex": {
		Name:        "codex",
		Summary:     "OpenAI Codex CLI through the gateway or direct OTEL visibility mode",
		DefaultMode: "subscription",
		DefaultAlias: func(string) string {
			return "codex-cli"
		},
		DefaultModel: func(mode string) string {
			if mode == "subscription" {
				return "chatgpt-gpt5.3-codex"
			}
			return "openai-gpt5.2"
		},
		Modes: []modeSpec{
			{Name: "subscription", Summary: "Use ChatGPT subscription upstream through the gateway"},
			{Name: "api-key", Summary: "Use API-key-based upstream providers through the gateway"},
			{Name: "direct", Summary: "Bypass the gateway and send OTEL telemetry only"},
		},
	},
	"claude": {
		Name:        "claude",
		Summary:     "Claude Code through the gateway with API-key or subscription-backed routing",
		DefaultMode: "api-key",
		DefaultAlias: func(mode string) string {
			if mode == "subscription" {
				return "claude-code-max"
			}
			return "claude-code"
		},
		DefaultModel: func(string) string {
			return "claude-haiku-4-5"
		},
		Modes: []modeSpec{
			{Name: "api-key", Summary: "Use Anthropic-compatible API-key routing through the gateway"},
			{Name: "subscription", Summary: "Use Claude subscription OAuth through the gateway with LiteLLM headers"},
		},
	},
	"opencode": {
		Name:        "opencode",
		Summary:     "OpenCode through the gateway using the OpenAI-compatible routed path",
		DefaultMode: "api-key",
		DefaultAlias: func(string) string {
			return "opencode-cli"
		},
		DefaultModel: func(string) string {
			return "openai-gpt5.2"
		},
		Modes: []modeSpec{
			{Name: "api-key", Summary: "Use OpenAI-compatible API-key routing through the gateway"},
		},
	},
	"cursor": {
		Name:        "cursor",
		Summary:     "Cursor IDE with gateway-routed OpenAI-compatible settings",
		DefaultMode: "api-key",
		DefaultAlias: func(string) string {
			return "cursor-cli"
		},
		DefaultModel: func(string) string {
			return "openai-gpt5.2"
		},
		Modes: []modeSpec{
			{Name: "api-key", Summary: "Use OpenAI-compatible API-key routing through the gateway"},
		},
	},
}

func lookupToolSpec(tool string) (toolSpec, error) {
	spec, ok := onboardingTools[tool]
	if !ok {
		return toolSpec{}, fmt.Errorf("unsupported tool: %s", tool)
	}
	return spec, nil
}

func orderedToolSpecs() []toolSpec {
	specs := make([]toolSpec, 0, len(onboardingToolOrder))
	for _, name := range onboardingToolOrder {
		specs = append(specs, onboardingTools[name])
	}
	return specs
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
	if opts.Alias == "" && opts.Mode != "direct" {
		opts.Alias = spec.DefaultAlias(opts.Mode)
	}
	if opts.Model == "" && opts.Mode != "direct" {
		opts.Model = spec.DefaultModel(opts.Mode)
	}
	return opts, nil
}

func validateMode(spec toolSpec, mode string) error {
	if mode == "" {
		return errors.New("mode is required")
	}
	for _, supported := range spec.Modes {
		if supported.Name == mode {
			return nil
		}
	}
	return fmt.Errorf("unsupported mode %q for %s", mode, spec.Name)
}
