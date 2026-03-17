# GitHub Copilot and AI Control Plane

GitHub Copilot is not part of the current `acpctl onboard` supported surface.

## Current truth boundary

This repository does **not** currently ship a first-class governed Copilot path through the AI Control Plane product surface. In particular:

- there is no validated `acpctl onboard copilot` workflow
- there is no supported in-product forward-proxy service in the tracked host-first deployment
- there is no verified budget-enforced or content-inspecting Copilot path in this repo today

## What this means for operators

If you need a governed developer-tool happy path today, prefer one of the supported onboarding targets:

```bash
make onboard-codex
make onboard-claude
make onboard-opencode
make onboard-cursor
```

If you are evaluating Copilot, treat it as a separate governance problem that requires GitHub-native controls plus enterprise network controls outside the validated host-first ACP surface.

## Recommended governance stance for Copilot

Use GitHub-native enterprise controls for:
- seat assignment and entitlement management
- organization policy and audit logging
- data-residency and enterprise settings where available

Use network and endpoint controls for:
- direct egress policy
- proxy enforcement where your environment supports it
- bypass detection and exception management

## Why this doc exists

Buyers and operators ask about Copilot frequently. The honest answer today is that Copilot is adjacent context, not a completed ACP productized onboarding path. Keeping that boundary explicit is better than advertising a broken wizard path.

## References

- [Tooling reference links](TOOLING_REFERENCE_LINKS.md)
- [GitHub Copilot documentation](https://docs.github.com/copilot)
- [GitHub Copilot network settings](https://docs.github.com/copilot/configuring-github-copilot/configuring-network-settings-for-github-copilot)
