// failover_drill_test.go - Tests for active-passive HA failover drill contracts.
//
// Purpose:
//   - Verify normalization, validation, and runbook rendering for HA drill manifests.
//
// Responsibilities:
//   - Cover required-field validation and repo-version enforcement.
//   - Confirm repo-relative evidence paths normalize deterministically.
//   - Keep the truthful active-passive promotion contract enforced.
//
// Scope:
//   - Unit tests for internal/ha contract handling only.
//
// Usage:
//   - Run via `go test ./internal/ha` or the repository CI gates.
//
// Invariants/Assumptions:
//   - Tests use temporary files only.
//   - The passive host remains the only valid promotion target.
package ha

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeFailoverDrillContractResolvesRepoRelativePaths(t *testing.T) {
	repoRoot := t.TempDir()
	contract := NormalizeFailoverDrillContract(repoRoot, FailoverDrillContract{
		InventoryPath:           "deploy/ansible/inventory/hosts.ha.example.yml",
		SecretsEnvFile:          "/etc/ai-control-plane/secrets.env",
		ActiveHost:              "acp-active-1",
		PassiveHost:             "acp-passive-1",
		ReplicationEvidencePath: "demo/logs/ha/replication.txt",
		Fencing: FailoverStageEvidence{
			Method:       "power-off",
			EvidencePath: "demo/logs/ha/fencing.txt",
		},
		Promotion: PromotionEvidence{
			EvidencePath: "demo/logs/ha/promotion.txt",
		},
		TrafficCutover: TrafficCutoverEvidence{
			Method:       TrafficCutoverDNS,
			EvidencePath: "demo/logs/ha/cutover.txt",
		},
		PostcheckEvidencePath: "demo/logs/ha/postcheck.txt",
	})

	if contract.InventoryPath != filepath.Join(repoRoot, "deploy/ansible/inventory/hosts.ha.example.yml") {
		t.Fatalf("InventoryPath = %q", contract.InventoryPath)
	}
	if contract.Promotion.PromotedHost != "acp-passive-1" {
		t.Fatalf("PromotedHost default = %q", contract.Promotion.PromotedHost)
	}
	if contract.ReplicationEvidencePath != filepath.Join(repoRoot, "demo/logs/ha/replication.txt") {
		t.Fatalf("ReplicationEvidencePath = %q", contract.ReplicationEvidencePath)
	}
}

func TestValidateFailoverDrillContractAcceptsValidContract(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "v9.9.9\n")
	inventory := writeFile(t, filepath.Join(repoRoot, "deploy/ansible/inventory/hosts.ha.example.yml"), "all: {}\n")
	replication := writeFile(t, filepath.Join(repoRoot, "demo/logs/ha/replication.txt"), "streaming\n")
	fencing := writeFile(t, filepath.Join(repoRoot, "demo/logs/ha/fencing.txt"), "fenced\n")
	promotion := writeFile(t, filepath.Join(repoRoot, "demo/logs/ha/promotion.txt"), "promoted\n")
	cutover := writeFile(t, filepath.Join(repoRoot, "demo/logs/ha/cutover.txt"), "cutover complete\n")
	postcheck := writeFile(t, filepath.Join(repoRoot, "demo/logs/ha/postcheck.txt"), "health ok\n")
	secrets := writeFile(t, filepath.Join(repoRoot, "secrets.env"), "LITELLM_MASTER_KEY=secret\n")

	contract := NormalizeFailoverDrillContract(repoRoot, FailoverDrillContract{
		DrillHost:               "replacement-vm-1",
		InventoryPath:           inventory,
		SecretsEnvFile:          secrets,
		ActiveHost:              "acp-active-1",
		PassiveHost:             "acp-passive-1",
		ReplicationEvidencePath: replication,
		Fencing:                 FailoverStageEvidence{Method: "ipmi-power-off", EvidencePath: fencing},
		Promotion:               PromotionEvidence{PromotedHost: "acp-passive-1", EvidencePath: promotion},
		TrafficCutover:          TrafficCutoverEvidence{Method: TrafficCutoverVIP, EvidencePath: cutover},
		PostcheckEvidencePath:   postcheck,
		ExpectedRepoVersion:     "v9.9.9",
	})

	if err := ValidateFailoverDrillContract(repoRoot, contract); err != nil {
		t.Fatalf("ValidateFailoverDrillContract() error = %v", err)
	}
}

func TestValidateFailoverDrillContractRejectsPromotionTargetMismatch(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "VERSION"), "v1.0.0\n")
	inventory := writeFile(t, filepath.Join(repoRoot, "inventory.yml"), "all: {}\n")
	replication := writeFile(t, filepath.Join(repoRoot, "replication.txt"), "streaming\n")
	fencing := writeFile(t, filepath.Join(repoRoot, "fencing.txt"), "fenced\n")
	promotion := writeFile(t, filepath.Join(repoRoot, "promotion.txt"), "promoted\n")
	cutover := writeFile(t, filepath.Join(repoRoot, "cutover.txt"), "cutover complete\n")
	postcheck := writeFile(t, filepath.Join(repoRoot, "postcheck.txt"), "health ok\n")
	secrets := writeFile(t, filepath.Join(repoRoot, "secrets.env"), "LITELLM_MASTER_KEY=secret\n")

	contract := FailoverDrillContract{
		InventoryPath:           inventory,
		SecretsEnvFile:          secrets,
		ActiveHost:              "acp-active-1",
		PassiveHost:             "acp-passive-1",
		ReplicationEvidencePath: replication,
		Fencing:                 FailoverStageEvidence{Method: "power-off", EvidencePath: fencing},
		Promotion:               PromotionEvidence{PromotedHost: "unexpected-host", EvidencePath: promotion},
		TrafficCutover:          TrafficCutoverEvidence{Method: TrafficCutoverDNS, EvidencePath: cutover},
		PostcheckEvidencePath:   postcheck,
	}

	err := ValidateFailoverDrillContract(repoRoot, contract)
	if err == nil || !strings.Contains(err.Error(), "promotion.promoted_host must match passive_host") {
		t.Fatalf("expected promotion target mismatch, got %v", err)
	}
}

func TestCanonicalRunbookStepsPreserveManualBoundaries(t *testing.T) {
	steps := CanonicalRunbookSteps(FailoverDrillContract{
		InventoryPath:           "/repo/deploy/ansible/inventory/hosts.ha.example.yml",
		ActiveHost:              "acp-active-1",
		PassiveHost:             "acp-passive-1",
		ReplicationEvidencePath: "/tmp/replication.txt",
		Fencing:                 FailoverStageEvidence{Method: "ipmi-power-off", EvidencePath: "/tmp/fencing.txt"},
		Promotion:               PromotionEvidence{PromotedHost: "acp-passive-1", EvidencePath: "/tmp/promotion.txt"},
		TrafficCutover:          TrafficCutoverEvidence{Method: TrafficCutoverLoadBalancer, EvidencePath: "/tmp/cutover.txt"},
		PostcheckEvidencePath:   "/tmp/postcheck.txt",
	})

	joined := strings.Join(steps, "\n")
	if !strings.Contains(joined, "customer-owned traffic cutover") {
		t.Fatalf("runbook steps should keep customer-owned cutover explicit: %s", joined)
	}
	if !strings.Contains(EvidenceBoundary(), "does not automate") {
		t.Fatalf("evidence boundary should keep automation limits explicit")
	}
}

func writeFile(t *testing.T, path string, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	return path
}
