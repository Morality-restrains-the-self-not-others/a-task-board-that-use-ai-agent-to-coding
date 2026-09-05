# runAll 日志内容区滚动裁切修复 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 修复 `:9999` 日志面板 `#logs-content` 滚到底时末行被裁切的问题。

**Architecture:** 单文件 `status.html` flex 约束修复；Playwright 布局断言 + Go 片段测试。无 DDD/后端变更。

**Tech Stack:** 嵌入式 HTML/CSS；Go `go:embed` + Playwright

**DDD:** 跳过 — 无领域模型变更（见 NFR 澄清文档）。

---

## File Map

| File | Responsibility |
|------|----------------|
| `runAll/src/status.html` | `.logs-panel-header` / `.logs-content` CSS |
| `runAll/src/ui_test.go` | 锁定 flex CSS 片段 |
| `runAll/playwright/tests/runall-logs-content-scroll.playwright.test.js` | 末行可见 E2E |

---

### Task 1: CSS flex 约束修复

**Files:**
- Modify: `runAll/src/status.html`

- [x] **Step 1:** `.logs-panel-header` 增加 `flex-shrink: 0`
- [x] **Step 2:** `.logs-content` 将 `min-height: 280px` 改为 `min-height: 0`

**Verify:** Playwright 诊断脚本 `lastLineVisible: true`

---

### Task 2: Go 片段回归测试

**Files:**
- Modify: `runAll/src/ui_test.go`

- [x] **Step 1:** 新增 `TestUIHomePage_LogsContentScrollLayout` 断言：
  - `.logs-content` 含 `min-height: 0`
  - 不含 `min-height: 280px`
  - `.logs-panel-header` 含 `flex-shrink: 0`

**Verify:** `cd runAll/src && go test -run TestUIHomePage_LogsContentScrollLayout -count=1`

---

### Task 3: Playwright E2E

**Files:**
- Create: `runAll/playwright/tests/runall-logs-content-scroll.playwright.test.js`

- [x] **Step 1:** 视口 `1200×682`，点击首个 `.logs-btn`
- [x] **Step 2:** 等待日志加载，`#logs-content` 滚到 max
- [x] **Step 3:** 断言末行 bottom ≤ panel bottom；content bottom ≤ panel bottom

**Verify:** `cd runAll/playwright && npx playwright test runall-logs-content-scroll`

---

### Task 4: 验收

- [x] `go test -run TestUIHomePage ./...` in `runAll/src`
- [x] Playwright 用例全绿
