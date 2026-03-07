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
include mk/helm.mk
include mk/host.mk
include mk/demo.mk
include mk/terraform.mk
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
	@echo '  $(COLOR_GREEN)up-core$(COLOR_RESET)            Start LiteLLM core services only'
	@echo '  $(COLOR_GREEN)librechat-up$(COLOR_RESET)       Start managed LibreChat services'
	@echo '  $(COLOR_GREEN)health$(COLOR_RESET)             Check service health'
	@echo '  $(COLOR_GREEN)down$(COLOR_RESET)               Stop services'
	@echo '  $(COLOR_GREEN)ci$(COLOR_RESET)                 Run local full CI gate (runtime via pinned offline images)'
	@echo '  $(COLOR_GREEN)ci-pr$(COLOR_RESET)              Run PR-required fast checks'
	@echo '  $(COLOR_GREEN)ci-nightly$(COLOR_RESET)         Run nightly runtime + release checks (pinned offline images)'
	@echo '  $(COLOR_GREEN)ci-manual-heavy$(COLOR_RESET)    Run heavyweight manual checks (local hardened image build/scan)'
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
	@echo '  make help-demo       Demo scenario targets'
	@echo '  make help-security   Security and supply chain'
	@echo '  make help-test       Testing targets'
	@echo '  make help-ci         CI tier targets'
	@echo '  make help-db         Database operations'
	@echo '  make help-host       Host deployment targets'
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
	@echo '  $(COLOR_GREEN)up-core$(COLOR_RESET)            Start LiteLLM core services only'
	@echo '  $(COLOR_GREEN)librechat-up$(COLOR_RESET)       Start managed LibreChat services'
	@echo '  $(COLOR_GREEN)down$(COLOR_RESET)               Stop services'
	@echo '  $(COLOR_GREEN)restart$(COLOR_RESET)            Restart services'
	@echo '  $(COLOR_GREEN)ps$(COLOR_RESET)                 Show running services'
	@echo '  $(COLOR_GREEN)logs$(COLOR_RESET)               Tail service logs'
	@echo '  $(COLOR_GREEN)status$(COLOR_RESET)             Show system status'
	@echo '  $(COLOR_GREEN)health$(COLOR_RESET)             Run health checks'
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

.PHONY: help-demo
help-demo:
	@echo '$(COLOR_BOLD)Demo Scenario Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)demo-scenario$(COLOR_RESET)      Run specific scenario'
	@echo '  $(COLOR_GREEN)demo-all$(COLOR_RESET)           Run all scenarios'
	@echo '  $(COLOR_GREEN)demo-preset$(COLOR_RESET)        Run named preset'
	@echo '  $(COLOR_GREEN)demo-preset-list$(COLOR_RESET)   List available presets'
	@echo '  $(COLOR_GREEN)demo-snapshot$(COLOR_RESET)      Create snapshot'
	@echo '  $(COLOR_GREEN)demo-restore$(COLOR_RESET)       Restore snapshot'
	@echo '  $(COLOR_GREEN)demo-reset$(COLOR_RESET)         Reset to baseline'
	@echo ''
	@echo 'Examples:'
	@echo '  make demo-scenario SCENARIO=1'
	@echo '  make demo-preset PRESET=executive-demo'

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
	@echo '  $(COLOR_GREEN)ci-nightly$(COLOR_RESET)         Scheduled runtime + release checks (pinned offline images)'
	@echo '  $(COLOR_GREEN)ci-runtime-checks$(COLOR_RESET)  Runtime checks for running stack'
	@echo '  $(COLOR_GREEN)ci-manual-heavy$(COLOR_RESET)    On-demand heavyweight security/image checks'
	@echo '  $(COLOR_GREEN)readiness-evidence$(COLOR_RESET) Generate dated readiness proof package'
	@echo '  $(COLOR_GREEN)readiness-evidence-verify$(COLOR_RESET) Verify latest readiness proof package'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle$(COLOR_RESET) Build dated pilot closeout artifact package'
	@echo '  $(COLOR_GREEN)pilot-closeout-bundle-verify$(COLOR_RESET) Verify latest pilot closeout artifact package'
	@echo ''
	@echo 'Recommended usage:'
	@echo '  make ci-pr            # default for pull requests'
	@echo '  make ci               # local full validation via pinned offline images'
	@echo '  make ci-nightly       # scheduled or pre-release runtime sweep via pinned offline images'
	@echo '  make ci-manual-heavy  # optional local hardened image build/scan lane'
	@echo '  make readiness-evidence # regenerate current enterprise proof'
	@echo '  make pilot-closeout-bundle # assemble the pilot closeout packet'

