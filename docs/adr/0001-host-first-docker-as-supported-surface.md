# 0001: Host-first Docker as the supported primary surface

- Status: Accepted
- Date: 2026-03-16

## Context

The repository contains multiple technical tracks, but only one primary validated operator story.

## Decision

The supported primary surface is the host-first Docker reference implementation with:

- `make` as the main operator interface
- `acpctl` as the typed workflow engine
- explicit overlays for TLS, offline, DLP, and managed UI
- incubating Helm and Terraform retained outside the primary support claim

## Consequences

- Public docs must lead with the host-first Docker path.
- Incubating deployment assets must stay clearly marked and out of the default operator UX.
- Validation and release discipline must reinforce this claim boundary.

## Alternatives Considered

- Leading with Kubernetes-first positioning before validation existed
- Mixing incubating cloud assets into the default operator workflow
