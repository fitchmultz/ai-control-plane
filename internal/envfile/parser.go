// Package envfile provides strict .env parsing without shell execution.
//
// Purpose:
//
//	Read repository-managed .env files as configuration data only.
//
// Responsibilities:
//   - Parse KEY=VALUE entries without shell evaluation.
//   - Ignore blank lines and comments safely.
//   - Reject malformed keys and invalid non-comment lines.
//
// Non-scope:
//   - Does not execute shell expansions or command substitution.
//   - Does not support shell syntax such as `export KEY=VALUE`.
//
// Invariants/Assumptions:
//   - Keys follow POSIX-style environment variable naming.
//   - Values are returned literally, aside from paired quote trimming.
package envfile

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// LookupFile returns the literal value for a requested key in an env file.
func LookupFile(path string, wantKey string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return "", false, fmt.Errorf("%s:%d: invalid env line: expected KEY=VALUE", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		if !envKeyPattern.MatchString(key) {
			return "", false, fmt.Errorf("%s:%d: invalid env key %q", path, lineNumber, key)
		}

		if key != wantKey {
			continue
		}

		return trimPairedQuotes(strings.TrimSpace(value)), true, nil
	}

	if err := scanner.Err(); err != nil {
		return "", false, err
	}

	return "", false, nil
}

func trimPairedQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}
