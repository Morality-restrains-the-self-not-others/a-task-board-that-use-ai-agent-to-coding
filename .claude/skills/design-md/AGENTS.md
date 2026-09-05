# DESIGN.md skill

## Purpose

Make coding agents generate **this product's** UI from repo-root `DESIGN.md` (Google Stitch format), using [awesome-design-md](https://github.com/VoltAgent/awesome-design-md) only as a catalog of **templates to localize**, never as a brand to paste.

## Files

| File | Role |
|------|------|
| `SKILL.md` | Workflow — load this when doing product UI |
| `references/format.md` | Required DESIGN.md sections |
| `references/catalog.md` | Which upstream templates fit which job |
| `references/localize.md` | How to map a template without copying a brand |

## When to apply

- Vue / CSS / HTML in `taskFE` or other product surfaces
- User says DESIGN.md, Stitch, awesome-design-md, or “make it look like Linear/Stripe”
- Visual token, button, card, sidebar, or theme work

## Never do

- Never overwrite `DESIGN.md` with an upstream brand file
- Never invent page-local hex when a token exists
- Never let `frontend-design` override product chrome
