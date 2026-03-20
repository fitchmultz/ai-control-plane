// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Validate the tracked design-only tenant package and its cross-file
//     invariants without implying runtime multi-tenant support.
//
// Responsibilities:
//   - Validate the tenant design YAML against the tracked JSON schema.
//   - Cross-reference tenant design models and role bindings against tracked
//     model and RBAC catalogs.
//   - Enforce support-matrix and claim-discipline truth for the incubating
//     multi-tenant design surface.
//
// Scope:
//   - Design-only tenant validation.
//
// Usage:
//   - Used by `acpctl tenant validate` and `acpctl validate tenant`.
//
// Invariants/Assumptions:
//   - Multi-tenant remains incubating and design-only.
//   - Validation must not add runtime enforcement or database behavior.
package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/catalog"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/rbac"
	"github.com/mitchfultz/ai-control-plane/internal/tenant"
	"github.com/xeipuuv/gojsonschema"
)

const (
	tenantDesignSchemaRelativePath = "docs/contracts/config/tenant_design.schema.json"
	tenantADRRelativePath          = "docs/adr/0002-multi-tenant-isolation-design.md"
	tenantPolicyDocRelativePath    = "docs/policy/MULTI_TENANT_ISOLATION_AND_BILLING.md"
	tenantSupportSurfaceID         = "multi-tenant-isolation-design"
)

type TenantValidationOptions struct {
	ConfigPath string
}

// ValidateTenantConfig validates the tracked multi-tenant design package.
func ValidateTenantConfig(repoRoot string, opts TenantValidationOptions) ([]string, error) {
	configPath := repopath.DemoConfigPath(repoRoot, "tenant_design.yaml")
	if trimmed := strings.TrimSpace(opts.ConfigPath); trimmed != "" {
		configPath = repopath.ResolveRepoPath(repoRoot, trimmed)
	}
	displayPath := displayTenantPath(repoRoot, configPath)
	issues := NewIssues()

	schemaPath := repopath.FromRepoRoot(repoRoot, tenantDesignSchemaRelativePath)
	issues.Extend(validateTenantSchema(configPath, schemaPath, displayPath))

	design, err := tenant.LoadFile(configPath)
	if err != nil {
		issues.Addf("%s: %v", displayPath, err)
		return issues.Sorted(), nil
	}

	knownModels, roleModels, roleNames, err := loadTenantValidationContext(repoRoot)
	if err != nil {
		return nil, err
	}
	issues.Extend(tenant.Validate(design, tenant.ValidationOptions{
		SourcePath:  displayPath,
		KnownModels: knownModels,
		KnownRoles:  roleNames,
		RoleModels:  roleModels,
	}))
	issues.Extend(validateTenantDocumentationTruth(repoRoot))
	return issues.Sorted(), nil
}

func loadTenantValidationContext(repoRoot string) (map[string]struct{}, map[string]map[string]struct{}, map[string]struct{}, error) {
	manifest, err := loadConfigContractManifest(repoRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	modelPattern, err := regexp.Compile(manifest.Naming.ModelAliasPattern)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("compile model alias pattern: %w", err)
	}
	knownModels, _, err := loadLiteLLMAliases(repopath.DemoConfigPath(repoRoot, "litellm.yaml"), modelPattern)
	if err != nil {
		return nil, nil, nil, err
	}
	rbacConfig, err := rbac.LoadFile(repopath.DemoConfigPath(repoRoot, "roles.yaml"))
	if err != nil {
		return nil, nil, nil, err
	}
	roleModels := make(map[string]map[string]struct{}, len(rbacConfig.Roles))
	roleNames := make(map[string]struct{}, len(rbacConfig.Roles))
	for roleName, role := range rbacConfig.Roles {
		roleNames[roleName] = struct{}{}
		modelSet := make(map[string]struct{}, len(role.ModelAccess))
		for _, model := range role.ModelAccess {
			trimmed := strings.TrimSpace(model)
			if trimmed != "" {
				modelSet[trimmed] = struct{}{}
			}
		}
		roleModels[roleName] = modelSet
	}
	return knownModels, roleModels, roleNames, nil
}

