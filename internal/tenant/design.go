// Package tenant defines the tracked design-time multi-tenant contract.
//
// Purpose:
//   - Model organization/workspace isolation, row-level access design,
//     tenant-safe reporting, chargeback boundaries, and service-provider billing
//     boundaries without enabling runtime multi-tenant behavior.
//
// Responsibilities:
//   - Load the tracked tenant design YAML with strict field checking.
//   - Expose typed design primitives and summary helpers for CLI inspection.
//   - Keep design-only tenant concepts out of runtime enforcement packages.
//
// Scope:
//   - Design-time types, loading, and summary helpers only.
//
// Usage:
//   - Used by `acpctl tenant inspect`, `acpctl tenant validate`, and
//     `internal/validation`.
//
// Invariants/Assumptions:
//   - This package is design-only and must not add runtime enforcement.
//   - `design_state` remains `incubating` until a future validated runtime cutover.
package tenant

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

const DefaultDesignPath = "demo/config/tenant_design.yaml"

type DesignState string

const (
	DesignStateIncubating DesignState = "incubating"
)

type EnforcementMode string

const (
	EnforcementModeDesignOnly EnforcementMode = "design-only"
)

type IsolationBoundary string

const (
	IsolationBoundaryOrganization IsolationBoundary = "organization"
	IsolationBoundaryWorkspace    IsolationBoundary = "workspace"
)

type AggregationLevel string

const (
	AggregationLevelWorkspace    AggregationLevel = "workspace"
	AggregationLevelOrganization AggregationLevel = "organization"
)

type RedactionLevel string

const (
	RedactionLevelMetadataOnly RedactionLevel = "metadata-only"
)

type SubjectKind string

const (
	SubjectKindGroup   SubjectKind = "group"
	SubjectKindUser    SubjectKind = "user"
	SubjectKindService SubjectKind = "service"
)

// Design captures the tracked design-only multi-tenant contract.
type Design struct {
	Version         int                   `yaml:"version" json:"version"`
	DesignState     DesignState           `yaml:"design_state" json:"design_state"`
	Organizations   []Organization        `yaml:"organizations" json:"organizations"`
	RowLevelAccess  RowLevelAccessDesign  `yaml:"row_level_access" json:"row_level_access"`
	Reporting       ReportingDesign       `yaml:"reporting" json:"reporting"`
	Chargeback      ChargebackDesign      `yaml:"chargeback" json:"chargeback"`
	ProviderBilling ProviderBillingDesign `yaml:"provider_billing" json:"provider_billing"`
}

// Organization captures one top-level customer tenancy boundary.
type Organization struct {
	ID          string            `yaml:"id" json:"id"`
	DisplayName string            `yaml:"display_name" json:"display_name"`
	Chargeback  ChargebackBinding `yaml:"chargeback" json:"chargeback"`
	Workspaces  []Workspace       `yaml:"workspaces" json:"workspaces"`
}

// Workspace captures one isolated workspace inside an organization boundary.
type Workspace struct {
	ID                 string            `yaml:"id" json:"id"`
	DisplayName        string            `yaml:"display_name" json:"display_name"`
	KeyNamespacePrefix string            `yaml:"key_namespace_prefix" json:"key_namespace_prefix"`
	AllowedModels      []string          `yaml:"allowed_models" json:"allowed_models"`
	RoleBindings       []RoleBinding     `yaml:"role_bindings" json:"role_bindings"`
	Chargeback         ChargebackBinding `yaml:"chargeback" json:"chargeback"`
}

// RoleBinding captures a design-only tenant-scoped RBAC binding.
type RoleBinding struct {
	Role     string            `yaml:"role" json:"role"`
	Scope    IsolationBoundary `yaml:"scope" json:"scope"`
	Subjects []SubjectRef      `yaml:"subjects" json:"subjects"`
}

// SubjectRef captures one user/group/service binding target.
type SubjectRef struct {
	Kind SubjectKind `yaml:"kind" json:"kind"`
	Name string      `yaml:"name" json:"name"`
}

// ChargebackBinding captures allocation ownership metadata.
type ChargebackBinding struct {
	CostCenter string `yaml:"cost_center" json:"cost_center"`
	BillTo     string `yaml:"bill_to" json:"bill_to"`
}

// RowLevelAccessDesign captures row-level enforcement design requirements.
type RowLevelAccessDesign struct {
	Mode               EnforcementMode   `yaml:"mode" json:"mode"`
	RequiredPredicates []string          `yaml:"required_predicates" json:"required_predicates"`
	WriteBoundary      IsolationBoundary `yaml:"write_boundary" json:"write_boundary"`
}

// ReportingDesign captures tenant-safe reporting rules.
type ReportingDesign struct {
	Mode                     EnforcementMode    `yaml:"mode" json:"mode"`
	TenantSafeByDefault      bool               `yaml:"tenant_safe_by_default" json:"tenant_safe_by_default"`
	MetadataOnly             bool               `yaml:"metadata_only" json:"metadata_only"`
	AllowedAggregationLevels []AggregationLevel `yaml:"allowed_aggregation_levels" json:"allowed_aggregation_levels"`
	ReportDefinitions        []ReportDefinition `yaml:"report_definitions" json:"report_definitions"`
}

