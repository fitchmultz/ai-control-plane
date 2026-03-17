// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Verify structural deployment and repository-policy validation behavior.
//
// Responsibilities:
//   - Cover compose healthcheck enforcement.
//   - Cover supported-surface validation boundaries.
//   - Cover header and direct-env policy checks.
//
// Scope:
//   - Unit tests for validation package behavior only.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Tests use temporary fixture repositories.
//   - Validation output remains deterministic for equivalent fixtures.
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/testutil"
)

func TestValidateComposeHealthchecksFlagsMissingTest(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  app:\n    image: example/app:1@sha256:abc\n    healthcheck:\n      interval: 5s\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")

	issues, err := ValidateComposeHealthchecks(repoRoot)
	if err != nil {
		t.Fatalf("ValidateComposeHealthchecks returned error: %v", err)
	}
	if len(issues) == 0 || !strings.Contains(strings.Join(issues, "\n"), `service "app" healthcheck must define test`) {
		t.Fatalf("expected missing test issue, got %v", issues)
	}
}

func TestValidateDeploymentSurfacesFlagsHelmContractDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "Chart.yaml"), "apiVersion: v2\nname: acp\nversion: 0.1.0\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "values.demo.yaml"), "profile: demo\ndemo:\n  enabled: true\n")

	issues, err := ValidateHelmSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ValidateHelmSurfaces returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "values.yaml: profile must be production") {
		t.Fatalf("expected helm profile drift issue, got %v", issues)
	}
}

func TestValidateHelmSurfacesIgnoresNonHelmIssues(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services:\n  app:\n    image: example/app:1\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "Chart.yaml"), "apiVersion: v2\nname: acp\nversion: 0.1.0\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "values.demo.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "values.offline.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "templates", "deployment-litellm.yaml"), "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: litellm\n")

	issues, err := ValidateHelmSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ValidateHelmSurfaces returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "values.yaml: profile must be production") {
		t.Fatalf("expected helm issue, got %v", issues)
	}
	if strings.Contains(joined, "docker-compose") {
		t.Fatalf("expected helm-only validation, got %v", issues)
	}
}

func TestValidateDeploymentSurfacesFlagsNestedCanonicalTargets(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.production.yaml"), "receivers: [\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "tasks: [\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "RUN echo missing-base-image\n")

	issues, err := ValidateDeploymentSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDeploymentSurfaces returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		"demo/config/otel-collector/config.production.yaml: invalid YAML",
		"deploy/ansible/playbooks/gateway_host.yml: invalid YAML",
		"demo/images/litellm-hardened/Dockerfile: Dockerfile must declare at least one FROM instruction",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue %q, got %v", expected, issues)
		}
	}
}

func TestValidateDeploymentSurfacesAllowsTemplateOnlyHelmFiles(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "Chart.yaml"), "apiVersion: v2\nname: acp\nversion: 0.1.0\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "values.yaml"), "profile: production\ndemo:\n  enabled: false\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "values.demo.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "examples", "values.offline.yaml"), "profile: demo\ndemo:\n  enabled: true\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "incubating", "helm", "ai-control-plane", "templates", "validate.yaml"), "{{/* helper template */}}\n{{ include \"acp.validate\" . }}\n")

	issues, err := ValidateHelmSurfaces(repoRoot)
	if err != nil {
		t.Fatalf("ValidateHelmSurfaces returned error: %v", err)
	}
	if joined := strings.Join(issues, "\n"); strings.Contains(joined, "templates/validate.yaml: Helm template must declare apiVersion and kind") {
		t.Fatalf("expected template-only helm file to be allowed, got %v", issues)
	}
}

func TestValidateGoHeadersFlagsMissingSections(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "sample.go"), "package sample\n")

	issues, err := ValidateGoHeaders(repoRoot)
	if err != nil {
		t.Fatalf("ValidateGoHeaders returned error: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "internal/sample.go") {
		t.Fatalf("expected missing header issue, got %v", issues)
	}
}

func TestValidateDirectEnvAccessFlagsForbiddenCalls(t *testing.T) {
	repoRoot := t.TempDir()
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "sample.go"), "package sample\nimport \"os\"\nfunc value() string { return os.Getenv(\"X\") }\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "internal", "config", "allowed.go"), "package config\nimport \"os\"\nfunc value() string { return os.Getenv(\"X\") }\n")

	issues, err := ValidateDirectEnvAccess(repoRoot)
	if err != nil {
		t.Fatalf("ValidateDirectEnvAccess returned error: %v", err)
	}
	if len(issues) != 1 || !strings.Contains(issues[0], "internal/sample.go") {
		t.Fatalf("expected one forbidden env-access issue, got %v", issues)
	}
}

