# Fix `// 500-line rule exception` Comment Leak in Rendered HTML

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop JavaScript-style `// 500-line rule exception` comments from rendering as visible text on the task-detail page.

**Architecture:** Two Vue SFC files had `//` comments inside `<template>` blocks, which Vue's template compiler treats as raw text nodes. The fix converts them to HTML comments (`<!-- ... -->`), which browsers strip from rendering.

**Tech Stack:** Vue 3 SFC

---

### Task 1: Fix ServerConfigHardwarePanel.vue template comment

**Files:**
- Modify: `task2app/front_project/app/src/components/ServerConfigHardwarePanel.vue:2`

- [x] **Step 1: Convert `//` comment to HTML comment**

Line 2 changed from:
```html
// 500-line rule exception: ServerConfigHardwarePanel is 2062 lines. See taskDetailCloneProgress.js for split progress.
```
To:
```html
<!-- 500-line rule exception: ServerConfigHardwarePanel is 2062 lines. See taskDetailCloneProgress.js for split progress. -->
```

- [x] **Step 2: Verify the change**

Run: `head -4 task2app/front_project/app/src/components/ServerConfigHardwarePanel.vue`
Expected: Line 2 starts with `<!--`, not `//`

---

### Task 2: Fix ServerConfig.logic.vue template comment

**Files:**
- Modify: `task2app/front_project/app/src/components/ServerConfig.logic.vue:2`

- [x] **Step 1: Convert `//` comment to HTML comment**

Line 2 changed from:
```html
// 500-line rule exception: ServerConfig.logic is 1941 lines. See taskDetailCloneProgress.js for split progress.
```
To:
```html
<!-- 500-line rule exception: ServerConfig.logic is 1941 lines. See taskDetailCloneProgress.js for split progress. -->
```

- [x] **Step 2: Verify the change**

Run: `head -4 task2app/front_project/app/src/components/ServerConfig.logic.vue`
Expected: Line 2 starts with `<!--`, not `//`

---

### Task 3: Confirm no other `//` leaks in template blocks

**Files:**
- (none modified)

- [x] **Step 1: Scan for `//` comments immediately after `<template>` tags**

Run: `grep -A1 "^<template>" task2app/front_project/app/src/**/*.vue | grep "// "`
Expected: No output (all leaks fixed)

---

### Task 4: Build verification (if frontend build tooling is available)

- [ ] **Step 1: Build the frontend project**

Run: `cd task2app/front_project/app && npm run build 2>&1 | tail -20`
Expected: Build succeeds without errors
