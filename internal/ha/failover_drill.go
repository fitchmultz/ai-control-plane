// failover_drill.go - Active-passive HA failover drill contract validation.
//
// Purpose:
//   - Validate customer-operated active-passive HA failover drill manifests.
//
// Responsibilities:
//   - Load and normalize tracked failover drill contracts.
//   - Enforce truthful evidence requirements for replication, fencing, promotion,
//     traffic cutover, and post-cutover verification.
//   - Provide canonical runbook-step summaries without implying automation.
//
// Scope:
//   - Manual customer-operated failover drill validation and evidence planning only.
//
// Usage:
//   - Used by `acpctl host failover-drill` and readiness evidence workflows.
//
// Invariants/Assumptions:
//   - ACP does not automate failover, fencing, PostgreSQL promotion, or traffic cutover.
//   - Customer-owned DNS, load balancers, and network controls remain explicit.
//   - Evidence files referenced by the contract must already exist before validation.
package ha

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TrafficCutoverMethod captures the customer-owned cutover pattern used for the drill.
type TrafficCutoverMethod string

const (
	TrafficCutoverDNS          TrafficCutoverMethod = "dns"
	TrafficCutoverLoadBalancer TrafficCutoverMethod = "load-balancer"
	TrafficCutoverVIP          TrafficCutoverMethod = "vip"
	TrafficCutoverManual       TrafficCutoverMethod = "manual"
)

func (m TrafficCutoverMethod) Valid() bool {
	switch m {
	case TrafficCutoverDNS, TrafficCutoverLoadBalancer, TrafficCutoverVIP, TrafficCutoverManual:
		return true
	default:
		return false
	}
}

// FailoverStageEvidence captures one required manual step plus its proof artifact.
type FailoverStageEvidence struct {
	Method       string `yaml:"method" json:"method"`
	EvidencePath string `yaml:"evidence_path" json:"evidence_path"`
}

// PromotionEvidence captures the standby promotion proof.
type PromotionEvidence struct {
	PromotedHost string `yaml:"promoted_host" json:"promoted_host"`
	EvidencePath string `yaml:"evidence_path" json:"evidence_path"`
}

// TrafficCutoverEvidence captures customer-operated traffic swing proof.
type TrafficCutoverEvidence struct {
	Method       TrafficCutoverMethod `yaml:"method" json:"method"`
	EvidencePath string               `yaml:"evidence_path" json:"evidence_path"`
}

// FailoverDrillContract captures the operator-supplied HA failover drill inputs.
type FailoverDrillContract struct {
	DrillHost               string                 `yaml:"drill_host,omitempty" json:"drill_host,omitempty"`
	InventoryPath           string                 `yaml:"inventory_path" json:"inventory_path"`
	SecretsEnvFile          string                 `yaml:"secrets_env_file" json:"secrets_env_file"`
	ActiveHost              string                 `yaml:"active_host" json:"active_host"`
	PassiveHost             string                 `yaml:"passive_host" json:"passive_host"`
	ReplicationEvidencePath string                 `yaml:"replication_evidence_path" json:"replication_evidence_path"`
	Fencing                 FailoverStageEvidence  `yaml:"fencing" json:"fencing"`
	Promotion               PromotionEvidence      `yaml:"promotion" json:"promotion"`
	TrafficCutover          TrafficCutoverEvidence `yaml:"traffic_cutover" json:"traffic_cutover"`
	PostcheckEvidencePath   string                 `yaml:"postcheck_evidence_path" json:"postcheck_evidence_path"`
	ExpectedRepoVersion     string                 `yaml:"expected_repo_version,omitempty" json:"expected_repo_version,omitempty"`
	Notes                   string                 `yaml:"notes,omitempty" json:"notes,omitempty"`
}

// LoadFailoverDrillContract reads one YAML failover drill contract from disk.
func LoadFailoverDrillContract(path string) (FailoverDrillContract, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FailoverDrillContract{}, nil, fmt.Errorf("read failover drill contract: %w", err)
	}

	var contract FailoverDrillContract
	if err := yaml.Unmarshal(data, &contract); err != nil {
		return FailoverDrillContract{}, nil, fmt.Errorf("parse failover drill contract: %w", err)
	}
	return normalizeContractFields(contract), data, nil
}

