# DESIGN.md format (Stitch + this repo)

Upstream spec: [Google Stitch DESIGN.md](https://stitch.withgoogle.com/docs/design-md/specification/). Catalog examples: [VoltAgent/awesome-design-md](https://github.com/VoltAgent/awesome-design-md).

Each catalog site ships `DESIGN.md` + `preview.html` + `preview-dark.html`. We keep **one** product `DESIGN.md` at repo root. Previews are optional (see OPT).

## Required YAML

```yaml
---
name: daydaymoney-product-design
description: "one-paragraph atmosphere + token story"
colors:
  primary: "#7241F2"   # must match taskFE --color-primary
---
```

## Required markdown sections

| Section | Capture |
|---------|---------|
| Visual Theme & Atmosphere | Mood, density, philosophy |
| Colors | Semantic name + hex + CSS variable + role |
| Typography | Family + hierarchy; **system UI for product SPA** |
| Components | Buttons, cards, inputs, nav, modals + states |
| Layout | Spacing, navbar/sidebar, whitespace |
| Elevation | Shadow steps |
| Do's and Don'ts | Guardrails including this repo's FE meta-rules |
| Responsive | Breakpoints, 44px targets |
| Agent Prompt Guide | Copy-paste token seed |

CI (`check_design_md.py`) enforces: file exists, those headings, `colors.primary` ↔ `--color-primary`.
