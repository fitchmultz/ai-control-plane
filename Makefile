# AI Control Plane - Main Makefile
#
# Purpose: Orchestrate all build, test, and deployment operations
# Responsibilities:
#   - Include focused sub-makefiles for each domain
#   - Provide main help target
#   - Ensure consistent variable definitions
#
# Non-scope:
#   - Does not define targets directly (see mk/*.mk files)
#   - Does not override sub-makefile behavior

# =============================================================================
# Include Sub-Makefiles (in dependency order)
# =============================================================================

include mk/variables.mk
include mk/colors.mk
include mk/install.mk
include mk/lint.mk
include mk/deploy.mk
include mk/production.mk
include mk/offline.mk
include mk/database.mk
include mk/host.mk
include mk/upgrade.mk
include mk/demo.mk
include mk/security.mk
include mk/release.mk
include mk/test.mk
include mk/validation.mk
include mk/key.mk
include mk/onboard.mk
include mk/ci.mk

# =============================================================================
# Help Target
# =============================================================================

.PHONY: help
help: ## Show this help message
	@echo '$(COLOR_BOLD)AI Control Plane - Available Targets$(COLOR_RESET)'
	@echo ''
	@echo 'Usage: make [target] [VARIABLE=value ...]'
	@echo ''
	@echo '$(COLOR_BOLD)Essential Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)install$(COLOR_RESET)            Set up dependencies'
	@echo '  $(COLOR_GREEN)up$(COLOR_RESET)                 Start services'
	@echo '  $(COLOR_GREEN)up-dlp$(COLOR_RESET)             Start the optional DLP overlay'
	@echo '  $(COLOR_GREEN)up-ui$(COLOR_RESET)              Start the optional managed UI overlay'
	@echo '  $(COLOR_GREEN)up-full$(COLOR_RESET)            Start both optional overlays'
	@echo '  $(COLOR_GREEN)health$(COLOR_RESET)             Check service health'
	@echo '  $(COLOR_GREEN)down$(COLOR_RESET)               Stop services'
	@echo '  $(COLOR_GREEN)ci$(COLOR_RESET)                 Run local full CI gate (runtime via pinned offline images)'
	@echo '  $(COLOR_GREEN)ci-pr$(COLOR_RESET)              Run PR-required fast checks'
	@echo '  $(COLOR_GREEN)ci-heavy$(COLOR_RESET)           Run heavyweight local checks'
	@echo '  $(COLOR_GREEN)readiness-evidence$(COLOR_RESET) Generate current readiness evidence'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle$(COLOR_RESET) Build a local pilot closeout bundle'
	@echo '  $(COLOR_GREEN)help$(COLOR_RESET)               Show this help'
	@echo '  $(COLOR_GREEN)onboard$(COLOR_RESET)            Onboard a CLI/IDE tool'
	@echo '  $(COLOR_GREEN)chatgpt-login$(COLOR_RESET)      Trigger ChatGPT OAuth device flow'
	@echo ''
	@echo '$(COLOR_BOLD)Category Help:$(COLOR_RESET)'
	@echo '  make help-install    Installation targets'
	@echo '  make help-deploy     Deployment targets'
	@echo '  make help-lint       Lint and validation targets'
	@echo '  make help-security   Security and supply chain'
	@echo '  make help-test       Testing targets'
	@echo '  make help-ci         CI tier targets'
	@echo '  make help-db         Database operations'
	@echo '  make help-host       Host deployment targets'
	@echo '  make help-upgrade    Upgrade and rollback targets'
	@echo '  make help-key        Virtual key operations'
	@echo '  make help-onboard    Tool onboarding targets'
	@echo ''
	@echo '$(COLOR_BOLD)Documentation:$(COLOR_RESET)'
	@echo '  See AGENTS.md for project guidelines'
	@echo '  See README.md for detailed setup instructions'

.PHONY: help-install
help-install:
	@echo '$(COLOR_BOLD)Installation Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)install$(COLOR_RESET)            Set up dependencies'
	@echo '  $(COLOR_GREEN)install-env$(COLOR_RESET)        Create demo/.env from example'
	@echo '  $(COLOR_GREEN)install-binary$(COLOR_RESET)     Build acpctl binary'
	@echo '  $(COLOR_GREEN)install-ci$(COLOR_RESET)         CI-only setup (no image pull)'
	@echo '  $(COLOR_GREEN)completions$(COLOR_RESET)        Generate shell completions'
	@echo '  $(COLOR_GREEN)update$(COLOR_RESET)             Refresh generated files, pull base images, rebuild local hardened images'