// NormalizeFailoverDrillContract resolves repo-relative tracked paths.
func NormalizeFailoverDrillContract(repoRoot string, contract FailoverDrillContract) FailoverDrillContract {
	contract = normalizeContractFields(contract)
	contract.InventoryPath = normalizePossiblyRelativePath(repoRoot, contract.InventoryPath)
	contract.ReplicationEvidencePath = normalizePossiblyRelativePath(repoRoot, contract.ReplicationEvidencePath)
	contract.Fencing.EvidencePath = normalizePossiblyRelativePath(repoRoot, contract.Fencing.EvidencePath)
	contract.Promotion.EvidencePath = normalizePossiblyRelativePath(repoRoot, contract.Promotion.EvidencePath)
	contract.TrafficCutover.EvidencePath = normalizePossiblyRelativePath(repoRoot, contract.TrafficCutover.EvidencePath)
	contract.PostcheckEvidencePath = normalizePossiblyRelativePath(repoRoot, contract.PostcheckEvidencePath)
	return normalizeContractFields(contract)
}

// ValidateFailoverDrillContract enforces the active-passive failover proof contract.
func ValidateFailoverDrillContract(repoRoot string, contract FailoverDrillContract) error {
	if strings.TrimSpace(repoRoot) == "" {
		return fmt.Errorf("repo root is required")
	}
	if strings.TrimSpace(contract.InventoryPath) == "" {
		return fmt.Errorf("inventory_path is required")
	}
	if strings.TrimSpace(contract.SecretsEnvFile) == "" {
		return fmt.Errorf("secrets_env_file is required")
	}
	if !filepath.IsAbs(contract.SecretsEnvFile) {
		return fmt.Errorf("secrets_env_file must be an absolute path")
	}
	if strings.TrimSpace(contract.ActiveHost) == "" {
		return fmt.Errorf("active_host is required")
	}
	if strings.TrimSpace(contract.PassiveHost) == "" {
		return fmt.Errorf("passive_host is required")
	}
	if contract.ActiveHost == contract.PassiveHost {
		return fmt.Errorf("active_host and passive_host must differ")
	}
	if strings.TrimSpace(contract.ReplicationEvidencePath) == "" {
		return fmt.Errorf("replication_evidence_path is required")
	}
	if strings.TrimSpace(contract.Fencing.Method) == "" {
		return fmt.Errorf("fencing.method is required")
	}
	if strings.TrimSpace(contract.Fencing.EvidencePath) == "" {
		return fmt.Errorf("fencing.evidence_path is required")
	}
	if strings.TrimSpace(contract.Promotion.PromotedHost) == "" {
		return fmt.Errorf("promotion.promoted_host is required")
	}
	if contract.Promotion.PromotedHost != contract.PassiveHost {
		return fmt.Errorf("promotion.promoted_host must match passive_host for the active-passive reference pattern")
	}
	if strings.TrimSpace(contract.Promotion.EvidencePath) == "" {
		return fmt.Errorf("promotion.evidence_path is required")
	}
	if !contract.TrafficCutover.Method.Valid() {
		return fmt.Errorf("traffic_cutover.method must be %q, %q, %q, or %q", TrafficCutoverDNS, TrafficCutoverLoadBalancer, TrafficCutoverVIP, TrafficCutoverManual)
	}
	if strings.TrimSpace(contract.TrafficCutover.EvidencePath) == "" {
		return fmt.Errorf("traffic_cutover.evidence_path is required")
	}
	if strings.TrimSpace(contract.PostcheckEvidencePath) == "" {
		return fmt.Errorf("postcheck_evidence_path is required")
	}

	for label, path := range map[string]string{
		"inventory_path":                contract.InventoryPath,
		"secrets_env_file":              contract.SecretsEnvFile,
		"replication_evidence_path":     contract.ReplicationEvidencePath,
		"fencing.evidence_path":         contract.Fencing.EvidencePath,
		"promotion.evidence_path":       contract.Promotion.EvidencePath,
		"traffic_cutover.evidence_path": contract.TrafficCutover.EvidencePath,
		"postcheck_evidence_path":       contract.PostcheckEvidencePath,
	} {
		if err := requireRegularFile(label, path); err != nil {
			return err
		}
	}

	if contract.ExpectedRepoVersion != "" {
		repoVersion, err := readTrackedRepoVersion(repoRoot)
		if err != nil {
			return err
		}
		if repoVersion != contract.ExpectedRepoVersion {
			return fmt.Errorf("expected_repo_version mismatch: contract=%q repo=%q", contract.ExpectedRepoVersion, repoVersion)
		}
	}

	return nil
}

