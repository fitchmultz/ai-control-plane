# AI Control Plane - Tool Onboarding Targets
#
# Purpose: Provide guided one-command onboarding entrypoints for supported local tools.
# Responsibilities:
#   - Invoke the native acpctl onboarding wizard.
#   - Offer tool-preselected shortcuts for the supported onboarding surface.
#   - Trigger ChatGPT OAuth device login for LiteLLM ChatGPT provider.
#
# Non-scope:
#   - Does not start or stop services.
#   - Does not manage upstream SaaS accounts.

.PHONY: onboard
onboard: ## Launch the guided onboarding wizard
	@$(ACPCTL_BIN) onboard

.PHONY: onboard-help
onboard-help: ## Show onboarding wizard help
	@$(ACPCTL_BIN) onboard --help

.PHONY: onboard-codex
onboard-codex: ## Launch the onboarding wizard with Codex preselected
	@$(ACPCTL_BIN) onboard codex

.PHONY: onboard-claude
onboard-claude: ## Launch the onboarding wizard with Claude Code preselected
	@$(ACPCTL_BIN) onboard claude

.PHONY: onboard-opencode
onboard-opencode: ## Launch the onboarding wizard with OpenCode preselected
	@$(ACPCTL_BIN) onboard opencode

.PHONY: onboard-cursor
onboard-cursor: ## Launch the onboarding wizard with Cursor preselected
	@$(ACPCTL_BIN) onboard cursor

.PHONY: chatgpt-login
chatgpt-login: ## Trigger ChatGPT OAuth device login for LiteLLM provider
	@bash scripts/libexec/chatgpt_login_impl.sh \
		$(if $(HOST),--host "$(HOST)",) \
		$(if $(PORT),--port "$(PORT)",) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(filter 1 true TRUE yes YES,$(TLS)),--tls,)

.PHONY: chatgpt-auth-copy
chatgpt-auth-copy: ## Copy local Codex auth cache into running LiteLLM container
	@bash scripts/libexec/chatgpt_auth_cache_copy_impl.sh \
		$(if $(AUTH_FILE),--auth-file "$(AUTH_FILE)",) \
		$(if $(CONTAINER),--container "$(CONTAINER)",)

.PHONY: test-onboard
test-onboard: ## Run onboarding shell contract checks
	@bash scripts/tests/onboard_help_contract_test.sh
	@bash scripts/tests/onboard_export_contract_test.sh
	@bash scripts/tests/onboard_verify_mode_test.sh
