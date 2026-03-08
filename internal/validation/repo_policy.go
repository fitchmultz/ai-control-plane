// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Enforce repository source-policy invariants that CI must keep green.
//
// Responsibilities:
//   - Check Go file header-comment shape.
//   - Prevent direct environment access outside `internal/config`.
//
// Scope:
//   - Source-tree policy checks only.
//
// Usage:
//   - Run through `acpctl validate headers` and `acpctl validate env-access`.
//
// Invariants/Assumptions:
//   - Policy checks are deterministic and operate on tracked source files only.
//   - Enforcement excludes test files from direct env access migration checks.
package validation

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var requiredHeaderFields = []string{
	"Purpose:",
	"Responsibilities:",
	"Scope:",
	"Usage:",
	"Invariants/Assumptions:",
}

func ValidateGoHeaders(repoRoot string) ([]string, error) {
	issues := make([]string, 0)
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch filepath.Base(path) {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !hasRequiredHeader(data) {
			issues = append(issues, fmt.Sprintf("%s: missing required top-of-file purpose header fields", relPath))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(issues)
	return issues, nil
}

func hasRequiredHeader(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	headerLines := make([]string, 0)
	seenComment := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if seenComment {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "package ") {
			break
		}
		if !strings.HasPrefix(trimmed, "//") {
			break
		}
		seenComment = true
		headerLines = append(headerLines, trimmed)
	}
	if len(headerLines) == 0 {
		return false
	}
	header := strings.Join(headerLines, "\n")
	for _, field := range requiredHeaderFields {
		if !strings.Contains(header, field) {
			return false
		}
	}
	return true
}

func ValidateDirectEnvAccess(repoRoot string) ([]string, error) {
	issues := make([]string, 0)
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch filepath.Base(path) {
			case ".git", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "internal/validation/repo_policy.go" {
			return nil
		}
		if strings.HasPrefix(relPath, "internal/config/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, forbidden := range []string{"os.Getenv(", "os.LookupEnv(", "envfile.LookupFile("} {
			if strings.Contains(content, forbidden) {
				issues = append(issues, fmt.Sprintf("%s: direct config access %q is forbidden outside internal/config", relPath, forbidden))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(issues)
	return issues, nil
}
