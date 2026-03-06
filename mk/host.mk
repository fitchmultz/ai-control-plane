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
	@$(ACPCTL_BIN) host preflight

.PHONY: host-check
host-check: ## Run declarative host preflight/check mode
	@echo '$(COLOR_BOLD)Running host check mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host check INVENTORY="$(INVENTORY)"

.PHONY: host-apply
host-apply: ## Run declarative host apply/converge
	@echo '$(COLOR_BOLD)Running host apply mode...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host apply INVENTORY="$(INVENTORY)"

.PHONY: host-install
host-install: ## Install systemd service
	@echo '$(COLOR_BOLD)Installing systemd service...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host install

.PHONY: host-secrets-refresh
host-secrets-refresh: ## Refresh host secrets contract
	@echo '$(COLOR_BOLD)Refreshing host secrets...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host secrets-refresh

.PHONY: host-uninstall
host-uninstall: ## Uninstall systemd service
	@echo '$(COLOR_BOLD)Uninstalling systemd service...$(COLOR_RESET)'
	@$(ACPCTL_BIN) host uninstall

.PHONY: host-service-status
host-service-status: ## Show service status
	@$(ACPCTL_BIN) host service-status

.PHONY: host-service-start
host-service-start: ## Start service
	@$(ACPCTL_BIN) host service-start

.PHONY: host-service-stop
host-service-stop: ## Stop service
	@$(ACPCTL_BIN) host service-stop

.PHONY: host-service-restart
host-service-restart: ## Restart service
	@$(ACPCTL_BIN) host service-restart

# Host Upgrade Targets
.PHONY: host-upgrade-prepare
host-upgrade-prepare: ## Prepare standby slot
	@$(ACPCTL_BIN) host upgrade-prepare

.PHONY: host-upgrade-smoke-standby
host-upgrade-smoke-standby: ## Smoke test standby slot
	@$(ACPCTL_BIN) host upgrade-smoke-standby

.PHONY: host-upgrade-cutover
host-upgrade-cutover: ## Cut over to standby slot
	@$(ACPCTL_BIN) host upgrade-cutover

.PHONY: host-upgrade-rollback
host-upgrade-rollback: ## Rollback traffic to active slot
	@$(ACPCTL_BIN) host upgrade-rollback

.PHONY: host-upgrade-status
host-upgrade-status: ## Show slot upgrade status
	@$(ACPCTL_BIN) host upgrade-status

.PHONY: host-upgrade-rehearse
host-upgrade-rehearse: ## Run full upgrade rehearsal
	@$(ACPCTL_BIN) host upgrade-rehearse
