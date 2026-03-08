// cmd_validate_test.go - Tests for detection and SIEM validation commands.
//
// Purpose: Verify enterprise-governance config validation stays meaningful.
// Responsibilities:
//   - Test happy-path validation for detection and SIEM contracts.
//   - Test rejection of missing or drifted mappings.
//   - Ensure schema validation catches incomplete normalized mappings.
//
// Non-scope:
//   - Does not exercise live database or SIEM systems.
//   - Does not validate every production rule permutation.
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

func TestRunValidateDetections_Success(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidationFixtureRepo(t, repoRoot, false)

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateDetections(context.Background(), []string{"--verbose"}, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if !strings.Contains(readFile(t, stdout), "Validated 2 detection rule(s)") {
		t.Fatalf("expected success output, got %s", readFile(t, stdout))
	}
}

func TestRunValidateDetections_FailsOnDuplicateRuleID(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidationFixtureRepo(t, repoRoot, true)

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateDetections(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stderr), "duplicate rule_id") {
		t.Fatalf("expected duplicate rule_id error, got %s", readFile(t, stderr))
	}
}

func TestRunValidateSIEMQueries_FailsOnMissingDetectionMapping(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidationFixtureRepo(t, repoRoot, false)
	writeFile(t, filepath.Join(repoRoot, siemQueriesRelativePath), validSIEMQueriesYAML("DR-002"))

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateSiemQueries(context.Background(), nil, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stderr), "missing SIEM mapping for enabled detection rule \"DR-001\"") {
		t.Fatalf("expected missing rule coverage error, got %s", readFile(t, stderr))
	}
}

func TestRunValidateSIEMQueries_SchemaValidation(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidationFixtureRepo(t, repoRoot, false)
	broken := strings.Replace(validSIEMQueriesYAML("DR-001", "DR-002"), "  - normalized: policy.action\n    splunk: action\n    elk: event.action\n    sentinel: Action\n", "", 1)
	writeFile(t, filepath.Join(repoRoot, siemQueriesRelativePath), broken)

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateSiemQueries(context.Background(), []string{"--validate-schema"}, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitDomain {
		t.Fatalf("expected domain failure, got %d", exitCode)
	}
	if !strings.Contains(readFile(t, stderr), "missing siem_config.field_mappings entry for \"policy.action\"") {
		t.Fatalf("expected schema mapping error, got %s", readFile(t, stderr))
	}
}

func TestRunValidateConfigProduction_Success(t *testing.T) {
	repoRoot := t.TempDir()
	writeProductionValidationFixtureRepo(t, repoRoot)
	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	writeFile(t, secretsPath, ""+
		"ACP_DATABASE_MODE=external\n"+
		"CADDY_PUBLISH_HOST=0.0.0.0\n"+
		"CADDYFILE_PATH=./config/caddy/Caddyfile.prod\n"+
		"CADDY_ACME_CA=letsencrypt\n"+
		"CADDY_DOMAIN=gateway.example.com\n"+
		"CADDY_EMAIL=ops@example.com\n"+
		"DATABASE_URL=postgresql://app:verysecurepassword@db.example.com:5432/acp?sslmode=require\n"+
		"LITELLM_MASTER_KEY=prod-master-token-abcdefghijklmnopqrstuvwxyz1234567890\n"+
		"LITELLM_PUBLISH_HOST=127.0.0.1\n"+
		"LITELLM_PUBLIC_URL=https://gateway.example.com\n"+
		"LITELLM_SALT_KEY=prod-salt-token-abcdefghijklmnopqrstuvwxyz1234567890\n"+
		"OTEL_INGEST_AUTH_TOKEN=otel-ingest-auth-token-abcdefghijklmnopqrstuvwxyz\n")
	if err := os.Chmod(secretsPath, 0o600); err != nil {
		t.Fatalf("chmod %s: %v", secretsPath, err)
	}

	stdout, stderr := newTestFiles(t)
	exitCode := withRepoRoot(t, repoRoot, func() int {
		return runValidateConfig(context.Background(), []string{"--production", "--secrets-env-file", secretsPath}, stdout, stderr)
	})

	if exitCode != exitcodes.ACPExitSuccess {
		t.Fatalf("expected success, got %d stderr=%s", exitCode, readFile(t, stderr))
	}
	if got := readFile(t, stdout); !strings.Contains(got, "Profile: production") {
		t.Fatalf("expected production output, got %s", got)
	}
}

func TestRunValidateConfigProduction_MissingSecretsFlagValue(t *testing.T) {
	stdout, stderr := newTestFiles(t)
	exitCode := runValidateConfig(context.Background(), []string{"--production", "--secrets-env-file"}, stdout, stderr)
	if exitCode != exitcodes.ACPExitUsage {
		t.Fatalf("expected usage exit, got %d", exitCode)
	}
	if got := readFile(t, stderr); !strings.Contains(got, "missing value for --secrets-env-file") {
		t.Fatalf("expected missing flag value error, got %s", got)
	}
}

func newTestFiles(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	stdout, err := os.CreateTemp("", "acpctl-validate-stdout")
	if err != nil {
		t.Fatalf("create stdout temp file: %v", err)
	}
	stderr, err := os.CreateTemp("", "acpctl-validate-stderr")
	if err != nil {
		t.Fatalf("create stderr temp file: %v", err)
	}
	t.Cleanup(func() {
		_ = stdout.Close()
		_ = stderr.Close()
		_ = os.Remove(stdout.Name())
		_ = os.Remove(stderr.Name())
	})
	return stdout, stderr
}

func readFile(t *testing.T, file *os.File) string {
	t.Helper()
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek %s: %v", file.Name(), err)
	}
	data, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("read %s: %v", file.Name(), err)
	}
	return string(data)
}

