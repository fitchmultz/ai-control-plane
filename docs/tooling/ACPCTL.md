# ACPCTL

`acpctl` is the typed implementation engine for the supported host-first surface. `make` remains the primary human operator interface. The canonical command inventory is generated from the typed registry and published in [acpctl.md](../reference/acpctl.md).

## Contract

- Use `make <target>` for day-to-day runtime operations.
- Use `./scripts/acpctl.sh <command>` for typed workflows and machine-oriented checks.
- Supported roots are limited to the host-first surface: `ci`, `env`, `chargeback`, `status`, `health`, `doctor`, `benchmark`, `smoke`, `completion`, `onboard`, `deploy`, `validate`, `db`, `key`, and `host`.
- `deploy` is restricted to typed artifact workflows only: release bundle, readiness evidence, pilot closeout, and artifact retention.
- Incubating deployment tracks are not part of the public `acpctl` surface.

## Gateway operator contract

Use one gateway variable everywhere in operator shells:

```bash
export GATEWAY_URL="${GATEWAY_URL:-http://${GATEWAY_HOST:-127.0.0.1}:${LITELLM_PORT:-4000}}"
MASTER_KEY="$(./scripts/acpctl.sh env get LITELLM_MASTER_KEY)"
```

Resolution order:
1. `ACP_GATEWAY_URL` or `GATEWAY_URL`
2. `GATEWAY_HOST` + `LITELLM_PORT` + `ACP_GATEWAY_SCHEME`/`ACP_GATEWAY_TLS`
3. `http://127.0.0.1:4000`

For remote TLS on standard 443, prefer setting `GATEWAY_URL=https://gateway.example.com` directly.

## Guided onboarding

Use the guided wizard for supported local tools:

```bash
make onboard
```

You can optionally preselect a tool and skip the first prompt:

```bash
make onboard-codex
make onboard-claude
make onboard-opencode
make onboard-cursor
```

Or use the typed entrypoint directly:

```bash
./scripts/acpctl.sh onboard
./scripts/acpctl.sh onboard codex
```

Legacy onboarding flags such as `--mode`, `--verify`, `--write-config`, and `--show-key` were removed. The wizard now asks for tool, mode, gateway address, verification, and any tool-specific setup choices interactively. It also lints the emitted env/config contract, validates ACP-managed writes, and when verification is enabled, checks reachability and authorized model access before reporting `Onboarding complete.`

## References

- [ACPCTL Reference](../reference/acpctl.md)
- [Support](../SUPPORT.md)
- [Deployment](../DEPLOYMENT.md)

## Generating Completion Scripts

Use the `make completions` target to regenerate completion scripts:

```bash
make completions
```

This generates three files in `scripts/completions/`:
- `acpctl.bash` - Bash completion script
- `acpctl.zsh` - Zsh completion script
- `acpctl.fish` - Fish completion script

## Installing Completions

### Bash

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

### Zsh

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

### Fish

Copy the completion file to Fish's completions directory:

```bash
mkdir -p ~/.config/fish/completions
cp scripts/completions/acpctl.fish ~/.config/fish/completions/acpctl.fish
```

## Dynamic Completions

The completion system provides intelligent suggestions for:

- **Root commands**: All top-level acpctl commands
- **Group subcommands**: Subcommands for delegated groups (e.g., `deploy up`, `deploy health`)
- **Dynamic values**: Based on repository configuration:
  - Model names: `MODEL=`, `SCENARIO_MODEL=` (parsed from `demo/config/litellm.yaml`)
  - Scenario IDs: `SCENARIO=` (derived from tracked `demo/config/demo_presets.yaml`)
  - Config keys: `CONFIG_KEY=` (parsed from config YAML files)
  - Preset names: `PRESET=` (parsed from `demo/config/demo_presets.yaml`)

## Testing Completions

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

## Completion Command Reference

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
