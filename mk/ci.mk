# AI Control Plane - CI/CD Targets
#
# Purpose: Continuous Integration pipeline targets
# Responsibilities:
#   - Local CI gate
#   - CI-specific validation
#   - Cleanup operations
#
# Non-scope:
#   - Does not replace CI/CD pipeline configuration
#   - Does not handle deployment automation

.PHONY: ci
ci: ## Local full CI gate (format, lint, static/security checks, runtime-aware tests via pinned offline images)
	@echo '$(COLOR_BOLD)Running local CI gate...$(COLOR_RESET)'
	@$(MAKE) --silent install-ci
	@$(MAKE) --silent format
	@$(MAKE) --silent ci-pr
	@set -euo pipefail; \
	if ! command -v docker >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker is required for make ci runtime stage$(COLOR_RESET)'; \
		exit 2; \
	fi; \
	if ! docker compose version >/dev/null 2>&1 && ! command -v docker-compose >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker Compose is required for make ci runtime stage$(COLOR_RESET)'; \
		exit 2; \
	fi; \
	ACP_SLOT=ci-runtime $(MAKE) --silent down-offline-clean >/dev/null 2>&1 || true; \
	trap 'ACP_SLOT=ci-runtime $(MAKE) --silent down-offline-clean >/dev/null 2>&1 || true' EXIT; \
	ACP_SLOT=ci-runtime $(MAKE) --silent up-offline-ci; \
	ACP_SLOT=ci-runtime $(MAKE) --silent ci-runtime-checks
	@echo '$(COLOR_GREEN)✓ CI gate passed$(COLOR_RESET)'

.PHONY: ci-pr
ci-pr: ## PR-required checks (fast/deterministic: lint, static checks, unit + policy tests)
	@echo '$(COLOR_BOLD)Running PR-required CI checks...$(COLOR_RESET)'
	@$(MAKE) --silent install-ci
	@$(MAKE) --silent public-hygiene-check
	@$(MAKE) --silent lint-shell
	@$(MAKE) --silent lint-yaml
	@$(MAKE) --silent lint-compose
	@$(MAKE) --silent lint-healthchecks
	@$(MAKE) --silent lint-siem
	@$(MAKE) --silent lint-secrets
	@$(MAKE) --silent license-check
	@$(MAKE) --silent supply-chain-gate
	@$(MAKE) --silent type-check
	@$(MAKE) --silent script-tests
	@$(MAKE) --silent test-go
	@echo '$(COLOR_GREEN)✓ PR-required checks passed$(COLOR_RESET)'

.PHONY: ci-nightly
ci-nightly: ## Nightly checks (PR checks + runtime smoke + release bundle verification via pinned offline images)
	@echo '$(COLOR_BOLD)Running nightly CI checks...$(COLOR_RESET)'
	@set -euo pipefail; \
	$(MAKE) --silent ci-pr; \
	ACP_SLOT=ci-runtime $(MAKE) --silent down-offline-clean >/dev/null 2>&1 || true; \
	trap 'ACP_SLOT=ci-runtime $(MAKE) --silent down-offline-clean >/dev/null 2>&1 || true' EXIT; \
	ACP_SLOT=ci-runtime $(MAKE) --silent up-offline-ci; \
	ACP_SLOT=ci-runtime $(MAKE) --silent ci-runtime-checks; \
	$(MAKE) --silent release-bundle; \
	$(MAKE) --silent release-bundle-verify
	@echo '$(COLOR_GREEN)✓ Nightly checks passed$(COLOR_RESET)'

.PHONY: ci-manual-heavy
ci-manual-heavy: ## Manual heavy checks (nightly checks + local hardened image build/scan)
	@echo '$(COLOR_BOLD)Running manual heavy CI checks...$(COLOR_RESET)'
	@$(MAKE) --silent ci-nightly
	@$(MAKE) --silent hardened-images-build
	@$(MAKE) --silent hardened-images-scan
	@echo '$(COLOR_GREEN)✓ Manual heavy checks passed$(COLOR_RESET)'

.PHONY: ci-fast
ci-fast: ## Fast CI gate (skip runtime tests; keep static/security checks)
	@echo '$(COLOR_BOLD)Running fast CI gate...$(COLOR_RESET)'
	@$(MAKE) --silent install-ci
	@$(MAKE) --silent lint-shell
	@$(MAKE) --silent lint-yaml
	@$(MAKE) --silent license-check
	@$(MAKE) --silent supply-chain-gate
	@$(MAKE) --silent type-check
	@echo '$(COLOR_GREEN)✓ Fast CI gate passed$(COLOR_RESET)'

