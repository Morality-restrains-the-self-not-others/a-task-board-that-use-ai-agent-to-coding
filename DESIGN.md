---
version: 1.0.0
name: daydaymoney-product-design
description: "Light B2B workbench for task / tenant / billing. Cool gray canvas (#F7F9FC), near-black ink (#1A1A1A), and a single chromatic brand — 紫曜 purple (#7241F2). Accent red (#FF324D) is reserved for destructive / attention, never for primary CTAs. Density is product-app, not marketing cinema. Tokens must stay in CSS variables; do not invent hex per page."
colors:
  primary: "#7241F2"
  primary-dark: "#5C2DC7"
  primary-light: "#E6E0FF"
  on-primary: "#ffffff"
  accent: "#FF324D"
  ink: "#1A1A1A"
  ink-muted: "#4B5563"
  canvas: "#F7F9FC"
  surface: "#ffffff"
  secondary: "#F1F2F6"
  hairline: "#E5E7EB"
  focus-ring: "#A855F7"
  semantic-success: "#16A34A"
  semantic-warning: "#D97706"
  semantic-error: "#FF324D"
typography:
  sans: ui-sans-serif, system-ui, "Segoe UI", sans-serif
  mono: ui-monospace, SFMono-Regular, Menlo, monospace
radius:
  control: 6px
  card: 8px
spacing:
  navbar-h: 74px
  sidebar: 200px
  sidebar-collapsed: 64px
---

# Product DESIGN.md

Visual SSOT for **taskFE product chrome** (work panel, tenant console, billing, auth shells). Google Stitch / [awesome-design-md](https://github.com/VoltAgent/awesome-design-md) format, **localized** to this product. Do not copy Linear / Stripe / Vercel brand names, copy, or proprietary typefaces into the SPA.

Load with skill `.claude/skills/design-md`. Platform compliance (WCAG) still uses `web-design-guidelines`.

## Visual Theme & Atmosphere

- **Mood**: Precise SaaS workbench. Calm canvas, one purple signal, no decorative gradients.
- **Density**: Information-first (task lists, sidebars, forms). Marketing heroes are out of scope for product pages.
- **Philosophy**: Reuse semantic CSS variables from `taskFE/app/src/css/tailwind.css`. A page that introduces a one-off hex has drifted.

## Colors

| Token | Hex | CSS variable | Role |
|-------|-----|--------------|------|
| primary | `#7241F2` | `--color-primary` | Brand, primary CTA, active nav, focus identity |
| primary-dark | `#5C2DC7` | `--color-primary-dark` | Hover / pressed on primary |
| primary-light | `#E6E0FF` | `--color-primary-light` | Selected chip, tinted surfaces |
| accent | `#FF324D` | `--color-accent` | Destructive, unpaid, blocking alerts |
| ink | `#1A1A1A` | `--color-text` | Body and titles |
| ink-muted | `#4B5563` | `--color-text-light` | Secondary labels, idle nav |
| canvas | `#F7F9FC` | `--color-background` | App background |
| secondary | `#F1F2F6` | `--color-secondary` | Secondary buttons, inset wells |
| surface | `#ffffff` | (card `bg-white`) | Elevated panels |

Accent red is **not** a second brand color. Primary CTAs stay purple.

## Typography

- **Family**: Tailwind / system UI sans. Do not switch the product to Inter, Geist, or a display serif to “look like” a catalog site.
- **Body**: 14–16px, regular, line-height ~1.5.
- **Title**: 18–24px, medium/semibold. Page titles are not marketing display (no 80px heroes in the SPA).
- **Mono**: IDs, traces, repo URLs, JSON.

## Components

### Buttons

- Primary: `.btn-primary` — purple fill, white label, `rounded-md`, `px-4 py-2`. Hover darkens to `--color-primary-dark` / `purple-700`.
- Secondary: `.btn-secondary` — `--color-secondary` fill, ink label.
- Destructive: `.btn-accent` — accent fill, white label. Only for delete / irreversible.
- Write-actions: synchronous click guard + `disabled` + `aria-busy` + `Idempotency-Key` (元规则 52). Do not rely on next-tick `disabled` alone.

### Cards & panels

- `.card`: white, `rounded-lg`, `shadow-md`, `p-6`. Hover may lift to `shadow-lg`. No glassmorphism, no gradient borders.

### Inputs

- `.input`: full width, gray-300 border, `rounded-md`, purple focus ring (`ring-2 ring-purple-500/50`). Labels are real `<label>`, not placeholder-only.

### Navigation

- Top bar height `--app-navbar-h` (74px).
- Tenant console sidebar: 200px, collapsed 64px (`.tenant-console-sidebar`).
- Idle links: `--color-text-light`. Active: purple + 2px underline (`.nav-link-active`).
- Real `<a href>` for navigation. Do not `@click.prevent` + `router.push`.

### Modals

- Custom Vue modal components (`{Feature}Modal.vue`). Never `alert` / `confirm` / `prompt`.
- Overlay `rgba(0,0,0,0.5)`. Nested modals stack above parents.

## Layout

- Spacing scale: Tailwind 4px grid (2 / 3 / 4 / 6 / 8).
- Workbench: navbar → optional 200px sidebar → `min-w-0` main.
- Forms: single column on mobile; two-column only when labels stay aligned.
- Whitespace: dense but not cramped; do not add marketing-size vertical padding to CRUD pages.

## Elevation & Depth

- Canvas (0) → card `shadow-md` (1) → hover `shadow-lg` (2) → modal (3).
- No 24px cinematic drop shadows. No inner neon glow.

## Do's and Don'ts

### Do

- Read this file before changing Vue / CSS that users see.
- Map color / radius / type to tokens above or existing utility classes.
- Keep destructive actions visually distinct (accent), primary actions purple.
- Pair color with text / icon (WCAG: never color-only status).
- Put `data-traceId` on request-failure error DOM.

### Don't

- Copy a brand DESIGN.md (Linear lavender, Stripe purple gradient, Vercel Geist, Nike Futura) into the product.
- Invent page-local hex that is not a token.
- Use accent red as the default CTA.
- Use `alert` / intercept link clicks / Teleport without `Teleport-OK`.
- Poll APIs on a timer with no user gesture or SSE.
- Restyle the SPA with `frontend-design` “bold unique aesthetic” — that skill is for one-off artifacts, not product chrome.

## Responsive Behavior

- Breakpoints: Tailwind `sm` 640 / `md` 768 / `lg` 1024.
- Touch targets ≥ 44px on interactive controls.
- Sidebar collapses to 64px; main content `min-w-0` so tables scroll inside, not the page.
- Do not set `user-scalable=no` or `maximum-scale=1`.

## Agent Prompt Guide

Quick tokens: primary `#7241F2`, canvas `#F7F9FC`, ink `#1A1A1A`, accent `#FF324D` (destructive only).

Prompt seed: “Implement this view using repo-root DESIGN.md tokens and existing `.btn-primary` / `.card` / `.input` utilities. Do not introduce new colors or display fonts.”
