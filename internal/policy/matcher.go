// Package policy defines canonical repository validation and scan scope.
//
// Purpose:
//   - Provide recursive path-scope traversal for canonical repository policies.
//
// Responsibilities:
//   - Match repository-relative glob patterns with `**` semantics.
//   - Walk repository files against shared include/exclude scopes.
//   - Normalize target paths for downstream validators.
//
// Scope:
//   - Repository-local path matching and traversal only.
//
// Usage:
//   - Used by deployment-surface expansion and license scan scopes.
//
// Invariants/Assumptions:
//   - Matching is slash-normalized for deterministic cross-platform behavior.
//   - `.git`, `vendor`, and `node_modules` remain outside canonical scan scope.
package policy

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type PathScope struct {
	Include []string
	Exclude []string
}

func WalkScopeFiles(repoRoot string, scope PathScope) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(relPath, scope) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(scope.Include) > 0 && !MatchAnyGlob(relPath, scope.Include) {
			return nil
		}
		if MatchAnyGlob(relPath, scope.Exclude) {
			return nil
		}
		paths = append(paths, relPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return uniqStrings(paths), nil
}

func MatchAnyGlob(relPath string, patterns []string) bool {
	normalizedPath := filepath.ToSlash(relPath)
	for _, pattern := range patterns {
		if globMatch(normalizedPath, pattern) {
			return true
		}
	}
	return false
}

func shouldSkipDir(relPath string, scope PathScope) bool {
	base := filepath.Base(relPath)
	switch base {
	case ".git", "vendor", "node_modules":
		return true
	}
	return MatchAnyGlob(relPath, scope.Exclude) || MatchAnyGlob(relPath+"/", scope.Exclude)
}

func globMatch(relPath string, pattern string) bool {
	const (
		doubleStarDirToken = "<<double-star-dir>>"
		doubleStarToken    = "<<double-star>>"
	)
	normalizedPattern := filepath.ToSlash(strings.TrimSpace(pattern))
	if normalizedPattern == "" {
		return false
	}
	normalizedPattern = strings.ReplaceAll(normalizedPattern, "**/", doubleStarDirToken)
	normalizedPattern = strings.ReplaceAll(normalizedPattern, "**", doubleStarToken)
	regexPattern := regexp.QuoteMeta(normalizedPattern)
	regexPattern = strings.ReplaceAll(regexPattern, regexp.QuoteMeta(doubleStarDirToken), `(?:.*/)?`)
	regexPattern = strings.ReplaceAll(regexPattern, regexp.QuoteMeta(doubleStarToken), `.*`)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `[^/]*`)
	regexPattern = strings.ReplaceAll(regexPattern, `\?`, `[^/]`)
	matched, err := regexp.MatchString("^"+regexPattern+"$", relPath)
	return err == nil && matched
}
