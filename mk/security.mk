# AI Control Plane - Security Targets
#
# Purpose: Security validation and supply chain checks
# Responsibilities:
#   - Secrets audit
#   - Public-release tracked-file hygiene enforcement
#   - Supply-chain policy validation and expiry guard
#   - License boundary checks
#   - Deterministic supply-chain report generation
#   - Optional hardened image build/scan workflows
#
# Non-scope:
#   - Does not run hosted CI scanners
#   - Does not replace manual security review

.PHONY: secrets-audit
secrets-audit: ## Run secrets and token leak audit
	@echo '$(COLOR_BOLD)Running secrets audit...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate secrets-audit \
		&& echo '$(COLOR_GREEN)✓ Secrets audit passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Secrets audit failed$(COLOR_RESET)'; exit 1; }

.PHONY: public-hygiene-check
public-hygiene-check: ## Fail if local-only secrets/artifacts are tracked by git
	@echo '$(COLOR_BOLD)Checking public-release tracked file hygiene...$(COLOR_RESET)'
	@if ! command -v git >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ git is required for public-hygiene-check$(COLOR_RESET)'; \
		exit 2; \
	fi
	@violations="$$(git ls-files | grep -E '^(\.env$$|demo/\.env$$|demo/.*/\.env$$|demo/logs/|demo/backups/|handoff-packet/|\.ralph/|docs/presentation/slides-internal/|\.scratchpad\.md$$)' | grep -Ev '/\.gitkeep$$|/\.gitignore$$' || true)"; \
	if [ -n "$$violations" ]; then \
		echo '$(COLOR_RED)✗ Local-only files are tracked and block public release:$(COLOR_RESET)'; \
		printf '%s\n' "$$violations"; \
		echo '$(COLOR_YELLOW)Remove from git index (git rm --cached ...) and keep in .gitignore.$(COLOR_RESET)'; \
		exit 1; \
	fi
	@echo '$(COLOR_GREEN)✓ Public-release tracked file hygiene passed$(COLOR_RESET)'

.PHONY: security-gate
security-gate: ## Run full security gate bundle (hygiene, secrets, license, supply chain)
	@echo '$(COLOR_BOLD)Running full security gate...$(COLOR_RESET)'
	@$(MAKE) --silent public-hygiene-check
	@$(MAKE) --silent secrets-audit
	@$(MAKE) --silent license-check
	@$(MAKE) --silent supply-chain-gate
	@echo '$(COLOR_GREEN)✓ Security gate passed$(COLOR_RESET)'