func withRepoRoot(t *testing.T, repoRoot string, fn func() int) int {
	t.Helper()
	original := os.Getenv("ACP_REPO_ROOT")
	if err := os.Setenv("ACP_REPO_ROOT", repoRoot); err != nil {
		t.Fatalf("set ACP_REPO_ROOT: %v", err)
	}
	defer func() {
		if original == "" {
			_ = os.Unsetenv("ACP_REPO_ROOT")
			return
		}
		_ = os.Setenv("ACP_REPO_ROOT", original)
	}()
	return fn()
}

func writeValidationFixtureRepo(t *testing.T, repoRoot string, duplicateRule bool) {
	t.Helper()
	writeFile(t, filepath.Join(repoRoot, litellmConfigRelativePath), validLiteLLMYAML)
	detectionYAML := validDetectionRulesYAML
	if duplicateRule {
		detectionYAML = strings.Replace(detectionYAML, "rule_id: DR-002", "rule_id: DR-001", 1)
	}
	writeFile(t, filepath.Join(repoRoot, detectionRulesRelativePath), detectionYAML)
	writeFile(t, filepath.Join(repoRoot, siemQueriesRelativePath), validSIEMQueriesYAML("DR-001", "DR-002"))
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeProductionValidationFixtureRepo(t *testing.T, repoRoot string) {
	t.Helper()
	writeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), ""+
		"services:\n"+
		"  otel-collector:\n"+
		"    image: otel/opentelemetry-collector-contrib:0.147.0\n"+
		"    ports:\n"+
		"      - \"127.0.0.1:4317:4317\"\n"+
		"      - \"127.0.0.1:4318:4318\"\n"+
		"      - \"127.0.0.1:13133:13133\"\n"+
		"    healthcheck:\n"+
		"      test: [\"CMD\", \"/otelcol-contrib\", \"validate\", \"--config=/etc/otel-collector/config.production.yaml\"]\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services:\n  caddy:\n    image: caddy:2\n    healthcheck:\n      test: [\"CMD\", \"caddy\", \"validate\", \"--config\", \"/etc/caddy/Caddyfile\"]\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.production.yaml"), "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 127.0.0.1:4317\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.yaml"), "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 127.0.0.1:4317\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "config", "caddy", "Caddyfile.prod"), ""+
		"{$CADDY_DOMAIN} {\n"+
		"    handle_path /otel/* {\n"+
		"        @authorized header Authorization \"Bearer {$OTEL_INGEST_AUTH_TOKEN}\"\n"+
		"        reverse_proxy @authorized otel-collector:4318\n"+
		"        respond \"OTEL ingest authorization required\" 401\n"+
		"    }\n"+
		"    reverse_proxy litellm:4000\n"+
		"}\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "Chart.yaml"), "apiVersion: v2\nname: acp\nversion: 0.1.0\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "values.schema.json"), `{"type":"object"}`)
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "values.yaml"), "profile: production\ndemo:\n  enabled: false\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "examples", "values.demo.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "examples", "values.offline.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "helm", "ai-control-plane", "templates", "deployment-litellm.yaml"), "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: litellm\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "hosts: all\ntasks:\n  - debug:\n      msg: ok\n")
	writeFile(t, filepath.Join(repoRoot, "deploy", "terraform", "examples", "aws-complete", "main.tf"), "terraform {\n  required_version = \">= 1.9.0\"\n}\n")
	writeFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM scratch\n")
}

