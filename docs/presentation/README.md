# AI Control Plane Leadership Presentation Materials

**AI Control Plane Project**

This directory contains presentation materials for leadership/internal and customer-facing AI Control Plane briefings.

> Canonical source decks in this repository:
> - `ai-control-plane-leadership-deck.md` for leadership/internal audiences
> - `ai-control-plane-external-deck.md` for customer-facing audiences
>
> Generated PDF and PNG exports are local build artifacts and are **not committed**.

---

## Materials Overview

| File | Format | Purpose | Audience |
|------|--------|---------|----------|
| `ai-control-plane-leadership-deck.md` | Marp (Markdown) | Full 14-slide leadership/internal source deck | Leadership, CTO, Security stakeholders |
| `ai-control-plane-external-deck.md` | Marp (Markdown) | 12-slide customer-facing source deck | Prospective customers, external briefings |
| `ai-control-plane-leadership-deck.pdf` | PDF (generated locally) | Leadership deck render (not committed) | Distribution, offline viewing |
| `ai-control-plane-external-deck.pdf` | PDF (generated locally) | Customer-facing deck render (not committed) | Distribution, offline viewing |
| `slides-external/README.md` | Markdown | External slide export guide and inventory | Presenters, maintainers |
| `slides-external/*.png` | PNG (generated locally) | Customer-facing slide image exports (not committed) | PowerPoint/Keynote import, static export |
| `EXECUTIVE_ONE_PAGER.md` | Markdown | Text-only executive summary | Quick reference, version control |
| `executive-one-pager.html` | HTML | Styled one-page summary | Print to PDF, web viewing |
| `generate-pdfs.sh` | Shell script | PDF generation automation | Maintainers |

---

## Quick Start

### Option 1: View/Print HTML One-Pager (Recommended)

```bash
# Open in browser, then Print → Save as PDF
open docs/presentation/executive-one-pager.html
```

**Print Settings:**
- Enable "Background Graphics" 
- Set margins to "Minimum" or "None"
- Use A4 or Letter size

### Option 2: Generate PDFs from Marp Decks

```bash
# Install Marp CLI (one-time)
npm install -g @marp-team/marp-cli

# Generate PDF
cd docs/presentation
./generate-pdfs.sh

# Or manually:
marp ai-control-plane-leadership-deck.md --pdf --output deck.pdf
```

### Option 3: Generate Customer-Facing PNG Slides

```bash
cd docs/presentation
marp ai-control-plane-external-deck.md --images --output slides-external/
```

These PNGs are generated delivery assets for external presentations and should not be committed to git.

### Option 4: VS Code Extension

Install the **Marp for VS Code** extension for live preview and export:
- Open `ai-control-plane-leadership-deck.md` or `ai-control-plane-external-deck.md`
- Click preview icon in top-right
- Use "Export slide deck" for PDF or image output

---

## Slide Deck Structure (14 Slides)

1. **Title** — Enterprise AI Control Plane
2. **Problem** — Two AI adoption channels, coverage gaps
3. **Solution** — Unified control plane overview
4. **Governance Model** — Route-based (enforce routed traffic, detect/respond on bypass)
5. **Architecture** — Visual system diagram
6. **Evidence** — Demonstrable proof points
7. **Capabilities** — Honest enforce vs detect matrix
8. **Services** — Four productized offerings
9. **Boundaries** — Clear customer/Project responsibilities
10. **Financial** — Billing models and chargeback
11. **Roadmap** — Four-phase implementation
12. **Readiness** — 8/8 gates PASS status
13. **Decisions** — Four leadership approvals needed
14. **Closing** — Call to action

## Customer-Facing Deck Structure (12 Slides)

1. **Title** — Enterprise AI Control Plane
2. **The AI Governance Gap** — Uncontrolled API and SaaS usage
3. **The Solution** — Unified control plane overview
4. **How It Works** — Architecture diagram
5. **Route-Based Governance** — Enforce routed traffic and detect/respond on bypass
6. **Service Offerings** — Four service tiers
7. **Financial Governance** — Cost attribution and chargeback
8. **Why This Project?** — Differentiation and operating model
9. **Implementation Approach** — Four-phase rollout
10. **Get Started** — Engagement options
11. **Closing** — Call to action
12. **Contact** — Next steps and references

---

## Brand Compliance

