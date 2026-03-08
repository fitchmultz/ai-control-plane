// parser.go - Key generation argument parser
//
// Purpose: Parse and validate key generation command arguments
//
// Responsibilities:
//   - Parse command-line arguments
//   - Convert string values to typed values
//   - Return structured configuration
//
// Non-scope:
//   - Does not validate business logic (see validator.go)
//   - Does not generate keys
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package keygen

import (
	"fmt"
	"strconv"
	"strings"
)

// Config holds parsed key generation configuration
type Config struct {
	Alias    string
	Budget   float64
	RPM      int
	TPM      int
	Parallel int
	Duration string
	Role     string
	DryRun   bool
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Budget:   10.00,
		Duration: "30d",
		Role:     "",
	}
}

// ParseArgs parses key generation command arguments
func ParseArgs(args []string) (*Config, error) {
	config := DefaultConfig()

	if len(args) == 0 {
		return nil, fmt.Errorf("alias is required")
	}

	// First argument is alias (unless it's a flag)
	if !strings.HasPrefix(args[0], "-") {
		config.Alias = args[0]
		args = args[1:]
	}

	// Parse remaining flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--budget":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --budget")
			}
			b, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid budget: %s", args[i+1])
			}
			config.Budget = b
			i++
		case "--rpm":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --rpm")
			}
			r, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid RPM: %s", args[i+1])
			}
			config.RPM = r
			i++
		case "--tpm":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --tpm")
			}
			t, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid TPM: %s", args[i+1])
			}
			config.TPM = t
			i++
		case "--parallel":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --parallel")
			}
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid parallel: %s", args[i+1])
			}
			config.Parallel = p
			i++
		case "--duration":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --duration")
			}
			config.Duration = args[i+1]
			i++
		case "--role":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --role")
			}
			config.Role = args[i+1]
			i++
		case "--dry-run":
			config.DryRun = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, fmt.Errorf("unknown option: %s", args[i])
			}
		}
	}

	return config, nil
}
