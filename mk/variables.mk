# AI Control Plane - Makefile Variables
#
# Purpose: Centralize all Makefile variables and configuration
# Responsibilities:
#   - Define all variables used across Makefile targets
#   - Auto-detect tools and environment
#   - Export essential environment variables
#
# Non-scope:
#   - Does not define targets
#   - Does not include other makefiles

# Directories
COMPOSE_DIR := demo
COMPOSE_FILE := $(COMPOSE_DIR)/docker-compose.yml
COMPOSE_UI_FILE := $(COMPOSE_DIR)/docker-compose.ui.yml
COMPOSE_DLP_FILE := $(COMPOSE_DIR)/docker-compose.dlp.yml
COMPOSE_OFFLINE_FILE := $(COMPOSE_DIR)/docker-compose.offline.yml
COMPOSE_TLS_FILE := $(COMPOSE_DIR)/docker-compose.tls.yml
COMPOSE_ENV_FILE ?= $(abspath $(COMPOSE_DIR)/.env)

# Auto-detect Docker Compose: prefer V2 (docker compose) over V1 (docker-compose)
DOCKER_COMPOSE := $(shell docker compose version >/dev/null 2>&1 && echo "docker compose" || echo "docker-compose")
ACP_SLOT ?= active
export ACP_SLOT
ACP_COMPOSE_PROJECT ?= ai-control-plane-$(ACP_SLOT)
DOCKER_COMPOSE_PROJECT := $(DOCKER_COMPOSE) --env-file $(COMPOSE_ENV_FILE) --project-name $(ACP_COMPOSE_PROJECT)
COMPOSE_ENV_LITELLM_MASTER_KEY = LITELLM_MASTER_KEY="$$($(ACPCTL_BIN) env get --file "$(COMPOSE_ENV_FILE)" LITELLM_MASTER_KEY 2>/dev/null || true)"
comma := ,
empty :=
space := $(empty) $(empty)
ACP_RUNTIME_OVERLAYS ?=
ACP_RUNTIME_PULL_POLICY ?= never
ACP_RUNTIME_LITELLM_IMAGE ?= ai-control-plane/litellm-hardened:local
ACP_RUNTIME_LIBRECHAT_IMAGE ?= ai-control-plane/librechat-hardened:local
ACP_RUNTIME_PRODUCTION_PROFILE ?= 0
ACP_RUNTIME_OTEL_COLLECTOR_CONFIG_FILE ?= config.yaml

