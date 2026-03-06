# AI Control Plane - Release Bundle Targets
#
# Purpose: Build and verify deployment release bundles
# Responsibilities:
#   - Build versioned deployment bundles
#   - Verify bundle checksums
#   - Generate and verify readiness evidence packs
#   - Manage artifact retention
#
# Non-scope:
#   - Does not deploy bundles to environments
#   - Does not manage release versioning

.PHONY: release-bundle
release-bundle: install-binary ## Build versioned deployment bundle and verify checksums
	@echo '$(COLOR_BOLD)Building release bundle...$(COLOR_RESET)'
	@$(ACPCTL_BIN) deploy release-bundle build \
		--version "$(RELEASE_BUNDLE_VERSION)" \
		--output-dir "$(RELEASE_BUNDLE_OUT_DIR)"
	@echo ''
	@echo '$(COLOR_BOLD)Verifying release bundle...$(COLOR_RESET)'
	@$(ACPCTL_BIN) deploy release-bundle verify \
		--bundle "$(RELEASE_BUNDLE_PATH)"
	@echo ''
	@echo '$(COLOR_BOLD)Applying artifact retention policy...$(COLOR_RESET)'
	@$(ACPCTL_BIN) deploy artifact-retention --apply --keep-evidence 1 --keep-bundles 1
	@echo ''
	@echo '$(COLOR_GREEN)✓ Release bundle built and verified$(COLOR_RESET)'

.PHONY: release-bundle-verify
release-bundle-verify: install-binary ## Verify release bundle checksum manifest
	@$(ACPCTL_BIN) deploy release-bundle verify \
		--bundle "$(RELEASE_BUNDLE_PATH)"

.PHONY: readiness-evidence
readiness-evidence: install-binary ## Generate a timestamped readiness evidence pack
	@echo '$(COLOR_BOLD)Generating readiness evidence...$(COLOR_RESET)'
	@set -euo pipefail; \
	args='--output-dir "$(READINESS_EVIDENCE_OUT_DIR)" --bundle-version "$(RELEASE_BUNDLE_VERSION)"'; \
	if [ "$(READINESS_INCLUDE_PRODUCTION)" = "1" ]; then \
		args="$$args --include-production --secrets-env-file \"$(SECRETS_ENV_FILE)\""; \
	fi; \
	eval '$(ACPCTL_BIN) deploy readiness-evidence run '$$args

.PHONY: readiness-evidence-verify
readiness-evidence-verify: install-binary ## Verify the latest readiness evidence pack
	@$(ACPCTL_BIN) deploy readiness-evidence verify

.PHONY: pilot-closeout-bundle
pilot-closeout-bundle: install-binary ## Build a local pilot closeout bundle from example or customer-specific docs
	@echo '$(COLOR_BOLD)Building pilot closeout bundle...$(COLOR_RESET)'
	@set -euo pipefail; \
	args='--output-dir "$(PILOT_CLOSEOUT_OUT_DIR)" --customer "$(PILOT_CUSTOMER)" --pilot-name "$(PILOT_NAME)" --decision "$(PILOT_DECISION)" --charter "$(PILOT_CHARTER)" --acceptance-memo "$(PILOT_ACCEPTANCE_MEMO)" --validation-checklist "$(PILOT_VALIDATION_CHECKLIST)"'; \
	if [ -n "$(PILOT_OPERATOR_CHECKLIST)" ]; then \
		args="$$args --operator-checklist \"$(PILOT_OPERATOR_CHECKLIST)\""; \
	fi; \
	if [ -n "$(PILOT_READINESS_RUN_DIR)" ]; then \
		args="$$args --readiness-run-dir \"$(PILOT_READINESS_RUN_DIR)\""; \
	fi; \
	eval '$(ACPCTL_BIN) deploy pilot-closeout-bundle build '$$args

.PHONY: pilot-closeout-bundle-verify
pilot-closeout-bundle-verify: install-binary ## Verify the latest pilot closeout bundle
	@set -euo pipefail; \
	args=''; \
	if [ -n "$(PILOT_CLOSEOUT_RUN_DIR)" ]; then \
		args='--run-dir "$(PILOT_CLOSEOUT_RUN_DIR)"'; \
	fi; \
	eval '$(ACPCTL_BIN) deploy pilot-closeout-bundle verify '$$args

.PHONY: artifact-retention-check
artifact-retention-check: ## Check stale handoff/release document artifacts
	@echo '$(COLOR_BOLD)Checking document artifact retention...$(COLOR_RESET)'
	@$(ACPCTL_BIN) deploy artifact-retention --check --keep-evidence 1 --keep-bundles 1 \
		&& echo '$(COLOR_GREEN)✓ Artifact retention check passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Artifact retention check failed$(COLOR_RESET)'; exit 1; }

.PHONY: artifact-retention-apply
artifact-retention-apply: ## Apply artifact retention policy (destructive)
	@echo '$(COLOR_BOLD)Applying artifact retention policy...$(COLOR_RESET)'
	@$(ACPCTL_BIN) deploy artifact-retention --apply --keep-evidence 1 --keep-bundles 1
	@echo '$(COLOR_GREEN)✓ Artifact retention policy applied$(COLOR_RESET)'
