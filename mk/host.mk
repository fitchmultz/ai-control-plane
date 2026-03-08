# AI Control Plane - Host Deployment Targets
#
# Purpose: Host-first deployment and operations
# Responsibilities:
#   - Host preflight checks
#   - Host deployment (check/apply)
#   - Systemd service management
#   - Host upgrade operations
#
# Non-scope:
#   - Does not manage Docker containers
#   - Does not handle Kubernetes deployments

INVENTORY ?= deploy/ansible/inventory/hosts.yml

.PHONY: host-preflight
host-preflight: ## Validate host readiness
	@echo '$(COLOR_BOLD)Running host preflight checks...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host preflight \
		--secrets-env-file "$(SECRETS_ENV_FILE)" \
		--compose-env-file "$(HOST_COMPOSE_ENV_FILE)"

.PHONY: host-check
host-check: ## Run declarative host preflight/check mode
	@echo '$(COLOR_BOLD)Running host check mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host check --inventory "$(INVENTORY)"

.PHONY: host-apply
host-apply: ## Run declarative host apply/converge
	@echo '$(COLOR_BOLD)Running host apply mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host apply --inventory "$(INVENTORY)"

.PHONY: host-install
host-install: ## Install systemd service
	@echo '$(COLOR_BOLD)Installing systemd service...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host install \
		--env-file "$(SECRETS_ENV_FILE)" \
		--compose-env-file "$(HOST_COMPOSE_ENV_FILE)"

.PHONY: host-secrets-refresh
host-secrets-refresh: ## Refresh host secrets contract
	@echo '$(COLOR_BOLD)Refreshing host secrets...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host secrets-refresh \
		--secrets-file "$(SECRETS_ENV_FILE)" \
		--compose-env-file "$(HOST_COMPOSE_ENV_FILE)" \
		$(if $(SECRETS_FETCH_HOOK),--fetch-hook "$(SECRETS_FETCH_HOOK)",)

.PHONY: host-uninstall
host-uninstall: ## Uninstall systemd service
	@echo '$(COLOR_BOLD)Uninstalling systemd service...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host uninstall

.PHONY: host-service-status
host-service-status: ## Show service status
	@$(ACPCTL_BIN) host service-status

.PHONY: host-service-start
host-service-start: ## Start service
	@$(ACPCTL_BIN) host service-start \
		--env-file "$(SECRETS_ENV_FILE)" \
		--compose-env-file "$(HOST_COMPOSE_ENV_FILE)"

.PHONY: host-service-stop
host-service-stop: ## Stop service
	@$(ACPCTL_BIN) host service-stop

.PHONY: host-service-restart
host-service-restart: ## Restart service
	@$(ACPCTL_BIN) host service-restart \
		--env-file "$(SECRETS_ENV_FILE)" \
		--compose-env-file "$(HOST_COMPOSE_ENV_FILE)"