.PHONY: help-deploy
help-deploy:
	@echo '$(COLOR_BOLD)Deployment Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)up$(COLOR_RESET)                 Start services'
	@echo '  $(COLOR_GREEN)up-dlp$(COLOR_RESET)             Start the optional DLP overlay'
	@echo '  $(COLOR_GREEN)up-ui$(COLOR_RESET)              Start the optional managed UI overlay'
	@echo '  $(COLOR_GREEN)up-full$(COLOR_RESET)            Start the optional DLP + managed UI overlays'
	@echo '  $(COLOR_GREEN)down$(COLOR_RESET)               Stop services'
	@echo '  $(COLOR_GREEN)restart$(COLOR_RESET)            Restart services'
	@echo '  $(COLOR_GREEN)ps$(COLOR_RESET)                 Show running services'
	@echo '  $(COLOR_GREEN)logs$(COLOR_RESET)               Tail service logs'
	@echo '  $(COLOR_GREEN)status$(COLOR_RESET)             Show system status'
	@echo '  $(COLOR_GREEN)health$(COLOR_RESET)             Run health checks'
	@echo '  $(COLOR_GREEN)operator-report$(COLOR_RESET)    Generate canonical operator report'
	@echo '  $(COLOR_GREEN)operator-dashboard$(COLOR_RESET) Generate static HTML operator dashboard snapshot'
	@echo '  $(COLOR_GREEN)librechat-health$(COLOR_RESET)   Check managed LibreChat health'
	@echo '  $(COLOR_GREEN)doctor$(COLOR_RESET)             Run diagnostics'
	@echo '  $(COLOR_GREEN)readiness-evidence$(COLOR_RESET) Generate readiness proof pack'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle$(COLOR_RESET) Build pilot closeout artifact set'
	@echo ''
	@echo 'Production/Offline:'
	@echo '  $(COLOR_GREEN)up-production$(COLOR_RESET)      Start production profile'
	@echo '  $(COLOR_GREEN)up-tls$(COLOR_RESET)             Start TLS mode'
	@echo '  $(COLOR_GREEN)up-offline$(COLOR_RESET)         Start offline mode with local hardened images'
	@echo '  $(COLOR_GREEN)up-offline-ci$(COLOR_RESET)      Start offline mode for CI using pinned fallback images'
	@echo '  $(COLOR_GREEN)cert-status$(COLOR_RESET)        Check certificate lifecycle status'
	@echo '  $(COLOR_GREEN)cert-renew$(COLOR_RESET)         Trigger certificate renewal'

.PHONY: help-lint
help-lint:
	@echo '$(COLOR_BOLD)Lint and Validation Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)lint$(COLOR_RESET)               Run all linters'
	@echo '  $(COLOR_GREEN)lint-shell$(COLOR_RESET)         Run shellcheck'
	@echo '  $(COLOR_GREEN)lint-yaml$(COLOR_RESET)          Run yamllint'
	@echo '  $(COLOR_GREEN)lint-compose$(COLOR_RESET)       Validate Docker Compose'
	@echo '  $(COLOR_GREEN)lint-secrets$(COLOR_RESET)       Run secrets audit'
	@echo '  $(COLOR_GREEN)format$(COLOR_RESET)             Format shell scripts'
	@echo '  $(COLOR_GREEN)type-check$(COLOR_RESET)         Run Go type checks'

.PHONY: help-security
help-security:
	@echo '$(COLOR_BOLD)Security Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)public-hygiene-check$(COLOR_RESET) Fail if local-only secrets/artifacts are tracked'
	@echo '  $(COLOR_GREEN)secrets-audit$(COLOR_RESET)      Run secrets audit'
	@echo '  $(COLOR_GREEN)license-check$(COLOR_RESET)      Check license boundaries'
	@echo '  $(COLOR_GREEN)supply-chain-gate$(COLOR_RESET)  Run supply chain gate'
	@echo '  $(COLOR_GREEN)security-gate$(COLOR_RESET)      Run aggregate security gate'
	@echo '  $(COLOR_GREEN)hardened-images-build$(COLOR_RESET) Build hardened images'
	@echo '  $(COLOR_GREEN)hardened-images-scan$(COLOR_RESET) Scan hardened images (Trivy)'

