# Localize a catalog DESIGN.md into this product

`.ai/04_frontend_development/02_design_md_integration.md` is the policy. This file is the procedure.

## Mapping

| Catalog role | This product |
|--------------|--------------|
| primary / brand accent | `--color-primary` `#7241F2` (紫曜) |
| canvas / background | `--color-background` `#F7F9FC` |
| ink / text | `--color-text` `#1A1A1A` |
| muted | `--color-text-light` `#4B5563` |
| destructive | `--color-accent` `#FF324D` |
| surface / card | white + `.card` |
| radius-control | `rounded-md` (6px) |
| radius-card | `rounded-lg` (8px) |

Keep names **semantic** (`color.primary`, `surface.elevated`). Strip upstream proper nouns (“Linear lavender”, “Geist”, “Binance Yellow”).

## Conflict order

1. Accessibility and this repo's FE meta-rules (links, click guard, traceId).
2. Existing CSS variables / utilities in `taskFE/app/src/css/tailwind.css`.
3. Atmosphere borrowed from a catalog template (density, one-accent discipline).
4. Aesthetic novelty (`frontend-design`) — **last**, and not on product chrome.

## After editing DESIGN.md

1. `python3 db/scripts/ci/check_design_md.py`
2. `python3 db/scripts/ci/test_check_design_md.py`
3. If `colors.primary` changes, update `--color-primary` in the same change.
