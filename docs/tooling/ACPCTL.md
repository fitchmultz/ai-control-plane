# ACPCTL (Typed CLI Core)

## Purpose
`acpctl` is the typed Go command core for the AI Control Plane. Canonical operator entrypoints are `make <target>` for day-to-day operations and `./scripts/acpctl.sh <subcommand>` for typed workflows, with `scripts/libexec/*` retained as an internal compatibility layer.

## Responsibilities
- Provide statically typed implementations for control-plane CLI workflows.
- Preserve the repository exit-code contract (`0/1/2/3/64`).
- Keep command behavior deterministic and testable.
- Provide a single stable command surface for operators.

## Non-scope
- `acpctl` does not replace Docker runtime services.
- `acpctl` does not replace internal implementation scripts under `scripts/libexec/`.
- `acpctl` does not change governance semantics without explicit contract updates.

## Invariants
- `make <target>` is the primary operator surface for documented workflows.
- `./scripts/acpctl.sh` is the canonical shell entrypoint for typed subcommands.
- Legacy top-level wrapper scripts are removed.
- Business logic moves into typed modules under `internal/`.

## Command Surface

Primary entrypoints:
- Make targets: `make <target>`
- Shell wrapper: `./scripts/acpctl.sh` (auto-resolves repo-local binary, repo-root binary, or PATH binary; exits 2 with install guidance if none are found)
- Typed binary (optional direct use): `.bin/acpctl` (built via `make install-binary`)

Top-level commands:
- `ci` - CI and local gate helpers
- `chargeback` - typed chargeback reporting, rendering, and payload helpers
- `status` - aggregated system health overview
- `doctor` - environment preflight diagnostics
- `benchmark` - local reference-host performance baseline
- `smoke` - truthful runtime smoke checks
- `bridge` - compatibility execution of `scripts/libexec/*_impl.sh` workflows
- `deploy` - service lifecycle, TLS/offline, Helm, readiness-evidence, and release-bundle flows (delegates to Make targets)
- `validate` - configuration and policy validation flows (typed core checks + selective Make delegation)
- `db` - database backup, restore, inspection, and DR drill flows
- `key` - virtual key lifecycle flows
- `host` - host preflight, declarative deploy, secrets sync, and systemd lifecycle operations
- `demo` - demo scenario/state flows (delegates to Make targets)
- `terraform` - Terraform provisioning workflow flows (delegates to Make targets)

### CI Command (typed business logic)
```bash
./scripts/acpctl.sh ci should-run-runtime --help
```

Primary shell invocation:
```bash
./scripts/acpctl.sh ci should-run-runtime
```

### Chargeback Command (typed business logic)
```bash
./scripts/acpctl.sh chargeback report --help
```

Primary shell invocation:
```bash
./scripts/acpctl.sh chargeback report --format all
```

### Doctor Command (environment preflight diagnostics)
```bash
./scripts/acpctl.sh doctor --help
```

Primary shell invocation:
```bash
./scripts/acpctl.sh doctor
```

The doctor command validates the runtime environment before operations:
- **docker_available**: Docker binary and daemon accessible
- **ports_free**: Required ports (4000, 5432) are available
- **env_vars_set**: Required environment variables configured
- **gateway_healthy**: LiteLLM gateway responding
- **db_connectable**: PostgreSQL accepting connections
- **config_valid**: Deployment configuration valid
- **credentials_valid**: Master key valid and usable

Examples:
```bash
# Human-readable output
./scripts/acpctl.sh doctor

# JSON output for CI integration
./scripts/acpctl.sh doctor --json

# Show extended details
./scripts/acpctl.sh doctor --wide

# Skip specific checks
./scripts/acpctl.sh doctor --skip-check db_connectable --skip-check gateway_healthy

# Attempt auto-remediation
./scripts/acpctl.sh doctor --fix
```

### Bridge Command (compatibility execution)
```bash
./scripts/acpctl.sh bridge --help
```

Bridge currently exists only for a narrow set of compatibility workflows whose implementation still lives under `scripts/libexec/`. Onboarding is now a native root command, with `bridge onboard` retained as a shim for older entrypoints. Use command help for the authoritative surface:

```bash
./scripts/acpctl.sh bridge --help
./scripts/acpctl.sh onboard --help
```

### Operator Flows (mixed typed + delegated)

Each operator flow maps subcommands to existing `make` targets:

