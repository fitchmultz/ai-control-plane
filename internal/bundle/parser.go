// parser.go - Release bundle command-line argument parser.
//
// Purpose: Parse and validate release bundle command arguments
//
// Responsibilities:
//   - Parse command-line arguments for build/verify commands
//   - Validate version strings and paths
//   - Return structured configuration
//
// Non-scope:
//   - Does not execute build or verify operations
//   - Does not interact with filesystem
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.
package bundle

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Config holds parsed release bundle configuration
type Config struct {
	Command   string
	Version   string
	OutputDir string
	Bundle    string
	Verbose   bool
}

// ParseArgs parses release bundle command arguments
func ParseArgs(args []string, repoRoot string, defaultVersionFn func(string) string) (*Config, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	config := &Config{
		Version:   defaultVersionFn(repoRoot),
		OutputDir: filepath.Join(repoRoot, "demo/logs/release-bundles"),
	}

	// First argument is the command
	config.Command = args[0]

	// Parse remaining arguments
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--version":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --version")
			}
			config.Version = args[i+1]
			i++
		case "--output-dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --output-dir")
			}
			config.OutputDir = args[i+1]
			i++
		case "--bundle":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --bundle")
			}
			config.Bundle = args[i+1]
			i++
		case "--verbose":
			config.Verbose = true
		case "--help", "-h":
			// Help is handled at caller level
			return config, nil
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, fmt.Errorf("unknown option: %s", args[i])
			}
		}
	}

	return config, nil
}