.PHONY: help-test
help-test:
	@echo '$(COLOR_BOLD)Testing Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)test$(COLOR_RESET)               Run all tests'
	@echo '  $(COLOR_GREEN)test-go$(COLOR_RESET)            Run Go unit tests'
	@echo '  $(COLOR_GREEN)coverage-critical$(COLOR_RESET)  Enforce minimum coverage for critical high-risk Go packages'
	@echo '  $(COLOR_GREEN)coverage-report$(COLOR_RESET)    Print detailed coverage report for critical high-risk Go packages'
	@echo '  $(COLOR_GREEN)test-health$(COLOR_RESET)        Run health check tests'
	@echo '  $(COLOR_GREEN)script-tests$(COLOR_RESET)       Run shell script tests'
	@echo '  $(COLOR_GREEN)performance-baseline$(COLOR_RESET) Run local gateway performance baseline'
	@echo '                                          Example: make performance-baseline PERFORMANCE_PROFILE=interactive'
	@echo '  $(COLOR_GREEN)validate-detections$(COLOR_RESET) Run detection tests'

.PHONY: help-ci
help-ci:
	@echo '$(COLOR_BOLD)CI Tier Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)ci-pr$(COLOR_RESET)              PR-required fast deterministic checks'
	@echo '  $(COLOR_GREEN)ci$(COLOR_RESET)                 Local full gate (runtime via pinned offline images)'
	@echo '  $(COLOR_GREEN)ci-heavy$(COLOR_RESET)           Heavy local validation (runtime + hardened image checks)'
	@echo '  $(COLOR_GREEN)coverage-critical$(COLOR_RESET)  High-risk package coverage gate enforced by ci-pr and ci-fast'
	@echo '  $(COLOR_GREEN)readiness-evidence$(COLOR_RESET) Generate dated readiness proof package'
	@echo '  $(COLOR_GREEN)readiness-evidence-verify$(COLOR_RESET) Verify latest readiness proof package'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle$(COLOR_RESET) Build dated pilot closeout artifact package'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle-verify$(COLOR_RESET) Verify latest pilot closeout artifact package'
	@echo ''
	@echo 'Recommended usage:'
	@echo '  make ci-pr            # default for pull requests'
	@echo '  make ci               # local full validation via pinned offline images'
	@echo '  make ci-heavy         # optional local hardened image build/scan lane'
	@echo '  make readiness-evidence # regenerate current enterprise proof'
	@echo '  make pilot-closeout-bundle # assemble the pilot closeout packet'

.PHONY: help-db
help-db:
	@echo '$(COLOR_BOLD)Database Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)db-status$(COLOR_RESET)          Show database status'
	@echo '  $(COLOR_GREEN)chargeback-report$(COLOR_RESET)  Generate chargeback/showback report artifacts'
	@echo '  $(COLOR_GREEN)operator-report$(COLOR_RESET)    Generate typed operator runtime report'
	@echo '  $(COLOR_GREEN)operator-dashboard$(COLOR_RESET) Generate static HTML operator dashboard snapshot'
	@echo '  $(COLOR_GREEN)db-backup$(COLOR_RESET)          Create backup'
	@echo '  $(COLOR_GREEN)db-backup-retention$(COLOR_RESET) Check or apply backup retention cleanup'
	@echo '  $(COLOR_GREEN)db-restore$(COLOR_RESET)         Restore from backup'
	@echo '  $(COLOR_GREEN)db-shell$(COLOR_RESET)           Open database shell'
	@echo '  $(COLOR_GREEN)dr-drill$(COLOR_RESET)           Run automated restore verification'