```bash
# Deploy
./scripts/acpctl.sh deploy up
./scripts/acpctl.sh deploy health
./scripts/acpctl.sh deploy up-production
./scripts/acpctl.sh deploy up-tls
./scripts/acpctl.sh deploy up-offline
./scripts/acpctl.sh deploy helm-validate
./scripts/acpctl.sh deploy release-bundle
./scripts/acpctl.sh deploy readiness-evidence run
./scripts/acpctl.sh deploy readiness-evidence verify

# Chargeback (typed, no Make delegation)
./scripts/acpctl.sh chargeback report
./scripts/acpctl.sh chargeback report --format all
./scripts/acpctl.sh chargeback render --format json

# Doctor (environment preflight)
./scripts/acpctl.sh doctor
./scripts/acpctl.sh doctor --json
./scripts/acpctl.sh doctor --fix

# Benchmark
./scripts/acpctl.sh benchmark baseline
./scripts/acpctl.sh benchmark baseline --requests 40 --concurrency 4 --json
./scripts/acpctl.sh benchmark baseline --profile interactive --json

# Smoke
./scripts/acpctl.sh smoke
./scripts/acpctl.sh smoke --help

# Pilot closeout bundle
./scripts/acpctl.sh deploy pilot-closeout-bundle build --customer "Example Customer" --pilot-name "Example Pilot" --charter docs/templates/PILOT_CHARTER.md --acceptance-memo docs/templates/PILOT_ACCEPTANCE_MEMO.md --validation-checklist docs/PILOT_CUSTOMER_VALIDATION_CHECKLIST.md --operator-checklist docs/templates/PILOT_OPERATOR_HANDOFF_CHECKLIST.md
./scripts/acpctl.sh deploy pilot-closeout-bundle verify

# Validate
./scripts/acpctl.sh validate lint
./scripts/acpctl.sh validate config
./scripts/acpctl.sh validate detections
./scripts/acpctl.sh validate supply-chain

# DB
./scripts/acpctl.sh db status
./scripts/acpctl.sh db backup
./scripts/acpctl.sh db restore
./scripts/acpctl.sh db restore demo/backups/litellm-backup-20240128-143052.sql.gz
./scripts/acpctl.sh db dr-drill

# Key (typed)
./scripts/acpctl.sh key gen alice --budget 10.00
./scripts/acpctl.sh key gen-dev alice
./scripts/acpctl.sh key gen-lead alice
./scripts/acpctl.sh key revoke alice

# Host
./scripts/acpctl.sh host preflight --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host secrets-refresh --secrets-file /etc/ai-control-plane/secrets.env --compose-env-file demo/.env
./scripts/acpctl.sh host install --env-file /etc/ai-control-plane/secrets.env --compose-env-file demo/.env

# Demo
./scripts/acpctl.sh demo scenario SCENARIO=1
./scripts/acpctl.sh demo preset PRESET=executive-demo
./scripts/acpctl.sh demo snapshot NAME=baseline

# Terraform
./scripts/acpctl.sh terraform init
./scripts/acpctl.sh terraform plan
./scripts/acpctl.sh terraform apply
./scripts/acpctl.sh terraform destroy
./scripts/acpctl.sh terraform fmt
./scripts/acpctl.sh terraform validate
```

Subcommand-level help is available:
```bash
./scripts/acpctl.sh deploy --help
./scripts/acpctl.sh chargeback --help
./scripts/acpctl.sh chargeback report --help
./scripts/acpctl.sh doctor --help
./scripts/acpctl.sh smoke --help
./scripts/acpctl.sh validate --help
./scripts/acpctl.sh host install --help
```

### Testability override

Set `ACPCTL_MAKE_BIN` to override the `make` executable used by delegated flows:
```bash
ACPCTL_MAKE_BIN=/tmp/fake-make ./scripts/acpctl.sh deploy up
```

## Build And Validation
```bash
# Build local binary
make install-binary

# Run local reference-host performance baseline
make performance-baseline
make performance-baseline PERFORMANCE_PROFILE=interactive

# Build pilot closeout artifact set
make pilot-closeout-bundle
make pilot-closeout-bundle-verify

# Typed static/type checks
make type-check

# Full local CI gate
make ci
```

## Migration Strategy
1. Migrate high-complexity scripts first (detections, governance, host workflows).
2. Preserve CLI contracts and exit-code behavior while swapping internals to typed code.
3. Keep shell entrypoints canonical (`./scripts/acpctl.sh`) and remove superseded wrapper scripts.
4. Update docs and Ralph task artifacts as each migration phase completes.

