# AI Control Plane - Virtual Key Targets
#
# Purpose: Virtual key lifecycle operations
# Responsibilities:
#   - Generate virtual keys
#   - Revoke virtual keys
#
# Non-scope:
#   - Does not manage master keys
#   - Does not handle key rotation policies

.PHONY: key-gen
key-gen: ## Generate a standard virtual key
	@$(ACPCTL_BIN) key gen $(if $(ALIAS),$(ALIAS),) $(if $(BUDGET),--budget $(BUDGET),)

.PHONY: key-revoke
key-revoke: ## Revoke a virtual key by alias
	@$(ACPCTL_BIN) key revoke $(if $(ALIAS),$(ALIAS),)

.PHONY: key-gen-dev
key-gen-dev: ## Generate a developer key
	@$(ACPCTL_BIN) key gen-dev $(if $(ALIAS),$(ALIAS),)

.PHONY: key-gen-lead
key-gen-lead: ## Generate a team-lead key
	@$(ACPCTL_BIN) key gen-lead $(if $(ALIAS),$(ALIAS),)
