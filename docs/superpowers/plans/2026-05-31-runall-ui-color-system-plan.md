# runAll Web UI 色彩系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 runAll `:9999` 状态页建立 CSS 设计令牌并强化按钮点击/按下视觉反馈，修复 `flashButtonClick` 引用错误。

**Architecture:** 单文件 `status.html` 内 `:root` 变量定义色彩系统；`.action-btn` / `.logs-panel-btn` 共享状态机；语义 variant 覆盖 accent；`pulseClickFeedback` 驱动 `.is-clicked` 脉冲。无 DDD/后端变更。

**Tech Stack:** 嵌入式 HTML/CSS/JS；Go `go:embed` + `ui_test.go` 片段断言

---

## File Map

| File | Responsibility |
|------|----------------|
| `runAll/src/status.html` | CSS 令牌、按钮状态、JS 反馈时长 |
| `runAll/src/ui_test.go` | 锁定色彩系统关键片段 |
| `docs/superpowers/specs/2026-05-31-runall-ui-color-system-design.md` | 设计规格（已完成） |

---

### Task 1: CSS 设计令牌与默认按钮状态

**Files:**
- Modify: `runAll/src/status.html`

- [x] **Step 1:** 在 `<style>` 顶部添加 `:root { --ra-* }` 变量块
- [x] **Step 2:** `.action-btn` / `.logs-panel-btn` 使用变量；`:active` / `.is-clicked` 改背景+边框+发光（移除纯 brightness）
- [x] **Step 3:** `clickFeedbackDurationMs = 300`

**Verify:** 目视 localhost:9999 或 grep `--ra-btn-bg-clicked`

---

### Task 2: 语义 Variant accent

**Files:**
- Modify: `runAll/src/status.html`

- [x] **Step 1:** `.restart-btn` 蓝、`.logs-btn` 紫、`.clear-logs-btn` / `.obs-clear-all-btn` 橙 `:active` + `.is-clicked`
- [x] **Step 2:** disabled 状态 suppress transform/glow

**Verify:** 点击重启/日志/清空按钮可见对应 accent

---

### Task 3: Bugfix + 回归测试

**Files:**
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`

- [x] **Step 1:** `flashButtonClick` → `pulseClickFeedback`
- [x] **Step 2:** 新增 `TestUIHomePage_ColorSystemPresent` 断言关键片段
- [x] **Step 3:** `cd runAll/src && go test -run TestUIHomePage -count=1`

**Required snippets for test:**
- `runAll UI Color System`
- `--ra-btn-bg-clicked`
- `.action-btn.is-clicked`
- `.restart-btn.is-clicked`
- `clickFeedbackDurationMs = 300`
- `pulseClickFeedback(event.currentTarget)`

---

### Task 4: 验收

- [x] `go test -run TestUIHomePage` in `runAll/src` 全绿
- [x] 设计文档验收清单（片段测试覆盖 CSS/JS 契约）
