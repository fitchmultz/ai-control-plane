# AI Control Plane Leadership Presentation Materials

**AI Control Plane Project**

This directory contains executive-ready presentation materials for the AI Control Plane strategy.

---

## Materials Overview

| File | Format | Purpose | Audience |
|------|--------|---------|----------|
| `ai-control-plane-leadership-deck.md` | Marp (Markdown) | Full 14-slide presentation | Leadership, CTO, Security stakeholders |
| `ai-control-plane-leadership-deck.pdf` | PDF (generated locally) | Rendered presentation output (not committed) | Distribution, offline viewing |
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

### Option 2: Generate PDF from Marp Deck

```bash
# Install Marp CLI (one-time)
npm install -g @marp-team/marp-cli

# Generate PDF
cd docs/presentation
./generate-pdfs.sh

# Or manually:
marp ai-control-plane-leadership-deck.md --pdf --output deck.pdf
```

### Option 3: VS Code Extension

Install the **Marp for VS Code** extension for live preview and PDF export:
- Open `ai-control-plane-leadership-deck.md`
- Click preview icon in top-right
- Use "Export slide deck" command for PDF

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

Edit `ai-control-plane-leadership-deck.md`:
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
2. Deck for discovery calls
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

# PNG (images)
marp deck.md --images
```

---

*Last Updated: March 5, 2026*

*Change requests: open a GitHub issue in this repository.*
