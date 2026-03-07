// readiness_plan.go - Typed readiness gate plan loading.
//
// Purpose:
//
//	Load the canonical readiness evidence gate plan from tracked repository
//	configuration instead of hard-coding it in command logic.
//
// Responsibilities:
//   - Define the YAML-backed readiness gate plan schema.
//   - Load and normalize gate definitions from demo/config/readiness_evidence.yaml.
//   - Materialize execution-ready gate specs for the current run options.
//
// Scope:
//   - Gate plan loading and expansion only.
//
// Usage:
//   - Called by readiness_evidence.go before executing gates.
//
// Invariants/Assumptions:
//   - The tracked readiness plan is the single source of truth for gate membership.
//   - Gate commands are Make argument vectors that run from the repository root.
package release

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const readinessPlanRelativePath = "demo/config/readiness_evidence.yaml"

type readinessPlanFile struct {
	Gates []readinessPlanGate `yaml:"gates"`
}

type readinessPlanGate struct {
	ID             string   `yaml:"id"`
	Title          string   `yaml:"title"`
	Required       bool     `yaml:"required"`
	LogName        string   `yaml:"log_name"`
	Command        []string `yaml:"command"`
	Notes          string   `yaml:"notes"`
	ProductionOnly bool     `yaml:"production_only"`
}

func loadReadinessPlan(repoRoot string) ([]readinessPlanGate, error) {
	path := filepath.Join(repoRoot, readinessPlanRelativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read readiness plan: %s not found", path)
		}
		return nil, fmt.Errorf("read readiness plan: %w", err)
	}
	var plan readinessPlanFile
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse readiness plan: %w", err)
	}
	if len(plan.Gates) == 0 {
		return nil, fmt.Errorf("readiness plan %s must define at least one gate", path)
	}
	return plan.Gates, nil
}

func materializeReadinessGates(opts ReadinessOptions, productionEnabled bool) ([]readinessGateSpec, error) {
	plan, err := loadReadinessPlan(opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	gates := make([]readinessGateSpec, 0, len(plan))
	for _, gate := range plan {
		if gate.ProductionOnly && !opts.IncludeProduction {
			continue
		}
		required := gate.Required
		if gate.ProductionOnly && !productionEnabled {
			required = false
		}
		command := append([]string(nil), gate.Command...)
		command = expandReadinessCommandArgs(command, opts)
		gates = append(gates, readinessGateSpec{
			ID:             gate.ID,
			Title:          gate.Title,
			Required:       required,
			LogName:        gate.LogName,
			Command:        command,
			Notes:          gate.Notes,
			ProductionOnly: gate.ProductionOnly,
		})
	}
	return gates, nil
}

func expandReadinessCommandArgs(args []string, opts ReadinessOptions) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "${BUNDLE_VERSION}":
			expanded = append(expanded, opts.BundleVersion)
		case "${SECRETS_ENV_FILE}":
			expanded = append(expanded, opts.SecretsEnvFile)
		default:
			expanded = append(expanded, arg)
		}
	}
	return expanded
}
