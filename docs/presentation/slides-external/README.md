# AI Control Plane - Customer Presentation Deck

**Audience:** Prospective Customers (CISO, CTO, Security Teams)  
**Purpose:** External stakeholder communication of architecture and rollout approach  
**Tone:** External, benefit-focused, professional

> `docs/presentation/slides-external/*.png` are generated customer-facing exports and are **not committed** to source control.
> The canonical source for this deck is `../ai-control-plane-external-deck.md`.

---

## Regenerate the PNG Slide Exports

```bash
cd docs/presentation
marp ai-control-plane-external-deck.md --images --output slides-external/
```

## Build a Static PDF from the Generated PNG Slides

```bash
cd docs/presentation/slides-external
img2pdf *.png -o ../ai-control-plane-external-deck.pdf
```

---

## Slide Inventory (12 slides)

| # | Slide | Focus |
|---|-------|-------|
| 1 | **Title** | Enterprise AI Control Plane |
| 2 | **The AI Governance Gap** | Problem - uncontrolled API and SaaS usage |
| 3 | **The Solution** | Unified control plane overview |
| 4 | **How It Works** | Architecture diagram |
| 5 | **Route-Based Governance** | Enforce routed traffic + detect/respond on bypass |
| 6 | **Service Offerings** | Four service tiers |
| 7 | **Financial Governance** | Cost attribution and chargeback |
| 8 | **Why This Project?** | Differentiation vs competitors |
| 9 | **Implementation Approach** | 4-phase rollout |
| 10 | **Get Started** | Three engagement options |
| 11 | **Closing** | Call to action |
| 12 | **Contact** | Next steps and resources |

---

## Key Messages

1. **Problem:** AI adoption outpacing governance (both API and SaaS)
2. **Solution:** Unified platform with complete visibility
3. **Benefit:** Enforceable controls + detective monitoring + audit trail
4. **Proof:** Productized services with clear deliverables
5. **Differentiation:** Security engineering with enforceable controls and measurable operating evidence

---

## Usage

**PowerPoint/Keynote:**
- Insert → Picture → Select slide PNG
- No additional formatting needed

**Presentation Tips:**
- Focus on business risk (slide 2)
- Emphasize unified approach (slide 3)
- Show clear path to value (slide 9)
- Close with specific next steps (slide 10)

---

## Related Documents

- [Executive One-Pager](../EXECUTIVE_ONE_PAGER.md)
- [Service Offerings](../../SERVICE_OFFERINGS.md)
- [Strategy Document](../../ENTERPRISE_STRATEGY.md)

---

*AI Control Plane Project - Customer Facing*
