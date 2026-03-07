# AI Control Plane Presentation Guide

This guide describes how to package and deliver presentation materials from this repository.

## Available Slide Assets

### External PNG Deck (Customer-facing, generated on demand)
- **Location:** `docs/presentation/slides-external/`
- **Format:** 12 PNG slide exports
- **Use for:** customer demos, external briefings, and static deck export
- **Source of truth:** `docs/presentation/ai-control-plane-external-deck.md`
- **Commit policy:** Generated locally for delivery/distribution; do not commit PNG exports

### Marp Source Decks
- **Leadership/Internal:** `docs/presentation/ai-control-plane-leadership-deck.md`
- **Customer/External:** `docs/presentation/ai-control-plane-external-deck.md`
- **Use for:** editable narrative updates and PDF/PPTX/PNG generation

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

### Build PNG slides for customer-facing export

```bash
cd docs/presentation
marp ai-control-plane-external-deck.md --images --output slides-external/
```

### Build a PDF directly from PNG slides

```bash
cd docs/presentation
marp ai-control-plane-external-deck.md --images --output slides-external/
cd slides-external
img2pdf *.png -o ../ai-control-plane-external-deck.pdf
```

## Presentation Flow (15–20 minutes)

1. Problem framing
2. Architecture and governance approach
3. Service offerings and operational model
4. Validation evidence and known limitations for leadership/internal audiences
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
