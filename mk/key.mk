# AI Control Plane - Virtual Key Targets
#
# Purpose: Virtual key lifecycle operations
# Responsibilities:
#   - Generate virtual keys
#   - List, inspect, rotate, and revoke virtual keys
#   - Provide role-shaped key presets
#
# Non-scope:
#   - Does not manage master keys
#   - Does not implement gateway-side key storage

.PHONY: key-gen
key-gen: install-binary ## Generate a standard virtual key
	@$(ACPCTL_BIN) key gen $(if $(ALIAS),$(ALIAS),) \
		$(if $(BUDGET),--budget $(BUDGET),) \
		$(if $(RPM),--rpm $(RPM),) \
		$(if $(TPM),--tpm $(TPM),) \
		$(if $(PARALLEL),--parallel $(PARALLEL),) \
		$(if $(DURATION),--duration $(DURATION),) \
		$(if $(ROLE),--role $(ROLE),) \
		$(if $(filter 1 true TRUE yes YES,$(DRY_RUN)),--dry-run,)

.PHONY: key-list
key-list: install-binary ## List virtual keys
	@$(ACPCTL_BIN) key list $(if $(filter 1 true TRUE yes YES,$(JSON)),--json,)

.PHONY: key-inspect
key-inspect: install-binary ## Inspect a virtual key and its usage
	@$(ACPCTL_BIN) key inspect $(if $(ALIAS),$(ALIAS),) \
		$(if $(REPORT_MONTH),--month $(REPORT_MONTH),) \
		$(if $(filter 1 true TRUE yes YES,$(JSON)),--json,)

.PHONY: key-rotate
key-rotate: install-binary ## Stage or execute virtual key rotation
	@$(ACPCTL_BIN) key rotate $(if $(ALIAS),$(ALIAS),) \
		$(if $(REPLACEMENT_ALIAS),--replacement-alias $(REPLACEMENT_ALIAS),) \
		$(if $(BUDGET),--budget $(BUDGET),) \
		$(if $(RPM),--rpm $(RPM),) \
		$(if $(TPM),--tpm $(TPM),) \
		$(if $(PARALLEL),--parallel $(PARALLEL),) \
		$(if $(DURATION),--duration $(DURATION),) \
		$(if $(ROLE),--role $(ROLE),) \
		$(if $(REPORT_MONTH),--month $(REPORT_MONTH),) \
		$(if $(filter 1 true TRUE yes YES,$(DRY_RUN)),--dry-run,) \
		$(if $(filter 1 true TRUE yes YES,$(REVOKE_OLD)),--revoke-old,) \
		$(if $(filter 1 true TRUE yes YES,$(JSON)),--json,)

.PHONY: key-revoke
key-revoke: install-binary ## Revoke a virtual key by alias
	@$(ACPCTL_BIN) key revoke $(if $(ALIAS),$(ALIAS),)

.PHONY: key-gen-dev
key-gen-dev: install-binary ## Generate a developer key
	@$(ACPCTL_BIN) key gen-dev $(if $(ALIAS),$(ALIAS),) \
		$(if $(BUDGET),--budget $(BUDGET),) \
		$(if $(RPM),--rpm $(RPM),) \
		$(if $(TPM),--tpm $(TPM),) \
		$(if $(PARALLEL),--parallel $(PARALLEL),) \
		$(if $(DURATION),--duration $(DURATION),) \
		$(if $(filter 1 true TRUE yes YES,$(DRY_RUN)),--dry-run,)

.PHONY: key-gen-lead
key-gen-lead: install-binary ## Generate a team-lead key
	@$(ACPCTL_BIN) key gen-lead $(if $(ALIAS),$(ALIAS),) \
		$(if $(BUDGET),--budget $(BUDGET),) \
		$(if $(RPM),--rpm $(RPM),) \
		$(if $(TPM),--tpm $(TPM),) \
		$(if $(PARALLEL),--parallel $(PARALLEL),) \
		$(if $(DURATION),--duration $(DURATION),) \
		$(if $(filter 1 true TRUE yes YES,$(DRY_RUN)),--dry-run,)