.PHONY: help-host
help-host:
	@echo '$(COLOR_BOLD)Host Deployment Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)host-preflight$(COLOR_RESET)     Run preflight checks'
	@echo '  $(COLOR_GREEN)host-check$(COLOR_RESET)         Check mode'
	@echo '  $(COLOR_GREEN)host-apply$(COLOR_RESET)         Apply mode'
	@echo '  $(COLOR_GREEN)ha-failover-drill$(COLOR_RESET)  Archive a customer-operated active-passive failover drill'
	@echo '  $(COLOR_GREEN)host-install$(COLOR_RESET)       Install systemd service and backup timer'
	@echo '  $(COLOR_GREEN)host-uninstall$(COLOR_RESET)     Remove systemd service and installed timers'
	@echo '  $(COLOR_GREEN)host-service-status$(COLOR_RESET) Show service and installed timer status'
	@echo '  $(COLOR_GREEN)host-service-start$(COLOR_RESET) Start service'
	@echo '  $(COLOR_GREEN)host-service-stop$(COLOR_RESET)  Stop service'
	@echo '  $(COLOR_GREEN)host-service-restart$(COLOR_RESET) Restart service'
	@echo '  $(COLOR_GREEN)cert-status$(COLOR_RESET)        Check certificate lifecycle status'
	@echo '  $(COLOR_GREEN)cert-renew$(COLOR_RESET)         Trigger certificate renewal'
	@echo '  $(COLOR_GREEN)cert-renew-install$(COLOR_RESET) Install the certificate renewal timer'

.PHONY: help-upgrade
help-upgrade:
	@echo '$(COLOR_BOLD)Upgrade Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)upgrade-plan$(COLOR_RESET)      Show the explicit upgrade path for FROM_VERSION -> current VERSION'
	@echo '  $(COLOR_GREEN)upgrade-check$(COLOR_RESET)     Validate upgrade config, DB, and host convergence'
	@echo '  $(COLOR_GREEN)upgrade-execute$(COLOR_RESET)   Execute a supported in-place upgrade with rollback artifacts'
	@echo '  $(COLOR_GREEN)upgrade-rollback$(COLOR_RESET)  Restore rollback artifacts from a recorded upgrade run'
	@echo ''
	@echo 'Examples:'
	@echo '  make upgrade-plan FROM_VERSION=0.1.0'
	@echo '  make upgrade-check FROM_VERSION=0.1.0 INVENTORY=deploy/ansible/inventory/hosts.yml'
	@echo '  make upgrade-rollback UPGRADE_RUN_DIR=demo/logs/upgrades/upgrade-<stamp>'

.PHONY: help-key
help-key:
	@echo '$(COLOR_BOLD)Virtual Key Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)key-gen$(COLOR_RESET)            Generate key'
	@echo '  $(COLOR_GREEN)key-list$(COLOR_RESET)           List keys'
	@echo '  $(COLOR_GREEN)key-inspect$(COLOR_RESET)        Inspect a key and its usage'
	@echo '  $(COLOR_GREEN)key-rotate$(COLOR_RESET)         Stage or execute key rotation'
	@echo '  $(COLOR_GREEN)key-revoke$(COLOR_RESET)         Revoke key'
	@echo '  $(COLOR_GREEN)key-gen-dev$(COLOR_RESET)        Generate developer key'
	@echo '  $(COLOR_GREEN)key-gen-lead$(COLOR_RESET)       Generate team-lead key'
	@echo ''
	@echo 'Examples:'
	@echo '  make key-gen ALIAS=alice BUDGET=10.00'
	@echo '  make key-inspect ALIAS=alice REPORT_MONTH=2026-02'
	@echo '  make key-rotate ALIAS=alice DRY_RUN=1'
	@echo '  make key-revoke ALIAS=alice'

.PHONY: help-onboard
help-onboard:
	@echo '$(COLOR_BOLD)Tool Onboarding Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)onboard$(COLOR_RESET)            Launch the guided onboarding wizard'
	@echo '  $(COLOR_GREEN)onboard-help$(COLOR_RESET)       Show onboarding wizard help'
	@echo '  $(COLOR_GREEN)onboard-codex$(COLOR_RESET)      Launch onboarding with Codex preselected'
	@echo '  $(COLOR_GREEN)onboard-claude$(COLOR_RESET)     Launch onboarding with Claude Code preselected'
	@echo '  $(COLOR_GREEN)onboard-opencode$(COLOR_RESET)   Launch onboarding with OpenCode preselected'
	@echo '  $(COLOR_GREEN)onboard-cursor$(COLOR_RESET)     Launch onboarding with Cursor preselected'
	@echo '  $(COLOR_GREEN)chatgpt-login$(COLOR_RESET)      Trigger ChatGPT OAuth device flow for gateway'
	@echo '  $(COLOR_GREEN)chatgpt-auth-copy$(COLOR_RESET)  Copy local Codex auth cache into LiteLLM container'
	@echo ''
	@echo 'Examples:'
	@echo '  make onboard'
	@echo '  make onboard-codex'
	@echo '  make chatgpt-login'

# Default target
.DEFAULT_GOAL := help