// ReportDefinition captures a design-time tenant-safe report shape.
type ReportDefinition struct {
	ID                           string           `yaml:"id" json:"id"`
	Aggregation                  AggregationLevel `yaml:"aggregation" json:"aggregation"`
	Redaction                    RedactionLevel   `yaml:"redaction" json:"redaction"`
	IncludeCrossOrganizationData bool             `yaml:"include_cross_organization_data" json:"include_cross_organization_data"`
}

// ChargebackDesign captures internal allocation boundaries inside one tenant.
type ChargebackDesign struct {
	Mode                             EnforcementMode   `yaml:"mode" json:"mode"`
	BillableBoundary                 IsolationBoundary `yaml:"billable_boundary" json:"billable_boundary"`
	AllowCrossOrganizationAllocation bool              `yaml:"allow_cross_organization_allocation" json:"allow_cross_organization_allocation"`
	RequireWorkspaceCostCenter       bool              `yaml:"require_workspace_cost_center" json:"require_workspace_cost_center"`
}

// ProviderBillingDesign captures service-provider billing boundaries between tenants.
type ProviderBillingDesign struct {
	Mode                           EnforcementMode   `yaml:"mode" json:"mode"`
	CustomerInvoiceBoundary        IsolationBoundary `yaml:"customer_invoice_boundary" json:"customer_invoice_boundary"`
	PassthroughUsageCosts          bool              `yaml:"passthrough_usage_costs" json:"passthrough_usage_costs"`
	SeparatePlatformFee            bool              `yaml:"separate_platform_fee" json:"separate_platform_fee"`
	ForbidCrossOrganizationSubsidy bool              `yaml:"forbid_cross_organization_subsidy" json:"forbid_cross_organization_subsidy"`
}

// Summary captures a concise inspectable summary of the tenant design package.
type Summary struct {
	Version                 int      `json:"version"`
	DesignState             string   `json:"design_state"`
	OrganizationCount       int      `json:"organization_count"`
	WorkspaceCount          int      `json:"workspace_count"`
	RoleBindingCount        int      `json:"role_binding_count"`
	ReportDefinitionCount   int      `json:"report_definition_count"`
	ChargebackBoundary      string   `json:"chargeback_boundary"`
	ProviderBillingBoundary string   `json:"provider_billing_boundary"`
	AllowedAggregations     []string `json:"allowed_aggregations"`
	RequiredPredicates      []string `json:"required_predicates"`
	Organizations           []string `json:"organizations"`
	WorkspaceRefs           []string `json:"workspace_refs"`
	KeyNamespaces           []string `json:"key_namespaces"`
}

// LoadFile loads a tenant design YAML file with strict field validation.
func LoadFile(path string) (Design, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Design{}, fmt.Errorf("read %s: %w", path, err)
	}

	var design Design
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&design); err != nil {
		return Design{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return design, nil
}

// WorkspaceRefs returns stable org/workspace references.
func (d Design) WorkspaceRefs() []string {
	refs := make([]string, 0)
	for _, org := range d.Organizations {
		for _, workspace := range org.Workspaces {
			refs = append(refs, org.ID+"/"+workspace.ID)
		}
	}
	sort.Strings(refs)
	return refs
}

// KeyNamespaces returns stable workspace key namespaces.
func (d Design) KeyNamespaces() []string {
	namespaces := make([]string, 0)
	for _, org := range d.Organizations {
		for _, workspace := range org.Workspaces {
			if workspace.KeyNamespacePrefix != "" {
				namespaces = append(namespaces, workspace.KeyNamespacePrefix)
			}
		}
	}
	sort.Strings(namespaces)
	return namespaces
}

// Summary returns a concise inspectable summary for CLI output.
func (d Design) Summary() Summary {
	orgs := make([]string, 0, len(d.Organizations))
	allowedAggregations := make([]string, 0, len(d.Reporting.AllowedAggregationLevels))
	roleBindingCount := 0
	workspaceCount := 0

	for _, level := range d.Reporting.AllowedAggregationLevels {
		allowedAggregations = append(allowedAggregations, string(level))
	}
	sort.Strings(allowedAggregations)

	for _, org := range d.Organizations {
		orgs = append(orgs, org.ID)
		workspaceCount += len(org.Workspaces)
		for _, workspace := range org.Workspaces {
			roleBindingCount += len(workspace.RoleBindings)
		}
	}
	sort.Strings(orgs)

	requiredPredicates := append([]string(nil), d.RowLevelAccess.RequiredPredicates...)
	sort.Strings(requiredPredicates)

	return Summary{
		Version:                 d.Version,
		DesignState:             string(d.DesignState),
		OrganizationCount:       len(d.Organizations),
		WorkspaceCount:          workspaceCount,
		RoleBindingCount:        roleBindingCount,
		ReportDefinitionCount:   len(d.Reporting.ReportDefinitions),
		ChargebackBoundary:      string(d.Chargeback.BillableBoundary),
		ProviderBillingBoundary: string(d.ProviderBilling.CustomerInvoiceBoundary),
		AllowedAggregations:     allowedAggregations,
		RequiredPredicates:      requiredPredicates,
		Organizations:           orgs,
		WorkspaceRefs:           d.WorkspaceRefs(),
		KeyNamespaces:           d.KeyNamespaces(),
	}
}