# Detect local Docker socket for CI runtime
DOCKER_LOCAL_SOCKET := $(firstword $(wildcard /var/run/docker.sock /run/docker.sock))
CI_DOCKER_HOST := $(if $(DOCKER_LOCAL_SOCKET),unix://$(DOCKER_LOCAL_SOCKET),)

# Ports and networking
GATEWAY_HOST ?= 127.0.0.1
LITELLM_PORT ?= 4000
TLS_PORT ?= 443
LIBRECHAT_PORT ?= 3080
ACP_GATEWAY_TLS ?=
_gateway_tls_truthy := 1 true TRUE yes YES on ON
ACP_GATEWAY_SCHEME ?= $(if $(filter $(_gateway_tls_truthy),$(strip $(ACP_GATEWAY_TLS))),https,http)
EFFECTIVE_GATEWAY_URL := $(strip $(or $(ACP_GATEWAY_URL),$(GATEWAY_URL),$(ACP_GATEWAY_SCHEME)://$(GATEWAY_HOST):$(LITELLM_PORT)))

# Database configuration
DB_NAME ?= litellm
DB_USER ?= litellm
ENV_FILE_DATABASE_MODE := $(strip $(shell [ -f "$(COMPOSE_ENV_FILE)" ] && awk -F= '/^[[:space:]]*ACP_DATABASE_MODE=/{gsub(/^[[:space:]]+|[[:space:]]+$$/, "", $$2); print $$2; exit}' "$(COMPOSE_ENV_FILE)"))
DB_MODE ?= $(if $(ACP_DATABASE_MODE),$(ACP_DATABASE_MODE),$(if $(ENV_FILE_DATABASE_MODE),$(ENV_FILE_DATABASE_MODE),embedded))
COMPOSE_DB_PROFILE := $(if $(filter embedded,$(DB_MODE)),--profile embedded-db,)

# CI and testing
CI_FULL ?= 0
SCRIPT_TEST_SCOPE ?= auto
SCRIPT_TEST_JOBS ?=
OFFLINE_GATEWAY_READY_MAX_ATTEMPTS ?= 75
PERFORMANCE_GATEWAY_URL ?= $(EFFECTIVE_GATEWAY_URL)
PERFORMANCE_MODEL ?= mock-gpt
PERFORMANCE_REQUESTS ?= 20
PERFORMANCE_CONCURRENCY ?= 2
PERFORMANCE_MAX_TOKENS ?= 32
PERFORMANCE_WAIT_TIMEOUT ?= 150
PERFORMANCE_PROFILE ?=
POLICY_RULES_FILE ?= demo/config/custom_policy_rules.yaml
POLICY_INPUT ?= examples/policy-engine/request_response_eval.sample.json
POLICY_OUTPUT_DIR ?= demo/logs/evidence/policy-eval
TENANT_DESIGN_FILE ?= demo/config/tenant_design.yaml

# Go toolchain
GO ?= go
ACPCTL_BIN ?= .bin/acpctl
GO_PACKAGES ?= ./...
GO_SOURCES := $(shell find cmd internal pkg -name '*.go' 2>/dev/null)

# Secrets Contract Configuration (RQ-0172)
SECRETS_ENV_FILE ?= /etc/ai-control-plane/secrets.env
SECRETS_FETCH_HOOK ?=

# Shellcheck files (tracked shell scripts; BSD/GNU portable)
SHELLCHECK_FILES := $(shell git ls-files '*.sh' 2>/dev/null | while IFS= read -r path; do [ -f "$$path" ] && printf '%s\n' "$$path"; done)

# Supply chain configuration
SUPPLY_CHAIN_ALLOWLIST_WARN_DAYS ?= 45
SUPPLY_CHAIN_ALLOWLIST_FAIL_DAYS ?= 14

# Release bundle configuration
VERSION_FILE ?= VERSION
ACP_VERSION ?= $(strip $(shell [ -f "$(VERSION_FILE)" ] && tr -d '[:space:]' < "$(VERSION_FILE)"))
RELEASE_BUNDLE_VERSION ?= $(if $(ACP_VERSION),$(ACP_VERSION),dev)
RELEASE_BUNDLE_OUT_DIR ?= demo/logs/release-bundles
RELEASE_BUNDLE_NAME ?= ai-control-plane-deploy-$(RELEASE_BUNDLE_VERSION).tar.gz
RELEASE_BUNDLE_PATH ?= $(RELEASE_BUNDLE_OUT_DIR)/$(RELEASE_BUNDLE_NAME)
READINESS_EVIDENCE_OUT_DIR ?= demo/logs/evidence
READINESS_INCLUDE_PRODUCTION ?= 0
PILOT_CLOSEOUT_OUT_DIR ?= demo/logs/pilot-closeout
ASSESSOR_PACKET_OUT_DIR ?= demo/logs/assessor-packet
ASSESSOR_READINESS_RUN_DIR ?=
ASSESSOR_PACKET_RUN_DIR ?=
PILOT_CUSTOMER ?= Falcon Insurance Group
PILOT_NAME ?= Claims Governance Pilot
PILOT_DECISION ?= EXPAND
PILOT_CHARTER ?= examples/falcon-insurance-group/PILOT_CHARTER.md
PILOT_ACCEPTANCE_MEMO ?= examples/falcon-insurance-group/PILOT_ACCEPTANCE_MEMO.md
PILOT_VALIDATION_CHECKLIST ?= examples/falcon-insurance-group/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md
PILOT_OPERATOR_CHECKLIST ?= examples/falcon-insurance-group/PILOT_OPERATOR_HANDOFF_CHECKLIST.md
PILOT_READINESS_RUN_DIR ?=
PILOT_CLOSEOUT_RUN_DIR ?=