All materials follow the Project visual identity used throughout this repository:

| Element | Specification |
|---------|--------------|
| **Primary Colors** | Orange `#f26522`, Black `#000000`, White `#ffffff` |
| **Background** | Dark (`#111111` to `#1a1a1a`) with light text |
| **Typography** | Inter font family, weights 400-700 |
| **Accent Usage** | Orange for emphasis, CTAs, headers |
| **Tables** | Orange headers, alternating row backgrounds |

---

## Content Sources

Materials synthesized from canonical documentation:

| Source Document | Key Content Used |
|----------------|------------------|
| `../ENTERPRISE_STRATEGY.md` | Strategy, architecture, roadmap |
| `../SERVICE_OFFERINGS.md` | Four service offerings, RACI, boundaries |
| `../release/PRESENTATION_READINESS_TRACKER.md` | Canonical readiness gates and evidence pointers |
| `../GO_TO_MARKET_SCOPE.md` | Scope and readiness criteria |
| `EXECUTIVE_ONE_PAGER.md` | Canonical executive narrative and decision framing |

---

## Customization Guide

### Changing the Date

```bash
# Update all files
sed -i '' 's/February 2026/[NEW DATE]/g' *.md *.html
```

### Modifying Service Offerings

Edit the applicable source deck:
```markdown
| Offering | Duration | Primary Outcome |
|----------|----------|-----------------|
| **Your New Offering** | X weeks | Outcome description |
```

### Updating Readiness References

Refresh the readiness reference text in `executive-one-pager.html` so it points to the current generated readiness run:
```html
<p>Latest published readiness run: <code>readiness-YYYYMMDDTHHMMSSZ</code></p>
```

---

## Distribution Recommendations

### For C-Suite Pre-Read
1. Print `executive-one-pager.html` to PDF
2. Include with meeting invite
3. Link to full deck for technical deep-dive

### For Board Presentation
1. Use a locally generated `ai-control-plane-leadership-deck.pdf` (16:9 format)
2. Use `EXECUTIVE_ONE_PAGER.md` + `PRESENTATION_GUIDE.md` for narrative notes and presenter flow
3. Have `SERVICE_OFFERINGS.md` ready for detailed questions

### For Sales Enablement
1. One-pager as leave-behind
2. Customer-facing deck for discovery calls and static slide exports
3. Link to demo scripts in `../demo/`

---

## Pre-Presentation Checklist

Before presenting to leadership:

- [ ] Verify readiness materials point to the current generated readiness run
- [ ] Confirm `make ci` passes in current checkout
- [ ] Review the narrative framing in `EXECUTIVE_ONE_PAGER.md`
- [ ] Confirm capability claims align with `docs/KNOWN_LIMITATIONS.md`
- [ ] Update date if needed
- [ ] Test PDF export prints correctly
- [ ] Have evidence commands ready for demo

---

## Troubleshooting

### Marp CLI Not Found
```bash
npm install -g @marp-team/marp-cli
# Or use npx:
npx @marp-team/marp-cli deck.md --pdf
```

### PDF Export Missing Backgrounds
In Chrome Print dialog:
- Check "Background Graphics" option
- Set margins to "None"

### Fonts Not Rendering
Ensure Inter font is available or use system fallback:
```css
font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
```

---

## Related Materials

| Document | Location |
|----------|----------|
| Full Strategy | `../ENTERPRISE_STRATEGY.md` |
| Service Catalog | `../SERVICE_OFFERINGS.md` |
| Readiness Tracker | `../release/PRESENTATION_READINESS_TRACKER.md` |
| Demo Scripts | `../../demo/README.md` |
| Executive One-Pager | `EXECUTIVE_ONE_PAGER.md` |

---

## Marp Development

### Live Preview
```bash
marp -p ai-control-plane-leadership-deck.md
marp -p ai-control-plane-external-deck.md
```

### Theme Customization
Edit CSS in the `<style>` block of the Marp document header.

### Export Formats
```bash
# PDF (presentation)
marp deck.md --pdf

# HTML (web)
marp deck.md --html

# PPTX (PowerPoint)
marp deck.md --pptx

# PNG (customer-facing images, generated locally and not committed)
marp deck.md --images
```

---

*Last Updated: March 5, 2026*

*Change requests: open a GitHub issue in this repository.*
