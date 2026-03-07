# AI Control Plane - Tool Onboarding Targets
#
# Purpose: Provide one-command onboarding for local CLI/IDE tools
# Responsibilities:
#   - Invoke the native acpctl onboarding workflow
#   - Offer Codex-focused shortcuts for subscription-first demos
#   - Trigger ChatGPT OAuth device login for LiteLLM ChatGPT provider
#
# Non-scope:
#   - Does not start/stop services
#   - Does not manage upstream SaaS accounts

.PHONY: onboard
onboard: ## Onboard a tool (TOOL=codex|claude|opencode|cursor|copilot)
	@if [ -z "$(TOOL)" ]; then \
		echo 'Usage: make onboard TOOL=<tool> [MODE=<mode>] [VERIFY=1] [HOST=<host>] [TLS=1]'; \
		echo 'Try: make onboard-help'; \
		exit 64; \
	fi
	@$(ACPCTL_BIN) onboard "$(TOOL)" \
		$(if $(MODE),--mode "$(MODE)",) \
		$(if $(ALIAS),--alias "$(ALIAS)",) \
		$(if $(BUDGET),--budget "$(BUDGET)",) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(HOST),--host "$(HOST)",) \
		$(if $(PORT),--port "$(PORT)",) \
		$(if $(filter 1 true TRUE yes YES,$(TLS)),--tls,) \
		$(if $(filter 1 true TRUE yes YES,$(VERIFY)),--verify,) \
		$(if $(filter 1 true TRUE yes YES,$(WRITE_CONFIG)),--write-config,) \
		$(if $(filter 1 true TRUE yes YES,$(SHOW_KEY)),--show-key,)

.PHONY: onboard-help
onboard-help: ## Show onboarding script help
	@$(ACPCTL_BIN) onboard --help

.PHONY: onboard-codex
onboard-codex: ## Codex onboarding shortcut (default MODE=subscription)
	@$(ACPCTL_BIN) onboard codex \
		--mode "$(or $(MODE),subscription)" \
		$(if $(ALIAS),--alias "$(ALIAS)",) \
		$(if $(BUDGET),--budget "$(BUDGET)",) \
		$(if $(MODEL),--model "$(MODEL)",) \
		$(if $(HOST),--host "$(HOST)",) \
		$(if $(PORT),--port "$(PORT)",) \
		$(if $(filter 1 true TRUE yes YES,$(TLS)),--tls,) \
		$(if $(filter 1 true TRUE yes YES,$(VERIFY)),--verify,) \
		$(if $(filter 1 true TRUE yes YES,$(WRITE_CONFIG)),--write-config,) \
		$(if $(filter 1 true TRUE yes YES,$(SHOW_KEY)),--show-key,)

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
test-onboard: ## Run onboarding script checks
	@if [ -x scripts/tests/onboard_test.sh ]; then \
		bash scripts/tests/onboard_test.sh; \
	else \
		echo 'scripts/tests/onboard_test.sh not present or not executable'; \
		exit 2; \
	fi
