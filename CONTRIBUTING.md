# Contributing

Thanks for contributing to AI Control Plane.

## Quick Start

```bash
make install
make ci
```

## Local Development Workflow

1. Create a focused branch from `main`
2. Make changes in small, reviewable commits
3. Run validation gates before opening a PR:

```bash
make format
make lint
make type-check
make test
make ci
```

## Runtime Validation (Optional During Feature Work)

```bash
make up-offline
make health
./scripts/acpctl.sh doctor
```

## Commit and PR Expectations

- Keep commit messages explicit and easy to follow
- Update docs when behavior, commands, or policies change
- Add/adjust tests for functional changes
- Preserve deterministic behavior in scripts and tests

## Repository Hygiene Rules

Do not commit:
- `.env` files or credentials
- generated logs/evidence bundles under `demo/logs/` or `handoff-packet/`
- local agent/workflow state under `.ralph/`

See [docs/ARTIFACTS.md](docs/ARTIFACTS.md) for artifact handling policy.

## CLI and Script Standards

- Shell scripts must use `set -euo pipefail`
- Executable scripts must provide `--help` output with examples and exit codes
- Use the standardized exit code contract from `internal/exitcodes/exitcodes.go`

## Questions

If requirements are unclear, open an issue before implementing large changes.
