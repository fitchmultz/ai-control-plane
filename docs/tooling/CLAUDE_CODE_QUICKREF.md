# Claude Code Testing - Quick Reference

## Quick Start

### One-Command Onboarding (Recommended)
```bash
# Onboard Claude Code in API key mode
make onboard TOOL=claude MODE=api-key

# Onboard in MAX subscription mode
make onboard TOOL=claude MODE=subscription

# With connectivity verification
make onboard TOOL=claude MODE=api-key VERIFY=1

# For remote Docker host
make onboard TOOL=claude MODE=api-key HOST=GATEWAY_HOST
```

### Manual Mode Switching
```bash
# Activate API key mode
cp ~/.claude/settings.local.json ~/.claude/settings.json

# Activate MAX subscription mode
cp ~/.claude/settings.max.json ~/.claude/settings.json
```

## Virtual Keys

| Mode | Key Alias | How to create |
|------|-----------|---------------|
| API Key | `claude-code` | `make onboard TOOL=claude MODE=api-key` |
| MAX Subscription | `claude-code-max` | `make onboard TOOL=claude MODE=subscription` |

## Configuration Files

- `~/.claude/settings.local.json` - API key mode
- `~/.claude/settings.max.json` - MAX subscription mode  
- `~/.claude/settings.json` - Active config (copied from above)

**Note:** Configs are stored in `~/.claude/` (home directory) to avoid committing secrets to the repo.

## Key Differences

### API Key Mode
- Uses `ANTHROPIC_AUTH_TOKEN` with virtual key
- Full gateway enforcement (budgets, rate limits, policies)
- Requires `ANTHROPIC_API_KEY` in `demo/.env`
- Per-tool cost tracking

### MAX Subscription Mode
- Uses `ANTHROPIC_CUSTOM_HEADERS` with virtual key
- Claude Code signs in with MAX subscription (OAuth)
- Gateway enforces policies and logs requests, OAuth forwarded to Anthropic
- No upstream API key needed
- Full governance mode for routed subscription traffic

## Monitoring

```bash
# Real-time logs
make logs

# Database status
make db-status

# Health check
make health
```

## Documentation

Full guide: `CLAUDE_CODE_TESTING.md`

## Script Help

```bash
# Onboarding help
make onboard-help

# Onboarding help
./scripts/acpctl.sh onboard claude --help
```
