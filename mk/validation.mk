# AI Control Plane - Validation Targets
#
# Purpose: Configuration and policy validation
# Responsibilities:
#   - Validate detection rules
#   - Validate SIEM queries
#   - Validate supply chain
#
# Non-scope:
#   - Does not fix validation issues
#   - Does not enforce policies

.PHONY: validate-config
validate-config: install-binary ## Validate tracked deployment configuration and config contract
	@echo '$(COLOR_BOLD)Validating deployment configuration contract...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate config \
		&& echo '$(COLOR_GREEN)✓ Configuration validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Configuration validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-config-contract
validate-config-contract: validate-config ## Alias for machine-readable config contract validation

.PHONY: validate-librechat-config
validate-librechat-config: ## Validate required LibreChat environment variables
	@echo '$(COLOR_BOLD)Validating LibreChat configuration...$(COLOR_RESET)'
	@if [ ! -f "$(COMPOSE_ENV_FILE)" ]; then \
		echo '$(COLOR_RED)✗ Missing runtime env file:$(COLOR_RESET) $(COMPOSE_ENV_FILE)'; \
		echo 'Set COMPOSE_ENV_FILE to the env file used for the managed UI overlay.'; \
		exit 1; \
	fi
	@missing=''; \
	for key in LIBRECHAT_CREDS_KEY LIBRECHAT_CREDS_IV LIBRECHAT_MEILI_MASTER_KEY LIBRECHAT_LITELLM_API_KEY JWT_SECRET JWT_REFRESH_SECRET; do \
		value="$$( $(ACPCTL_BIN) env get --file "$(COMPOSE_ENV_FILE)" "$$key" 2>/dev/null || true )"; \
		if [ -z "$$value" ]; then \
			missing="$$missing $$key"; \
		fi; \
	done; \
	if [ -n "$$missing" ]; then \
		echo '$(COLOR_RED)✗ Missing required LibreChat env vars:$(COLOR_RESET)'$$missing; \
		echo 'Populate these values in $(COMPOSE_ENV_FILE), then retry.'; \
		exit 1; \
	fi
	@echo '$(COLOR_GREEN)✓ LibreChat configuration validation passed$(COLOR_RESET)'

.PHONY: validate-doc-links
validate-doc-links: ## Fail when public docs or generated refs use absolute local filesystem links
	@echo '$(COLOR_BOLD)Validating documentation link style...$(COLOR_RESET)'
	@$(GO) test ./cmd/acpctl -run 'TestDocumentationLinksStayRepoRelative' -count=1 \
		&& echo '$(COLOR_GREEN)✓ Documentation link validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Documentation link validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-detections
validate-detections: ## Validate detection rule output
	@echo '$(COLOR_BOLD)Validating detection rules...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate detections \
		&& echo '$(COLOR_GREEN)✓ Detection validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Detection validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: detection
detection: validate-detections ## Compatibility alias for validate-detections

.PHONY: detection-normalized
detection-normalized: validate-detections ## Compatibility alias for validate-detections

.PHONY: validate-siem-queries
validate-siem-queries: ## Validate SIEM query sync
	@echo '$(COLOR_BOLD)Validating SIEM queries...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate siem-queries \
		&& echo '$(COLOR_GREEN)✓ SIEM query validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ SIEM query validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-siem-schema
validate-siem-schema: ## Validate SIEM schema mappings
	@echo '$(COLOR_BOLD)Validating SIEM schema mappings...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate siem-queries --validate-schema \
		&& echo '$(COLOR_GREEN)✓ SIEM schema validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ SIEM schema validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-policy-rules
validate-policy-rules: ## Validate the tracked ACP custom policy rule contract
	@echo '$(COLOR_BOLD)Validating ACP custom policy rules...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate policy-rules --file "$(POLICY_RULES_FILE)" \
		&& echo '$(COLOR_GREEN)✓ Custom policy rule validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Custom policy rule validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: policy-eval
policy-eval: ## Evaluate the sample/local request-response payload against ACP custom policy rules
	@echo '$(COLOR_BOLD)Evaluating ACP custom policy rules...$(COLOR_RESET)'
	@$(ACPCTL_BIN) policy eval \
		--rules-file "$(POLICY_RULES_FILE)" \
		--file "$(POLICY_INPUT)" \
		--output-dir "$(POLICY_OUTPUT_DIR)"
	@echo '$(COLOR_GREEN)✓ Custom policy evaluation complete$(COLOR_RESET)'

.PHONY: validate-compose-healthchecks
validate-compose-healthchecks: ## Validate Docker Compose healthcheck syntax
	@echo '$(COLOR_BOLD)Validating Docker Compose healthchecks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate compose-healthchecks \
		&& echo '$(COLOR_GREEN)✓ Healthcheck validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Healthcheck validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-generated-docs
validate-generated-docs: ## Fail when generated completions/reference docs drift from source
	@echo '$(COLOR_BOLD)Validating generated docs and completion artifacts...$(COLOR_RESET)'
	@$(GO) test ./cmd/acpctl -run 'TestGeneratedCompletionArtifactsAreCurrent|TestGeneratedReferenceArtifactsAreCurrent' -count=1 \
		&& echo '$(COLOR_GREEN)✓ Generated-doc drift validation passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Generated-doc drift validation failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-acpctl-parity
validate-acpctl-parity: ## Fail when published Make/acpctl surfaces drift from the typed command registry
	@echo '$(COLOR_BOLD)Validating acpctl command surface parity...$(COLOR_RESET)'
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo '$(COLOR_RED)✗ go not installed - required for validate-acpctl-parity$(COLOR_RESET)'; \
		exit 2; \
	fi
	@$(GO) test ./cmd/acpctl -run 'TestCommandSpec_ApprovedCommandInventory|TestPublishedMakeTargetsResolve|TestPublishedACPCTLCommandsResolve|TestPublicSurfaceOmitsIncubatingTracks|TestRetiredCommandReferencesStayRemoved' -count=1 \
		&& echo '$(COLOR_GREEN)✓ acpctl command surface parity passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ acpctl command surface parity failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-headers
validate-headers: ## Validate Go source file purpose headers
	@echo '$(COLOR_BOLD)Validating Go file headers...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate headers \
		&& echo '$(COLOR_GREEN)✓ Go header policy passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Go header policy failed$(COLOR_RESET)'; exit 1; }

.PHONY: validate-env-access
validate-env-access: ## Fail on direct environment access outside internal/config
	@echo '$(COLOR_BOLD)Validating direct environment access policy...$(COLOR_RESET)'
	@$(ACPCTL_BIN) validate env-access \
		&& echo '$(COLOR_GREEN)✓ Environment access policy passed$(COLOR_RESET)' \
		|| { echo '$(COLOR_RED)✗ Environment access policy failed$(COLOR_RESET)'; exit 1; }

.PHONY: governance-report
governance-report: ## Compatibility target: governance report generation not included in public snapshot
	@echo '$(COLOR_YELLOW)⚠ Governance scorecard generation is not included in this public snapshot$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)  Use make release-bundle + docs/release/PRESENTATION_READINESS_TRACKER.md for release evidence$(COLOR_RESET)'
	@exit 2

.PHONY: governance-report-json
governance-report-json: governance-report ## Compatibility target for JSON governance scorecard output

.PHONY: governance-report-7d
governance-report-7d: governance-report ## Compatibility target for 7-day governance scorecard output