const validLiteLLMYAML = `---
model_list:
  - model_name: openai-gpt5.2
  - model_name: claude-haiku-4-5
`

const validDetectionRulesYAML = `---
detection_rules:
  - rule_id: DR-001
    name: "Non-Approved Model Access"
    description: "Detects requests to models not in the approved list"
    severity: "high"
    category: "policy_violation"
    operational_status: "validated"
    coverage_tier: "decision-grade"
    expected_signal: "Any request model not in approved list within last 24h"
    enabled: true
    auto_response:
      enabled: true
      action: "suspend_key"
      grace_period_minutes: 0
    sql_query: |
      SELECT *
      FROM logs
      WHERE model NOT IN (SELECT jsonb_array_elements_text(:'APPROVED_MODELS_JSON'::jsonb));
    remediation: "Review and revoke unauthorized access"
  - rule_id: DR-002
    name: "Token Usage Spike"
    description: "Detects unusual token consumption patterns per key"
    severity: "medium"
    category: "anomaly"
    operational_status: "example"
    coverage_tier: "demo"
    expected_signal: "Keys exceeding static 24h token threshold"
    enabled: true
    auto_response:
      enabled: false
      action: "alert_only"
      grace_period_minutes: 0
    sql_query: |
      SELECT key_alias, SUM(total_tokens)
      FROM logs
      GROUP BY key_alias;
    remediation: "Investigate unusual usage"
`

func validSIEMQueriesYAML(ruleIDs ...string) string {
	entries := make([]string, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		name := "Token Usage Spike"
		severity := "medium"
		category := "anomaly"
		splunkQuery := "index=ai_gateway"
		elkQuery := "sourcetype:litellm_audit"
		sentinelQuery := "AIGatewayLogs"
		sigmaDetection := "selection:\n          event.category: ai_usage\n        condition: selection"
		if ruleID == "DR-001" {
			name = "Non-Approved Model Access"
			severity = "high"
			category = "policy_violation"
			splunkQuery = "index=ai_gateway | eval approved_models=\"{{APPROVED_MODELS_SPLUNK}}\""
			elkQuery = "model_id:({{APPROVED_MODELS_ELK_OR}})"
			sentinelQuery = "let approved_models = dynamic({{APPROVED_MODELS_JSON}});"
			sigmaDetection = "selection:\n          model_id|contains: |\n            {{APPROVED_MODELS_SIGMA}}\n        condition: selection"
		}
		entry := strings.Join([]string{
			"",
			"  - rule_id: " + ruleID,
			"    name: \"" + name + "\"",
			"    description: \"Example SIEM mapping\"",
			"    severity: " + severity,
			"    category: " + category,
			"    enabled: true",
			"    splunk:",
			"      platform: \"Splunk\"",
			"      query: |",
			"        " + splunkQuery,
			"    elk_kql:",
			"      platform: \"ELK\"",
			"      query: |",
			"        " + elkQuery,
			"    sentinel_kql:",
			"      platform: \"Sentinel\"",
			"      query: |",
			"        " + sentinelQuery,
			"    sigma:",
			"      title: \"" + name + "\"",
			"      status: stable",
			"      description: \"Sigma mapping\"",
			"      detection:",
			"        " + sigmaDetection,
			"      level: " + severity,
			"      tags:",
			"        - attack.discovery",
		}, "\n")
		entries = append(entries, entry)
	}
	return `---
siem_queries:` + strings.Join(entries, "") + `
siem_config:
  field_mappings:
  - normalized: principal.id
    splunk: user_id
    elk: user.id
    sentinel: UserId
  - normalized: ai.model.id
    splunk: model_id
    elk: ai.model.id
    sentinel: ModelId
  - normalized: ai.request.timestamp
    splunk: _time
    elk: "@timestamp"
    sentinel: TimeGenerated
  - normalized: ai.cost.amount
    splunk: spend
    elk: ai.cost.amount
    sentinel: Spend
  - normalized: ai.tokens.total
    splunk: total_tokens
    elk: ai.tokens.total
    sentinel: TotalTokens
  - normalized: policy.action
    splunk: action
    elk: event.action
    sentinel: Action
`
}