func validateTenantSchema(documentPath string, schemaPath string, displayPath string) []string {
	jsonBytes, err := yamlFileToJSONBytes(documentPath)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", displayPath, err)}
	}
	schemaURL := url.URL{Scheme: "file", Path: filepath.ToSlash(schemaPath)}
	result, err := gojsonschema.Validate(
		gojsonschema.NewReferenceLoader(schemaURL.String()),
		gojsonschema.NewBytesLoader(jsonBytes),
	)
	if err != nil {
		return []string{fmt.Sprintf("%s: schema validation error: %v", displayPath, err)}
	}
	if result.Valid() {
		return nil
	}
	issues := NewIssues(len(result.Errors()))
	for _, issue := range result.Errors() {
		issues.Addf("%s: %s", displayPath, issue.String())
	}
	return issues.Sorted()
}

func validateTenantDocumentationTruth(repoRoot string) []string {
	issues := NewIssues()
	matrix, err := catalog.LoadSupportMatrix(repopath.FromRepoRoot(repoRoot, "docs", "support-matrix.yaml"))
	if err != nil {
		issues.Addf("docs/support-matrix.yaml: %v", err)
	} else {
		issues.Extend(validateTenantSupportSurface(matrix))
	}
	issues.Extend(requireDocumentContains(repopath.FromRepoRoot(repoRoot, tenantADRRelativePath), tenantADRRelativePath, []string{"design-only", "incubating", "organization"}))
	issues.Extend(requireDocumentContains(repopath.FromRepoRoot(repoRoot, tenantPolicyDocRelativePath), tenantPolicyDocRelativePath, []string{"row-level", "workspace", "provider billing", "design-only"}))
	issues.Extend(requireDocumentContains(repopath.FromRepoRoot(repoRoot, "docs", "GO_TO_MARKET_SCOPE.md"), "docs/GO_TO_MARKET_SCOPE.md", []string{"Multi-tenant managed-service claims"}))
	issues.Extend(requireDocumentContains(repopath.FromRepoRoot(repoRoot, "docs", "KNOWN_LIMITATIONS.md"), "docs/KNOWN_LIMITATIONS.md", []string{"Multi-Tenant Runtime Design-Only"}))
	return issues.Sorted()
}

func validateTenantSupportSurface(matrix catalog.SupportMatrix) []string {
	issues := NewIssues()
	var surface *catalog.SupportSurface
	for i := range matrix.Surfaces {
		if matrix.Surfaces[i].ID == tenantSupportSurfaceID {
			surface = &matrix.Surfaces[i]
			break
		}
	}
	if surface == nil {
		issues.Addf("docs/support-matrix.yaml: missing incubating surface %q", tenantSupportSurfaceID)
		return issues.Sorted()
	}
	if strings.TrimSpace(surface.Status) != "incubating" {
		issues.Addf("docs/support-matrix.yaml: surface %q must remain incubating", tenantSupportSurfaceID)
	}
	for _, requiredPath := range []string{tenant.DefaultDesignPath, tenantPolicyDocRelativePath} {
		if !containsTrimmed(surface.Paths, requiredPath) {
			issues.Addf("docs/support-matrix.yaml: surface %q must include path %q", tenantSupportSurfaceID, requiredPath)
		}
	}
	if !containsTrimmed(surface.Validation, "make validate-tenant") {
		issues.Addf("docs/support-matrix.yaml: surface %q must include validation command %q", tenantSupportSurfaceID, "make validate-tenant")
	}
	return issues.Sorted()
}

func requireDocumentContains(path string, displayPath string, snippets []string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", displayPath, err)}
	}
	content := strings.ToLower(string(data))
	issues := NewIssues()
	for _, snippet := range snippets {
		if !strings.Contains(content, strings.ToLower(snippet)) {
			issues.Addf("%s: missing required design-truth snippet %q", displayPath, snippet)
		}
	}
	return issues.Sorted()
}

func containsTrimmed(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func displayTenantPath(repoRoot string, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return path
	}
	if strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}
