# AI Control Plane Presentation Guide

This guide describes how to package and deliver presentation materials from this repository.

## Available Slide Assets

### External PNG Deck (Customer-facing)
- **Location:** `docs/presentation/slides-external/`
- **Format:** 12 PNG slides
- **Use for:** customer demos, leadership briefings, and static deck export

### Marp Source Deck
- **Location:** `docs/presentation/ai-control-plane-leadership-deck.md`
- **Use for:** editable narrative updates and PDF/PPTX generation

### Executive One-Pager
- **Files:**
  - `docs/presentation/EXECUTIVE_ONE_PAGER.md`
  - `docs/presentation/executive-one-pager.html`

## Quick Usage

### Build a PDF from the Marp deck

```bash
cd docs/presentation
./generate-pdfs.sh
```

### Build a PDF directly from PNG slides

```bash
cd docs/presentation/slides-external
img2pdf *.png -o ../ai-control-plane-strategy-external.pdf
```

## Presentation Flow (15–20 minutes)

1. Problem framing
2. Architecture and governance approach
3. Service offerings and operational model
4. Validation evidence and known limitations
5. Decision/next-step ask

## Messaging Guardrails

### Do say
- Gateway-routed traffic is enforceable; bypass requires detection + response
- Controls are evidence-backed and runbook-driven
- Known limitations are documented transparently

### Do not say
- "We block all AI usage"
- "Guaranteed compliance certification by default"
- "All prompts are stored by default"

## Pre-Presentation Checklist

- [ ] `make ci` passes
- [ ] Demo environment health check passes (`make health` or `make health-offline`)
- [ ] Readiness tracker reviewed (`docs/release/PRESENTATION_READINESS_TRACKER.md`)
- [ ] Known limitations reviewed (`docs/KNOWN_LIMITATIONS.md`)
- [ ] One-pager exported and reviewed for date/content freshness

## References

- [EXECUTIVE_ONE_PAGER.md](./EXECUTIVE_ONE_PAGER.md)
- [../SERVICE_OFFERINGS.md](../SERVICE_OFFERINGS.md)
- [../release/PRESENTATION_READINESS_TRACKER.md](../release/PRESENTATION_READINESS_TRACKER.md)
