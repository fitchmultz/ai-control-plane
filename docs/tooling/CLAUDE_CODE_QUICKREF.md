# Claude Code - Quick Reference

## Quick start

### API-key mode (default)

```bash
make onboard-claude
```

Accept the default `api-key` mode in the wizard. The wizard prints the environment variables Claude Code needs.

### Subscription-through-gateway mode

```bash
./scripts/acpctl.sh onboard claude
```

Choose `subscription` in the wizard when you want Claude Code OAuth upstream with LiteLLM gateway enforcement.

## Modes

| Mode | Default alias | What onboarding prints |
|------|---------------|------------------------|
| API key | `claude-code` | `ANTHROPIC_BASE_URL`, `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` |
| Subscription | `claude-code-max` | `ANTHROPIC_BASE_URL`, `ANTHROPIC_CUSTOM_HEADERS`, `ANTHROPIC_MODEL` |

## Recommended verification flow

```bash
make health
make onboard-claude
```

If you pick subscription mode, launch Claude Code after exporting the printed values and sign in with your Claude subscription when prompted.

## Notes

- The repo does not manage `~/.claude/` for you; apply the printed values in your preferred shell or config-management flow.
- API-key mode is the lowest-friction governed path.
- Subscription mode depends on LiteLLM header forwarding and Claude Code OAuth behavior; keep it aligned with the live repo config.

## References

- [Claude Code testing guide](CLAUDE_CODE_TESTING.md)
- [ACPCTL](ACPCTL.md)
- [LiteLLM Claude Code MAX subscription docs](https://docs.litellm.ai/docs/tutorials/claude_code_max_subscription)
