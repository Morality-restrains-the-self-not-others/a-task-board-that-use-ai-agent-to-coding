---
name: design-md
description: Use the repo-root DESIGN.md as the visual SSOT when building or restyling product UI (Vue/SPA, dashboards, forms, buttons, cards, tokens, colors, typography). Also use when the user mentions DESIGN.md, Google Stitch, awesome-design-md, VoltAgent design-md, or asks to make a page look like Linear/Stripe/Vercel/Notion — localize into this product's DESIGN.md instead of copying a brand. Do not use for one-off slides/HTML artifacts outside the product chrome (use frontend-design / theme-factory).
---

# DESIGN.md — product visual baseline

[VoltAgent/awesome-design-md](https://github.com/VoltAgent/awesome-design-md) is a **catalog of DESIGN.md files**, not Agent Skills. What we take: Stitch markdown format + “drop a DESIGN.md, then generate UI”. What we do **not** take: 73 brand identities copied into the SPA.

## Load order (mandatory for product UI)

1. Read repo-root [`DESIGN.md`](../../../DESIGN.md) (and [`DESIGN.md.ai.md`](../../../DESIGN.md.ai.md)).
2. Apply tokens; do not invent hex / display fonts.
3. Then load `.claude/skills/web-design-guidelines` for WCAG / responsive.
4. Then apply `.ai/04_frontend_development/` constraints (real `<a href>`, click guard, `data-traceId`, no Teleport, no poll).
5. `frontend-design` is **out of scope** for taskFE chrome — it fights this baseline (it tells agents to avoid system fonts and purple-on-white).

## When the user wants “looks like X”

1. Do **not** copy `design-md/<brand>/DESIGN.md` into the repo root.
2. Read [references/catalog.md](references/catalog.md) and pick the closest **template family** (density, dark/light, accent count).
3. Localize per [references/localize.md](references/localize.md): map roles onto existing CSS variables; keep 紫曜 `#7241F2`.
4. If the change is a new visual language, edit `DESIGN.md` first, then components. Gate: `python3 db/scripts/ci/check_design_md.py`.

## Format

Stitch sections we require in `DESIGN.md`: Colors, Typography, Components, Layout, Do's and Don'ts. See [references/format.md](references/format.md).

## Do not

- Vendor the awesome-design-md tree (copyright + identity drift).
- Use accent `#FF324D` as the default CTA.
- Restyle billing / KYC / payment surfaces as “playful marketing”.

总结清单：
- 产品 SPA: 读根目录 DESIGN.md、用 CSS 变量、再套 WCAG
- “像某品牌”: 只借结构气质，禁止覆盖本产品令牌
- 一次性海报/HTML artifact: 走 frontend-design，不走本技能