func TestValidateDeploymentConfigProductionPassesCanonicalContract(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidDeploymentSurfaceRepo(t, repoRoot)
	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	writeEnvFixtureFile(t, secretsPath, validProductionSecretsEnv(), 0o600)

	issues, err := ValidateDeploymentConfig(repoRoot, ConfigValidationOptions{
		Profile:        ConfigValidationProfileProduction,
		SecretsEnvFile: secretsPath,
	})
	if err != nil {
		t.Fatalf("ValidateDeploymentConfig returned error: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
}

func TestValidateDeploymentConfigProductionFlagsInsecureHostContract(t *testing.T) {
	repoRoot := t.TempDir()
	writeValidDeploymentSurfaceRepo(t, repoRoot)
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), insecureProductionComposeFixture())
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "caddy", "Caddyfile.prod"), "example.com {\n  reverse_proxy litellm:4000\n}\n")

	secretsPath := filepath.Join(t.TempDir(), "secrets.env")
	writeEnvFixtureFile(t, secretsPath, ""+
		"ACP_DATABASE_MODE=external\n"+
		"CADDY_PUBLISH_HOST=0.0.0.0\n"+
		"CADDYFILE_PATH=./config/caddy/Caddyfile.dev\n"+
		"CADDY_ACME_CA=internal\n"+
		"CADDY_DOMAIN=gateway.example.com\n"+
		"CADDY_EMAIL=ops@example.com\n"+
		"DATABASE_URL=postgresql://app:secret@db.example.com:5432/acp?sslmode=disable\n"+
		"LITELLM_MASTER_KEY=sk-litellm-master-change-me\n"+
		"LITELLM_SALT_KEY=short\n"+
		"LITELLM_PUBLISH_HOST=0.0.0.0\n"+
		"LITELLM_PUBLIC_URL=http://gateway.example.com\n"+
		"OTEL_INGEST_AUTH_TOKEN=replace-with-token\n"+
		"OTEL_PUBLISH_HOST=0.0.0.0\n", 0o644)

	issues, err := ValidateDeploymentConfig(repoRoot, ConfigValidationOptions{
		Profile:        ConfigValidationProfileProduction,
		SecretsEnvFile: secretsPath,
	})
	if err != nil {
		t.Fatalf("ValidateDeploymentConfig returned error: %v", err)
	}
	joined := strings.Join(issues, "\n")
	for _, expected := range []string{
		"secrets file permissions must deny group/other access",
		"must not use OTEL_PUBLISH_HOST",
		`port 4317 must bind 127.0.0.1`,
		"must remain localhost in production",
		"must use CADDYFILE_PATH=./config/caddy/Caddyfile.prod",
		"must use CADDY_ACME_CA=letsencrypt",
		"LITELLM_PUBLIC_URL must be a valid https:// URL",
		"external DATABASE_URL must set sslmode=require or stronger",
		"missing required production OTEL ingress contract",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected issue containing %q, got %v", expected, issues)
		}
	}
}

func writeFixtureFile(t *testing.T, path string, content string) {
	t.Helper()
	testutil.WriteFile(t, filepath.Clean(path), content)
}

func writeEnvFixtureFile(t *testing.T, path string, content string, mode os.FileMode) {
	t.Helper()
	testutil.WriteFileMode(t, filepath.Clean(path), content, mode)
}

func writeValidDeploymentSurfaceRepo(t *testing.T, repoRoot string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.yml"), canonicalProductionComposeFixture())
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.dlp.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.offline.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.tls.yml"), "services:\n  caddy:\n    image: caddy:2\n    healthcheck:\n      test: [\"CMD\", \"caddy\", \"validate\", \"--config\", \"/etc/caddy/Caddyfile\"]\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "docker-compose.ui.yml"), "services: {}\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.production.yaml"), "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 127.0.0.1:4317\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "otel-collector", "config.yaml"), "receivers:\n  otlp:\n    protocols:\n      grpc:\n        endpoint: 127.0.0.1:4317\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "caddy", "Caddyfile.prod"), canonicalProductionCaddyFixture())
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "inventory", "hosts.example.yml"), "all:\n  vars:\n    acp_runtime_overlays: [tls, ui]\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml"), "hosts: all\ntasks:\n  - debug:\n      msg: ok\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "images", "litellm-hardened", "Dockerfile"), "FROM scratch\n")
	writeValidConfigContractRepo(t, repoRoot)
}

func canonicalProductionComposeFixture() string {
	return "" +
		"services:\n" +
		"  otel-collector:\n" +
		"    image: otel/opentelemetry-collector-contrib:0.147.0\n" +
		"    ports:\n" +
		"      - \"127.0.0.1:4317:4317\"\n" +
		"      - \"127.0.0.1:4318:4318\"\n" +
		"      - \"127.0.0.1:13133:13133\"\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD\", \"/otelcol-contrib\", \"validate\", \"--config=/etc/otel-collector/config.production.yaml\"]\n"
}

func insecureProductionComposeFixture() string {
	return "" +
		"services:\n" +
		"  otel-collector:\n" +
		"    image: otel/opentelemetry-collector-contrib:0.147.0\n" +
		"    ports:\n" +
		"      - \"${OTEL_PUBLISH_HOST:-0.0.0.0}:4317:4317\"\n" +
		"      - \"${OTEL_PUBLISH_HOST:-0.0.0.0}:4318:4318\"\n" +
		"      - \"${OTEL_PUBLISH_HOST:-0.0.0.0}:13133:13133\"\n" +
		"    healthcheck:\n" +
		"      test: [\"CMD\", \"/otelcol-contrib\", \"validate\", \"--config=/etc/otel-collector/config.production.yaml\"]\n"
}

