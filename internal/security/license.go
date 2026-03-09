// Package security owns typed repository security validation logic.
//
// Purpose:
//   - Enforce the repository's third-party license boundary policy.
//
// Responsibilities:
//   - Validate required fields in the license policy document.
//   - Enumerate files from the policy-defined scan scope.
//   - Detect restricted component references by path or file content.
//
// Scope:
//   - Read-only license-boundary validation only.
//
// Usage:
//   - Called by `acpctl validate license`.
//
// Invariants/Assumptions:
//   - Scan scope uses the shared recursive policy walker.
//   - Findings are repository-relative and stably sorted.
package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/policy"
	validationissues "github.com/mitchfultz/ai-control-plane/internal/validation"
)

func ValidateLicensePolicy(repoRoot string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "policy", "THIRD_PARTY_LICENSE_MATRIX.json"))
	if err != nil {
		return nil, err
	}
	var policyDoc LicensePolicy
	if err := json.Unmarshal(data, &policyDoc); err != nil {
		return nil, err
	}
	if policyDoc.SchemaVersion == nil || policyDoc.PolicyID == nil || len(policyDoc.ScanScope.Include) == 0 || len(policyDoc.RestrictedComponents) == 0 {
		issues := validationissues.NewIssues(1)
		issues.Add("docs/policy/THIRD_PARTY_LICENSE_MATRIX.json: missing required policy fields")
		return issues.Sorted(), nil
	}
	return findRestrictedLicenseReferences(repoRoot, policyDoc)
}

func findRestrictedLicenseReferences(repoRoot string, policyDoc LicensePolicy) ([]string, error) {
	pathMatchers, err := compileRegexps(flattenLicensePathRegexps(policyDoc.RestrictedComponents))
	if err != nil {
		return nil, err
	}
	contentMatchers, err := compileRegexps(flattenLicenseContentRegexps(policyDoc.RestrictedComponents))
	if err != nil {
		return nil, err
	}
	scanPaths, err := policy.WalkScopeFiles(repoRoot, policy.PathScope{
		Include: policyDoc.ScanScope.Include,
		Exclude: policyDoc.ScanScope.Exclude,
	})
	if err != nil {
		return nil, err
	}
	findings := validationissues.NewIssues(len(scanPaths))
	for _, relPath := range scanPaths {
		if matchesAnyRegexp(relPath, pathMatchers) {
			findings.Add(relPath)
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, relPath))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if matchesAnyRegexp(string(data), contentMatchers) {
			findings.Add(relPath)
		}
	}
	return findings.Sorted(), nil
}

func flattenLicensePathRegexps(components []licenseRestrictedComponent) []string {
	patterns := make([]string, 0)
	for _, component := range components {
		patterns = append(patterns, component.Match.PathRegex...)
	}
	return patterns
}

func flattenLicenseContentRegexps(components []licenseRestrictedComponent) []string {
	patterns := make([]string, 0)
	for _, component := range components {
		patterns = append(patterns, component.Match.ContentRegex...)
	}
	return patterns
}

func compileRegexps(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func matchesAnyRegexp(value string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
