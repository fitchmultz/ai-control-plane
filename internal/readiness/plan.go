// plan.go - Typed readiness gate plan loading.
//
// Purpose:
//
//	Load the canonical readiness evidence gate plan from tracked repository
//	configuration.
//
// Responsibilities:
//   - Define the YAML-backed readiness gate plan schema.
//   - Load and normalize gate definitions from demo/config/readiness_evidence.yaml.
//   - Materialize execution-ready gate specs for the current run options.
//
// Scope:
//   - Gate plan loading and command expansion only.
//
// Usage:
//   - Called by readiness evidence workflows before executing gates.
//
// Invariants/Assumptions:
//   - The tracked readiness plan is the single source of truth for gate membership.
//   - Gate commands are Make argument vectors that run from the repository root.
package readiness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const planRelativePath = "demo/config/readiness_evidence.yaml"

type planFile struct {
	Gates []planGate `yaml:"gates"`
}

type planGate struct {
	ID             string   `yaml:"id"`
	Title          string   `yaml:"title"`
	Required       bool     `yaml:"required"`
	LogName        string   `yaml:"log_name"`
	Command        []string `yaml:"command"`
	Notes          string   `yaml:"notes"`
	ProductionOnly bool     `yaml:"production_only"`
}

func loadPlan(repoRoot string) ([]planGate, error) {
	path := filepath.Join(repoRoot, planRelativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read readiness plan: %s not found", path)
		}
		return nil, fmt.Errorf("read readiness plan: %w", err)
	}
	var plan planFile
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parse readiness plan: %w", err)
	}
	if len(plan.Gates) == 0 {
		return nil, fmt.Errorf("readiness plan %s must define at least one gate", path)
	}
	return plan.Gates, nil
}

func materializeGates(opts Options, productionEnabled bool) ([]gateSpec, error) {
	plan, err := loadPlan(opts.RepoRoot)
	if err != nil {
		return nil, err
	}
	gates := make([]gateSpec, 0, len(plan))
	for _, gate := range plan {
		if gate.ProductionOnly && !opts.IncludeProduction {
			continue
		}
		required := gate.Required
		if gate.ProductionOnly && !productionEnabled {
			required = false
		}
		gates = append(gates, gateSpec{
			ID:             gate.ID,
			Title:          gate.Title,
			Required:       required,
			LogName:        gate.LogName,
			Command:        expandCommandArgs(gate.Command, opts),
			Notes:          gate.Notes,
			ProductionOnly: gate.ProductionOnly,
		})
	}
	return gates, nil
}

func expandCommandArgs(args []string, opts Options) []string {
	replacer := strings.NewReplacer(
		"${BUNDLE_VERSION}", opts.BundleVersion,
		"${SECRETS_ENV_FILE}", opts.SecretsEnvFile,
	)
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		expanded = append(expanded, replacer.Replace(arg))
	}
	return expanded
}
