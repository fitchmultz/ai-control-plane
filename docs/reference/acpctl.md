# ACPCTL Reference

> Generated from the typed command registry. Do not edit manually.

`acpctl` is the typed implementation engine for supported host-first workflows. `make` remains the primary human operator UX.

## Top-Level Commands

### `ci`

CI and local gate helpers.

| Subcommand | Summary |
| --- | --- |
| `should-run-runtime` | Decide whether runtime checks should run |
| `wait` | Wait for services to become healthy |

Examples:

```bash
./scripts/acpctl.sh ci should-run-runtime --quiet
./scripts/acpctl.sh ci wait --timeout 120
```

### `env`

Strict .env access helpers.

| Subcommand | Summary |
| --- | --- |
| `get` | Read a single env key without shell execution |

Examples:

```bash
./scripts/acpctl.sh env get LITELLM_MASTER_KEY
./scripts/acpctl.sh env get --file demo/.env DATABASE_URL
```

### `chargeback`

Typed chargeback rendering helpers.

| Subcommand | Summary |
| --- | --- |
| `report` | Generate canonical chargeback report artifacts |
| `render` | Render canonical chargeback JSON or CSV |
| `payload` | Render canonical chargeback webhook payload JSON |

Examples:

```bash
./scripts/acpctl.sh chargeback report
./scripts/acpctl.sh chargeback render --format json
./scripts/acpctl.sh chargeback render --format csv
./scripts/acpctl.sh chargeback payload --target generic
```

### `status`

Aggregated system health overview.

Examples:

```bash
./scripts/acpctl.sh status
./scripts/acpctl.sh status --json
./scripts/acpctl.sh status --wide
./scripts/acpctl.sh status --watch --interval 5
```

### `health`

Run service health checks.

Examples:

```bash
./scripts/acpctl.sh health
./scripts/acpctl.sh health --verbose
```

### `doctor`

Environment preflight diagnostics.

Examples:

```bash
./scripts/acpctl.sh doctor
./scripts/acpctl.sh doctor --json
./scripts/acpctl.sh doctor --fix --skip-check db_connectable
./scripts/acpctl.sh doctor --wide
```

### `benchmark`

Lightweight local performance baseline.

| Subcommand | Summary |
| --- | --- |
| `baseline` | Run the local gateway performance baseline |

Examples:

```bash
./scripts/acpctl.sh benchmark baseline
./scripts/acpctl.sh benchmark baseline --profile interactive
./scripts/acpctl.sh benchmark baseline --requests 40 --concurrency 4
./scripts/acpctl.sh benchmark baseline --json
```

### `smoke`

Run truthful runtime smoke checks.

Examples:

```bash
./scripts/acpctl.sh smoke
./scripts/acpctl.sh smoke --verbose
```

### `completion`

Generate shell completion scripts.

| Subcommand | Summary |
| --- | --- |
| `bash` | Generate Bash completion script |
| `zsh` | Generate Zsh completion script |
| `fish` | Generate Fish completion script |

### `onboard`

Launch the guided onboarding wizard.

Examples:

```bash
./scripts/acpctl.sh onboard
./scripts/acpctl.sh onboard codex
make onboard
make onboard-codex
```

### `deploy`

Typed evidence and artifact workflows.

| Subcommand | Summary |
| --- | --- |
| `release-bundle` | Build deployment release bundle |
| `readiness-evidence` | Generate and verify dated readiness evidence |
| `pilot-closeout-bundle` | Assemble and verify a pilot closeout evidence bundle |
| `artifact-retention` | Enforce document artifact retention policy |

Examples:

```bash
./scripts/acpctl.sh deploy readiness-evidence run
./scripts/acpctl.sh deploy release-bundle build
./scripts/acpctl.sh deploy pilot-closeout-bundle build
```

### `validate`

Configuration and policy validation operations.

| Subcommand | Summary |
| --- | --- |
| `lint` | Run static validation/lint gate |
| `config` | Validate deployment configuration (use --production for host contract checks) |
| `detections` | Validate detection rule output |
| `siem-queries` | Validate SIEM query sync |
| `public-hygiene` | Fail when local-only files are tracked by git |
| `license` | Validate license policy structure and restricted references |
| `supply-chain` | Run supply-chain policy and digest validation |
| `secrets-audit` | Run deterministic tracked-file secrets audit |
| `compose-healthchecks` | Validate Docker Compose healthchecks |
| `headers` | Validate Go source file header policy |
| `env-access` | Fail on direct environment access outside internal/config |
| `security` | Run Make-composed security gate (hygiene, secrets, license, supply chain) |

Examples:

```bash
./scripts/acpctl.sh validate config
./scripts/acpctl.sh validate config --production --secrets-env-file /etc/ai-control-plane/secrets.env
./scripts/acpctl.sh validate lint
./scripts/acpctl.sh validate detections
```

### `db`

Database backup, restore, and inspection operations.

| Subcommand | Summary |
| --- | --- |
| `status` | Show database status and statistics |
| `backup` | Create database backup |
| `restore` | Restore embedded database from backup |
| `shell` | Open database shell |
| `dr-drill` | Run database DR restore drill |

Examples:

```bash
./scripts/acpctl.sh db status
./scripts/acpctl.sh db backup
./scripts/acpctl.sh db dr-drill
```

### `key`

Virtual key lifecycle operations.

| Subcommand | Summary |
| --- | --- |
| `gen` | Generate a standard virtual key |
| `revoke` | Revoke a virtual key by alias |
| `gen-dev` | Generate a developer key |
| `gen-lead` | Generate a team-lead key |

Examples:

```bash
./scripts/acpctl.sh key gen alice --budget 10.00
./scripts/acpctl.sh key revoke alice
```

### `host`

Host-first deployment and operations.

| Subcommand | Summary |
| --- | --- |
| `preflight` | Validate host readiness |
| `check` | Run declarative host preflight/check mode |
| `apply` | Run declarative host apply/converge |
| `install` | Install systemd service |
| `uninstall` | Uninstall systemd service |
| `service-status` | Show service status |
| `service-start` | Start the systemd service |
| `service-stop` | Stop the systemd service |
| `service-restart` | Restart the systemd service |

Examples:

```bash
./scripts/acpctl.sh host preflight
./scripts/acpctl.sh host check --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host apply --inventory deploy/ansible/inventory/hosts.yml
./scripts/acpctl.sh host install --service-user acp
```