.PHONY: license-check
license-check: ## Enforce third-party license boundary policy and restricted-pattern checks
	@echo '$(COLOR_BOLD)Checking third-party license boundaries...$(COLOR_RESET)'
	@if ! command -v jq >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ jq is required for license-check$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if ! command -v rg >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ rg (ripgrep) is required for license-check$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if [ ! -f docs/policy/THIRD_PARTY_LICENSE_MATRIX.json ]; then \
		echo '$(COLOR_RED)✗ Missing policy file: docs/policy/THIRD_PARTY_LICENSE_MATRIX.json$(COLOR_RESET)'; \
		exit 1; \
	fi
	@jq -e '(.schema_version != null) and (.policy_id != null) and ((.scan_scope.include | length) > 0) and ((.restricted_components | type) == "array")' \
		docs/policy/THIRD_PARTY_LICENSE_MATRIX.json >/dev/null \
		|| { echo '$(COLOR_RED)✗ License policy JSON missing required fields$(COLOR_RESET)'; exit 1; }
	@if rg --glob '!docs/**' --glob '!demo/logs/**' --glob '!handoff-packet/**' --glob '!.ralph/**' --glob '!mk/security.mk' \
		-n 'litellm-enterprise|from litellm\.enterprise|import litellm\.enterprise' . >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Restricted LiteLLM enterprise references detected outside docs$(COLOR_RESET)'; \
		rg --glob '!docs/**' --glob '!demo/logs/**' --glob '!handoff-packet/**' --glob '!.ralph/**' --glob '!mk/security.mk' \
			-n 'litellm-enterprise|from litellm\.enterprise|import litellm\.enterprise' .; \
		exit 1; \
	fi
	@echo '$(COLOR_GREEN)✓ License boundary check passed$(COLOR_RESET)'

.PHONY: license-report-update
license-report-update: ## Validate policy and remind maintainers where to update the committed report
	@echo '$(COLOR_BOLD)Validating license report prerequisites...$(COLOR_RESET)'
	@$(MAKE) --silent license-check
	@echo '$(COLOR_GREEN)✓ License policy validated$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)ℹ Update docs/deployment/THIRD_PARTY_LICENSE_SUMMARY.md when policy scope/allowlists change$(COLOR_RESET)'

# Supply Chain Security Targets
.PHONY: supply-chain-gate
supply-chain-gate: ## Validate supply-chain policy contract and digest pinning baseline
	@echo '$(COLOR_BOLD)Running supply-chain security gate...$(COLOR_RESET)'
	@if ! command -v jq >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ jq is required for supply-chain-gate$(COLOR_RESET)'; \
		exit 2; \
	fi
	@if [ ! -f demo/config/supply_chain_vulnerability_policy.json ]; then \
		echo '$(COLOR_RED)✗ Missing policy file: demo/config/supply_chain_vulnerability_policy.json$(COLOR_RESET)'; \
		exit 1; \
	fi
	@jq -e '(.policy_id != null) and (.allowlist | type == "array") and (.severity_policy.fail_on | type == "array")' \
		demo/config/supply_chain_vulnerability_policy.json >/dev/null \
		|| { echo '$(COLOR_RED)✗ Supply-chain policy JSON missing required fields$(COLOR_RESET)'; exit 1; }
	@if grep -H -E '^[[:space:]]*image:[[:space:]]+' demo/docker-compose*.yml | grep -vq '@sha256:'; then \
		echo '$(COLOR_RED)✗ Found non-digest-pinned image reference(s) in demo/docker-compose*.yml$(COLOR_RESET)'; \
		grep -H -E '^[[:space:]]*image:[[:space:]]+' demo/docker-compose*.yml | grep -v '@sha256:'; \
		exit 1; \
	fi
	@$(MAKE) --silent supply-chain-allowlist-expiry-check
	@echo '$(COLOR_GREEN)✓ Supply-chain gate passed (policy + digest + expiry checks)$(COLOR_RESET)'

.PHONY: supply-chain-allowlist-expiry-check
supply-chain-allowlist-expiry-check: ## Check allowlist expiry windows against warn/fail thresholds
	@echo '$(COLOR_BOLD)Checking supply-chain allowlist expiry...$(COLOR_RESET)'
	@if ! command -v python3 >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ python3 is required for allowlist expiry checks$(COLOR_RESET)'; \
		exit 2; \
	fi
	@python3 scripts/libexec/check_supply_chain_allowlist_expiry_impl.py \
		--policy demo/config/supply_chain_vulnerability_policy.json \
		--warn-days $(SUPPLY_CHAIN_ALLOWLIST_WARN_DAYS) \
		--fail-days $(SUPPLY_CHAIN_ALLOWLIST_FAIL_DAYS)

.PHONY: supply-chain-report
supply-chain-report: ## Generate local supply-chain policy summary artifact
	@echo '$(COLOR_BOLD)Generating supply-chain policy summary...$(COLOR_RESET)'
	@if ! command -v jq >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ jq is required for supply-chain-report$(COLOR_RESET)'; \
		exit 2; \
	fi
	@mkdir -p demo/logs/supply-chain
	@jq '{generated_at_utc:(if (env.SOURCE_DATE_EPOCH? // "") != "" then ((env.SOURCE_DATE_EPOCH | tonumber) | todate) else (now | todate) end),policy_id,allowlist_count:(.allowlist | length),fail_on:(.severity_policy.fail_on // []),max_counts:(.severity_policy.max_counts // {}),status:"policy_validated"}' demo/config/supply_chain_vulnerability_policy.json > demo/logs/supply-chain/summary.json
	@echo '$(COLOR_GREEN)✓ Wrote demo/logs/supply-chain/summary.json$(COLOR_RESET)'

# Hardened Images Targets
.PHONY: hardened-images-build
hardened-images-build: ## Build local hardened LiteLLM/LibreChat candidate images
	@echo '$(COLOR_BOLD)Building hardened candidate images...$(COLOR_RESET)'
	@if ! command -v docker >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ Docker is required for hardened image build$(COLOR_RESET)'; \
		exit 2; \
	fi
	@docker build -t ai-control-plane/litellm-hardened:local demo/images/litellm-hardened
	@docker build -t ai-control-plane/librechat-hardened:local demo/images/librechat-hardened
	@echo '$(COLOR_GREEN)✓ Hardened candidate images built$(COLOR_RESET)'

.PHONY: hardened-images-scan
hardened-images-scan: hardened-images-build ## Scan hardened candidate images with Trivy
	@echo '$(COLOR_BOLD)Scanning hardened candidate images with Trivy...$(COLOR_RESET)'
	@if ! command -v trivy >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ trivy is required for hardened image scan$(COLOR_RESET)'; \
		echo '$(COLOR_YELLOW)Install: https://trivy.dev/latest/getting-started/installation/$(COLOR_RESET)'; \
		exit 2; \
	fi
	@trivy image --no-progress --severity CRITICAL,HIGH --exit-code 1 ai-control-plane/litellm-hardened:local
	@trivy image --no-progress --severity CRITICAL,HIGH --exit-code 1 ai-control-plane/librechat-hardened:local
	@echo '$(COLOR_GREEN)✓ Hardened image scan passed$(COLOR_RESET)'

.PHONY: hardened-images-gate
hardened-images-gate: hardened-images-scan ## Build and gate hardened candidates
	@echo '$(COLOR_GREEN)✓ Hardened image gate passed$(COLOR_RESET)'