func canonicalProductionCaddyFixture() string {
	return "" +
		"{$CADDY_DOMAIN} {\n" +
		"    handle_path /otel/* {\n" +
		"        @authorized header Authorization \"Bearer {$OTEL_INGEST_AUTH_TOKEN}\"\n" +
		"        reverse_proxy @authorized otel-collector:4318\n" +
		"        respond \"OTEL ingest authorization required\" 401\n" +
		"    }\n" +
		"    reverse_proxy litellm:4000\n" +
		"}\n"
}

func validProductionSecretsEnv() string {
	return "" +
		"ACP_DATABASE_MODE=external\n" +
		"CADDY_PUBLISH_HOST=0.0.0.0\n" +
		"CADDYFILE_PATH=./config/caddy/Caddyfile.prod\n" +
		"CADDY_ACME_CA=letsencrypt\n" +
		"CADDY_DOMAIN=gateway.example.com\n" +
		"CADDY_EMAIL=ops@example.com\n" +
		"DATABASE_URL=postgresql://app:verysecurepassword@db.example.com:5432/acp?sslmode=require\n" +
		"LITELLM_MASTER_KEY=prod-master-token-abcdefghijklmnopqrstuvwxyz1234567890\n" +
		"LITELLM_PUBLISH_HOST=127.0.0.1\n" +
		"LITELLM_PUBLIC_URL=https://gateway.example.com\n" +
		"LITELLM_SALT_KEY=prod-salt-token-abcdefghijklmnopqrstuvwxyz1234567890\n" +
		"OTEL_INGEST_AUTH_TOKEN=otel-ingest-auth-token-abcdefghijklmnopqrstuvwxyz\n"
}

func writeValidConfigContractRepo(t *testing.T, repoRoot string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "contract.yaml"), "version: 1\nschemas:\n  - id: litellm\n    path: demo/config/litellm.yaml\n    schema: docs/contracts/config/litellm.schema.json\n  - id: roles\n    path: demo/config/roles.yaml\n    schema: docs/contracts/config/roles.schema.json\n  - id: demo-presets\n    path: demo/config/demo_presets.yaml\n    schema: docs/contracts/config/demo_presets.schema.json\nnaming:\n  model_alias_pattern: '^[a-z0-9]+(?:[-.][a-z0-9]+)*$'\n  role_name_pattern: '^[a-z0-9]+(?:-[a-z0-9]+)*$'\n  preset_name_pattern: '^[a-z0-9]+(?:-[a-z0-9]+)*$'\nruntime:\n  allowed_overlays:\n    - tls\n    - ui\n    - dlp\n    - offline\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "litellm.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "roles.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "docs", "contracts", "config", "demo_presets.schema.json"), `{"type":"object"}`)
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "ansible.cfg"), "[defaults]\nhost_key_checking = True\n[ssh_connection]\nssh_args = -o ControlMaster=auto -o ControlPersist=60s\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "deploy", "ansible", "inventory", "group_vars", "gateway.yml"), "acp_public_url: http://127.0.0.1:4000\nacp_backup_timer_enabled: true\nacp_backup_timer_on_calendar: daily\nacp_backup_timer_randomized_delay_sec: 15m\nacp_backup_retention_keep: 7\nacp_host_required_packages:\n  - ufw\n  - unattended-upgrades\nacp_host_firewall_default_incoming_policy: deny\nacp_host_firewall_default_outgoing_policy: allow\nacp_host_firewall_default_routed_policy: deny\nacp_host_minimum_debian_version: \"12\"\nacp_host_minimum_ubuntu_version: \"24.04\"\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "litellm.yaml"), "model_list:\n  - model_name: openai-gpt5.2\n    litellm_params:\n      model: openai/gpt-5.2\n      api_key: os.environ/OPENAI_API_KEY\ngeneral_settings:\n  database_url: os.environ/DATABASE_URL\nlitellm_settings:\n  master_key: os.environ/LITELLM_MASTER_KEY\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "roles.yaml"), "roles:\n  developer:\n    description: Developer\n    model_access: [openai-gpt5.2]\n    budget_ceiling: 25\n    can_approve: false\n    can_assign_roles: false\n    can_create_keys: true\n    read_only: false\n    approval_authority: null\ndefault_role: developer\nmodel_tiers:\n  standard: [openai-gpt5.2]\n")
	writeFixtureFile(t, filepath.Join(repoRoot, "demo", "config", "demo_presets.yaml"), "presets:\n  demo-default:\n    name: Demo Default\n    description: Default demo preset\n    timeout_minutes: 5\n    scenarios: [1]\n    stop_on_fail: true\n    intro_message: hello\nsettings:\n  default_timeout_minutes: 5\n  scenario_delay_seconds: 0\n  colors_enabled: true\n")
}