.PHONY: help-db
help-db:
	@echo '$(COLOR_BOLD)Database Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)db-status$(COLOR_RESET)          Show database status'
	@echo '  $(COLOR_GREEN)chargeback-report$(COLOR_RESET)  Generate chargeback/showback report artifacts'
	@echo '  $(COLOR_GREEN)db-backup$(COLOR_RESET)          Create backup'
	@echo '  $(COLOR_GREEN)db-restore$(COLOR_RESET)         Restore from backup'
	@echo '  $(COLOR_GREEN)db-shell$(COLOR_RESET)           Open database shell'
	@echo '  $(COLOR_GREEN)dr-drill$(COLOR_RESET)           Run DR drill'

.PHONY: help-host
help-host:
	@echo '$(COLOR_BOLD)Host Deployment Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)host-preflight$(COLOR_RESET)     Run preflight checks'
	@echo '  $(COLOR_GREEN)host-check$(COLOR_RESET)         Check mode'
	@echo '  $(COLOR_GREEN)host-apply$(COLOR_RESET)         Apply mode'
	@echo '  $(COLOR_GREEN)host-install$(COLOR_RESET)       Install systemd service'
	@echo '  $(COLOR_GREEN)host-upgrade-prepare$(COLOR_RESET) Prepare upgrade'
	@echo '  $(COLOR_GREEN)host-upgrade-cutover$(COLOR_RESET) Cut over to standby'

.PHONY: help-key
help-key:
	@echo '$(COLOR_BOLD)Virtual Key Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)key-gen$(COLOR_RESET)            Generate key'
	@echo '  $(COLOR_GREEN)key-revoke$(COLOR_RESET)         Revoke key'
	@echo '  $(COLOR_GREEN)key-gen-dev$(COLOR_RESET)        Generate developer key'
	@echo '  $(COLOR_GREEN)rbac-whoami$(COLOR_RESET)        Show current role'
	@echo '  $(COLOR_GREEN)rbac-roles$(COLOR_RESET)         List roles'
	@echo ''
	@echo 'Examples:'
	@echo '  make key-gen ALIAS=alice BUDGET=10.00'
	@echo '  make key-revoke ALIAS=alice'

.PHONY: help-onboard
help-onboard:
	@echo '$(COLOR_BOLD)Tool Onboarding Targets:$(COLOR_RESET)'
	@echo '  $(COLOR_GREEN)onboard$(COLOR_RESET)            Onboard a tool (TOOL=codex|claude|opencode|cursor|copilot)'
	@echo '  $(COLOR_GREEN)onboard-help$(COLOR_RESET)       Show onboarding CLI help'
	@echo '  $(COLOR_GREEN)onboard-codex$(COLOR_RESET)      Codex onboarding shortcut (default MODE=subscription)'
	@echo '  $(COLOR_GREEN)chatgpt-login$(COLOR_RESET)      Trigger ChatGPT OAuth device flow for gateway'
	@echo '  $(COLOR_GREEN)chatgpt-auth-copy$(COLOR_RESET)  Copy local Codex auth cache into LiteLLM container'
	@echo ''
	@echo 'Examples:'
	@echo '  make onboard TOOL=codex MODE=subscription VERIFY=1'
	@echo '  make chatgpt-login'

# Default target
.DEFAULT_GOAL := help
