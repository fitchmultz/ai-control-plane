// parser.go - Release bundle command-line argument parser
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
package release

import (
	"fmt"
	"os"
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

// ValidateVersion checks if version string is valid
func ValidateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	for _, c := range version {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '_' || c == '-') {
			return fmt.Errorf("version must match [A-Za-z0-9._-]+ (no path separators or spaces)")
		}
	}
	return nil
}

// GetDefaultVersion returns default version from git or "dev"
func GetDefaultVersion(repoRoot string) string {
	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return "dev"
	}

	headFile := filepath.Join(gitDir, "HEAD")
	data, err := os.ReadFile(headFile)
	if err != nil {
		return "dev"
	}

	ref := strings.TrimSpace(string(data))
	if after, ok := strings.CutPrefix(ref, "ref: "); ok {
		refPath := filepath.Join(gitDir, after)
		data, err = os.ReadFile(refPath)
		if err != nil {
			return "dev"
		}
		sha := strings.TrimSpace(string(data))
		if len(sha) >= 7 {
			return sha[:7]
		}
	} else {
		// Detached HEAD
		sha := strings.TrimSpace(ref)
		if len(sha) >= 7 {
			return sha[:7]
		}
	}
	return "dev"
}