## Bridge Status

`acpctl bridge` is a transitional compatibility layer for a small set of shell-backed workflows. Use `./scripts/acpctl.sh bridge --help` for the current supported surface.

## ACPCTL-First Policy Gate

`make script-tests` enforces this gate contract (implemented by `scripts/tests/acpctl_first_migration_gate_test.sh`).

The gate enforces:
- Changed top-level operator scripts (`scripts/*.sh`) must delegate to `acpctl`.
- Changed top-level operator scripts must remain thin wrappers (default limit: 180 LOC).
- Changed shell scripts fail if newly introduced above complexity threshold (default: 250 LOC), or if they cross that threshold from below.

Legacy operator allowlist has been retired. Compatibility execution is provided via `acpctl bridge` into `scripts/libexec/*_impl.sh` without restoring top-level wrapper scripts.

## Shell Completions

`acpctl` provides shell completion support for bash, zsh, and fish shells to improve CLI discoverability and reduce command errors.

### Generating Completion Scripts

Use the `make completions` target to regenerate completion scripts:

```bash
make completions
```

This generates three files in `scripts/completions/`:
- `acpctl.bash` - Bash completion script
- `acpctl.zsh` - Zsh completion script
- `acpctl.fish` - Fish completion script

### Installing Completions

#### Bash

Source the completion script in your shell:

```bash
source scripts/completions/acpctl.bash
```

To make completions persistent, copy the script to your system's bash completion directory:

```bash
# On most Linux distributions:
cp scripts/completions/acpctl.bash /etc/bash_completion.d/acpctl

# Or for user-local installation:
mkdir -p ~/.bash_completion.d
cp scripts/completions/acpctl.bash ~/.bash_completion.d/acpctl
echo 'source ~/.bash_completion.d/acpctl' >> ~/.bashrc
```

#### Zsh

Ensure completions are enabled and source the script:

```bash
autoload -U compinit && compinit
source scripts/completions/acpctl.zsh
```

To make completions persistent, copy to your Zsh completions directory:

```bash
mkdir -p ~/.zsh/completions
cp scripts/completions/acpctl.zsh ~/.zsh/completions/_acpctl
```

Ensure `~/.zsh/completions` is in your `fpath` by adding to `~/.zshrc`:

```bash
fpath=(~/.zsh/completions $fpath)
```

#### Fish

Copy the completion file to Fish's completions directory:

```bash
mkdir -p ~/.config/fish/completions
cp scripts/completions/acpctl.fish ~/.config/fish/completions/acpctl.fish
```

### Dynamic Completions

The completion system provides intelligent suggestions for:

- **Root commands**: All top-level acpctl commands
- **Group subcommands**: Subcommands for delegated groups (e.g., `deploy up`, `deploy health`)
- **Bridge scripts**: Available bridge script names
- **Dynamic values**: Based on repository configuration:
  - Model names: `MODEL=`, `SCENARIO_MODEL=` (parsed from `demo/config/litellm.yaml`)
  - Scenario IDs: `SCENARIO=` (derived from tracked `demo/config/demo_presets.yaml`)
  - Config keys: `CONFIG_KEY=` (parsed from config YAML files)
  - Preset names: `PRESET=` (parsed from `demo/config/demo_presets.yaml`)

### Testing Completions

Verify completions are working:

```bash
# List all root commands
./scripts/acpctl.sh __complete

# Get deploy subcommands
./scripts/acpctl.sh __complete deploy

# Get key alias completions
./scripts/acpctl.sh __complete key gen ALIAS=

# Get scenario completions
./scripts/acpctl.sh __complete demo scenario SCENARIO=

# Get model completions
./scripts/acpctl.sh __complete key gen MODEL=

# Get preset completions
./scripts/acpctl.sh __complete demo preset PRESET=
```

### Completion Command Reference

Generate completion scripts programmatically:

```bash
# Output bash completion script
./scripts/acpctl.sh completion bash

# Output zsh completion script
./scripts/acpctl.sh completion zsh

# Output fish completion script
./scripts/acpctl.sh completion fish
```

The internal `__complete` subcommand (invoked via `./scripts/acpctl.sh __complete`) is used by shell completion scripts and should not be called directly in normal usage.
