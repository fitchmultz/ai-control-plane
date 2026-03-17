// Package validation owns typed repository validation policies and checks.
//
// Purpose:
//   - Lock down the tracked host-first playbook postcheck contract.
//
// Responsibilities:
//   - Ensure baseline host hardening remains part of the tracked playbook.
//   - Ensure generic host health and smoke checks remain in place.
//   - Ensure supported overlays trigger their expected postchecks.
//
// Scope:
//   - Repository-contract tests against the tracked Ansible playbook.
//
// Usage:
//   - Run with `go test ./internal/validation`.
//
// Invariants/Assumptions:
//   - Host overlay validation stays additive after generic health/smoke.
//   - UI, TLS, and DLP overlays retain explicit postchecks.
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrackedGatewayHostPlaybookRunsOverlaySpecificPostchecks(t *testing.T) {
	repoRoot := repoRootForTrackedComposeContracts(t)
	path := filepath.Join(repoRoot, "deploy", "ansible", "playbooks", "gateway_host.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)

	requiredSnippets := []string{
		"upgrade: \"{{ acp_host_apt_upgrade_mode }}\"",
		"name: \"{{ acp_host_required_packages }}\"",
		"dest: /etc/apt/apt.conf.d/20auto-upgrades",
		"acp_upgrade_mode: false",
		"acp_upgrade_rollback: false",
		"acp_upgrade_from_version: \"\"",
		"acp_upgrade_to_version: \"\"",
		"Upgrade mode requires explicit acp_upgrade_from_version and acp_upgrade_to_version.",
		"path: \"{{ acp_repo_path }}/VERSION\"",
		"acp_tracked_target_version: \"{{ acp_version_file.content | b64decode | trim }}\"",
		"dest: /etc/ssh/sshd_config.d/60-ai-control-plane-hardening.conf",
		"dest: /etc/sysctl.d/60-ai-control-plane-hardening.conf",
		"src: ../../systemd/ai-control-plane-backup.service.tmpl",
		"dest: /etc/systemd/system/ai-control-plane-backup.service",
		"src: ../../systemd/ai-control-plane-backup.timer.tmpl",
		"dest: /etc/systemd/system/ai-control-plane-backup.timer",
		"name: ai-control-plane-backup.timer",
		"enabled: \"{{ acp_backup_timer_enabled | bool }}\"",
		"acp_cert_renewal_timer_enabled is defined",
		"acp_cert_renewal_timer_on_calendar is defined",
		"acp_cert_renewal_timer_randomized_delay_sec is defined",
		"acp_cert_renewal_threshold_days is defined",
		"src: ../../systemd/ai-control-plane-cert-renewal.service.tmpl",
		"dest: /etc/systemd/system/ai-control-plane-cert-renewal.service",
		"src: ../../systemd/ai-control-plane-cert-renewal.timer.tmpl",
		"dest: /etc/systemd/system/ai-control-plane-cert-renewal.timer",
		"name: ai-control-plane-cert-renewal.timer",
		"argv:\n          - ./scripts/acpctl.sh\n          - cert\n          - check",
		"Certificate renewal timer enabled: {{ acp_cert_renewal_timer_enabled }}",
		"Certificate renewal schedule: {{ acp_cert_renewal_timer_on_calendar }}",
		"Certificate renewal threshold days: {{ acp_cert_renewal_threshold_days }}",
		"path: \"{{ acp_repo_path }}/demo/logs/upgrades\"",
		"dest: \"{{ acp_repo_path }}/demo/logs/upgrades/deployed-version.json\"",
		"\"deployed_version\": \"{{ acp_upgrade_to_version if (acp_upgrade_mode | bool) else acp_tracked_target_version }}\"",
		"state: \"{{ 'started' if (acp_backup_timer_enabled | bool) else 'stopped' }}\"",
		"argv:\n          - ufw\n          - --force\n          - reset",
		"argv:\n          - ufw\n          - limit\n          - OpenSSH",
		"argv:\n          - make\n          - health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - prod-smoke\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - librechat-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - tls-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		"argv:\n          - make\n          - dlp-health\n          - COMPOSE_ENV_FILE={{ acp_env_file }}",
		`when: "'ui' in acp_runtime_overlays"`,
		`when: "'tls' in acp_runtime_overlays"`,
		`when: "'dlp' in acp_runtime_overlays"`,
		`acp_public_url must stay loopback-only without the tls overlay`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(content, snippet) {
			t.Fatalf("gateway_host.yml missing required overlay postcheck contract %q", snippet)
		}
	}
}