.PHONY: ci-runtime-checks
ci-runtime-checks: ## CI runtime checks (requires running services; stateless when paired with down-offline-clean)
	@echo '$(COLOR_BOLD)Running CI runtime checks...$(COLOR_RESET)'
	@# The hardened LiteLLM image can take longer to bind on arm64 hosts when the
	@# offline stack is running through emulation. Keep the CI wait budget aligned
	@# with that startup envelope instead of failing a healthy-but-slow boot.
	@set -euo pipefail; \
	attempt=1; \
	max_attempts=$(OFFLINE_GATEWAY_READY_MAX_ATTEMPTS); \
	until curl -fsS \
		-H "Authorization: Bearer $$LITELLM_MASTER_KEY" \
		"http://127.0.0.1:$(LITELLM_PORT)/health" >/dev/null 2>&1; do \
		if [ "$$attempt" -ge "$$max_attempts" ]; then \
			echo '$(COLOR_RED)✗ Offline gateway health endpoint did not become ready in time$(COLOR_RESET)'; \
			exit 1; \
		fi; \
		echo "  waiting for offline gateway readiness ($$attempt/$$max_attempts)..."; \
		attempt=$$((attempt + 1)); \
		sleep 2; \
	done
	@curl -fsS \
		-H "Authorization: Bearer $$LITELLM_MASTER_KEY" \
		"http://127.0.0.1:$(LITELLM_PORT)/v1/models" >/dev/null \
		&& echo '$(COLOR_GREEN)✓ Offline gateway health + model endpoints are ready$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Offline models endpoint check failed$(COLOR_RESET)'; exit 1; }
	@$(MAKE) --silent validate-detections
	@echo '$(COLOR_GREEN)✓ CI runtime checks passed$(COLOR_RESET)'

.PHONY: ci-runtime
ci-runtime: ci-runtime-checks ## Backward-compatible alias

.PHONY: clean
clean: ## Remove artifacts + logs. DESTRUCTIVE: deletes Docker volumes
	@echo '$(COLOR_BOLD)Cleaning up...$(COLOR_RESET)'
	@echo '$(COLOR_RED)WARNING: This will remove Docker volumes and logs.$(COLOR_RESET)'
	@read -p "Continue? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) down -v 2>/dev/null || true; \
		rm -rf $(COMPOSE_DIR)/logs/* 2>/dev/null || true; \
		rm -rf $(RELEASE_BUNDLE_OUT_DIR)/* 2>/dev/null || true; \
		$(GO) clean -cache 2>/dev/null || true; \
		echo '$(COLOR_GREEN)✓ Cleanup complete$(COLOR_RESET)'; \
	else \
		echo 'Cleanup cancelled.'; \
	fi

.PHONY: clean-force
clean-force: ## Force cleanup without prompt (use with caution)
	@echo '$(COLOR_BOLD)Force cleaning up...$(COLOR_RESET)'
	@cd $(COMPOSE_DIR) && $(DOCKER_COMPOSE_PROJECT) down -v 2>/dev/null || true
	@rm -rf $(COMPOSE_DIR)/logs/* 2>/dev/null || true
	@rm -rf $(RELEASE_BUNDLE_OUT_DIR)/* 2>/dev/null || true
	@$(GO) clean -cache 2>/dev/null || true
	@echo '$(COLOR_GREEN)✓ Force cleanup complete$(COLOR_RESET)'

.PHONY: build
build: ## Build artifacts (recreate Docker containers)
	@echo '$(COLOR_BOLD)Building artifacts...$(COLOR_RESET)'
	@$(MAKE) --silent install-binary
	@echo '$(COLOR_GREEN)✓ Build complete$(COLOR_RESET)'

.PHONY: generate
generate: install-binary ## Generate derived files (Helm file sync + shell completions)
	@echo '$(COLOR_BOLD)Generating derived files...$(COLOR_RESET)'
	@$(ACPCTL_BIN) files sync-helm
	@$(MAKE) --silent completions
	@echo '$(COLOR_GREEN)✓ Derived files generated$(COLOR_RESET)'
