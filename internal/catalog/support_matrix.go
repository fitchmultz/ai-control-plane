// Package catalog loads tracked repository configuration data.
//
// Purpose:
//
//	Provide typed loaders for stable, committed repository configuration used
//	by completions, validations, operator workflows, and generated references.
//
// Responsibilities:
//   - Load the machine-readable support matrix used by docs and lint gates.
//   - Expose deterministic helpers over supported/public document paths.
//
// Scope:
//   - Read-only access to tracked repository configuration under demo/config
//     and docs/.
//
// Usage:
//   - Used by doc generation and support-surface linting.
//
// Invariants/Assumptions:
//   - The tracked support matrix is the source of truth for public support levels.
package catalog

import "strings"

// SupportMatrix is the tracked support-level source of truth.
type SupportMatrix struct {
	PublicDocs      []string         `yaml:"public_docs"`
	ReferenceDocs   []string         `yaml:"reference_docs"`
	IncubatingTerms []string         `yaml:"incubating_terms"`
	Surfaces        []SupportSurface `yaml:"surfaces"`
}

// SupportSurface captures one public or incubating product surface.
type SupportSurface struct {
	ID         string   `yaml:"id"`
	Label      string   `yaml:"label"`
	Status     string   `yaml:"status"`
	Summary    string   `yaml:"summary"`
	Owner      string   `yaml:"owner"`
	Validation []string `yaml:"validation"`
}

// LoadSupportMatrix loads the tracked support matrix YAML.
func LoadSupportMatrix(path string) (SupportMatrix, error) {
	var config SupportMatrix
	if err := loadYAMLFile(path, &config); err != nil {
		return SupportMatrix{}, err
	}
	return config, nil
}

// SupportedSurfaces returns only supported public surfaces.
func (m SupportMatrix) SupportedSurfaces() []SupportSurface {
	return filterSupportSurfaces(m.Surfaces, "supported")
}

// IncubatingSurfaces returns only incubating surfaces.
func (m SupportMatrix) IncubatingSurfaces() []SupportSurface {
	return filterSupportSurfaces(m.Surfaces, "incubating")
}

func filterSupportSurfaces(values []SupportSurface, status string) []SupportSurface {
	filtered := make([]SupportSurface, 0, len(values))
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value.Status), status) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