// EvidenceBoundary returns the truthful claim boundary for the validated drill contract.
func EvidenceBoundary() string {
	return "Manual customer-operated active-passive failover proof only. ACP validates the drill contract and archives evidence for replication readiness, fencing, promotion, traffic cutover, and post-cutover checks. ACP does not automate PostgreSQL replication, promotion, or customer-owned DNS/load-balancer/VIP cutover."
}

// CanonicalRunbookSteps returns the expected operator sequence for the validated drill.
func CanonicalRunbookSteps(contract FailoverDrillContract) []string {
	steps := []string{
		fmt.Sprintf("./scripts/acpctl.sh host check --inventory %s --limit %s", contract.InventoryPath, contract.PassiveHost),
		fmt.Sprintf("Confirm replication readiness using the captured evidence at %s", contract.ReplicationEvidencePath),
		fmt.Sprintf("Fence the failed or about-to-fail primary host using the customer-owned method: %s", contract.Fencing.Method),
		fmt.Sprintf("Promote the passive PostgreSQL node on %s and capture evidence at %s", contract.PassiveHost, contract.Promotion.EvidencePath),
		fmt.Sprintf("./scripts/acpctl.sh host apply --inventory %s --limit %s --skip-smoke-tests", contract.InventoryPath, contract.PassiveHost),
		fmt.Sprintf("Perform customer-owned traffic cutover via %s and capture evidence at %s", contract.TrafficCutover.Method, contract.TrafficCutover.EvidencePath),
		fmt.Sprintf("./scripts/acpctl.sh host apply --inventory %s --limit %s", contract.InventoryPath, contract.PassiveHost),
		fmt.Sprintf("Confirm post-cutover health/smoke evidence at %s", contract.PostcheckEvidencePath),
	}
	return steps
}

func normalizeContractFields(contract FailoverDrillContract) FailoverDrillContract {
	contract.DrillHost = strings.TrimSpace(contract.DrillHost)
	contract.InventoryPath = cleanIfPresent(contract.InventoryPath)
	contract.SecretsEnvFile = cleanIfPresent(contract.SecretsEnvFile)
	contract.ActiveHost = strings.TrimSpace(contract.ActiveHost)
	contract.PassiveHost = strings.TrimSpace(contract.PassiveHost)
	contract.ReplicationEvidencePath = cleanIfPresent(contract.ReplicationEvidencePath)
	contract.Fencing.Method = strings.TrimSpace(contract.Fencing.Method)
	contract.Fencing.EvidencePath = cleanIfPresent(contract.Fencing.EvidencePath)
	contract.Promotion.PromotedHost = strings.TrimSpace(contract.Promotion.PromotedHost)
	if contract.Promotion.PromotedHost == "" {
		contract.Promotion.PromotedHost = contract.PassiveHost
	}
	contract.Promotion.EvidencePath = cleanIfPresent(contract.Promotion.EvidencePath)
	contract.TrafficCutover.Method = TrafficCutoverMethod(strings.TrimSpace(string(contract.TrafficCutover.Method)))
	contract.TrafficCutover.EvidencePath = cleanIfPresent(contract.TrafficCutover.EvidencePath)
	contract.PostcheckEvidencePath = cleanIfPresent(contract.PostcheckEvidencePath)
	contract.ExpectedRepoVersion = strings.TrimSpace(contract.ExpectedRepoVersion)
	contract.Notes = strings.TrimSpace(contract.Notes)
	return contract
}

func cleanIfPresent(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func normalizePossiblyRelativePath(repoRoot string, path string) string {
	if path == "" || filepath.IsAbs(path) || strings.TrimSpace(repoRoot) == "" {
		return path
	}
	return filepath.Join(repoRoot, filepath.Clean(path))
}

func requireRegularFile(label string, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file: %s", label, path)
	}
	return nil
}

func readTrackedRepoVersion(repoRoot string) (string, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("read tracked VERSION: %w", err)
	}
	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("tracked VERSION file is empty")
	}
	return version, nil
}
