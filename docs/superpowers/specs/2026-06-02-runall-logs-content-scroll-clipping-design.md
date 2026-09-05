# runAll 日志内容区无法滚到底 — 设计文档

**日期:** 2026-06-02  
**范围:** `http://localhost:9999/` runAll Web UI — 日志面板 `#logs-content` 滚动裁切  
**文件:** `runAll/src/status.html`（embed 静态页）

---

## 问题描述

用户在 runAll 状态页打开任意服务的「日志」面板后，无法将日志内容滚到最底部；滚到 `scrollTop` 最大值时，最后一行仍部分不可见。

## 根因调查

Playwright 在 `1200×682` 视口下复现并采集布局指标：

| 元素 | 指标 | 说明 |
|------|------|------|
| `.logs-panel` | 高 ~431px，`overflow: hidden` | 面板在视窗内 |
| `.logs-panel-header` | 高 ~174px | 多按钮换行，占用偏大 |
| `.logs-content` | `min-height: 280px`，`flex: 1 1 auto`，**无 `min-height: 0`** | 可用高度仅 ~257px，却被强制 ≥280px |
| 滚到 max | `lastLineVisible: false`，裁切 ~8.5px | 内容区底边超出面板底边 ~22px |

**机制：** header 较高时，`.logs-content` 的 `min-height: 280px` 大于 flex 剩余空间，元素溢出 `.logs-panel` 并被 `overflow: hidden` 裁切。内部滚动已到最大值，最后一行仍落在裁切区。

**已排除：**

- 日志 API 返回不完整（`scrollHeight` 正常，问题在 CSS）
- workspace 整体裁切（2026-06-02 服务列表 flex 修复后 workspace 底边在视窗内）
- `renderStatus()` 2s 刷新导致 scroll 丢失（滚到 max 仍不可见）

## 目标

- 打开日志面板后，滚到最底时**最后一行完全可见**（`lastLine.bottom ≤ panel.bottom`）。
- header 按钮换行、面板拖拽变窄时仍成立。
- 仅改 `status.html` 内联 CSS；不改动 API / Go runner。

## 非目标

- 日志 auto-refresh 吸底（可选后续增强）。
- header 按钮单行布局优化。
- 日志虚拟滚动或 DOM diff 重构。

---

## 方案对比

### A. 修正日志内容区 Flex 约束（推荐）

```css
.logs-panel-header { flex-shrink: 0; }
.logs-content {
  flex: 1 1 auto;
  min-height: 0;   /* 替换 min-height: 280px */
  overflow: auto;
}
```

| 优点 | 缺点 |
|------|------|
| ~2 行 CSS，与 `.services-pane` 模式一致 | 极矮视口下面板可能很矮 |
| 自适应 header 换行 | |

### B. 增大 magic number（不推荐）

将 `min-height: 280px` 改为更小固定值 — header 高度变化时仍会失效。

### C. 日志面板 `position: fixed`（不推荐）

改动大，与分栏拖拽冲突。

---

## 推荐设计（方案 A）

### CSS 变更

1. `.logs-panel-header` → `flex-shrink: 0`
2. `.logs-content` → 删除 `min-height: 280px`，改为 `min-height: 0`

### JS 变更

无必须变更。

### 验收标准

| # | 条件 | 通过标准 |
|---|------|----------|
| 1 | 打开任意服务日志，`1200×682` | `#logs-content` 滚到 max 后最后一行在面板内 |
| 2 | 同上 | `#logs-content.getBoundingClientRect().bottom ≤ #logs-panel.bottom + 1px` |
| 3 | 窄面板 header 换行 | 仍满足 1、2 |

### 测试计划

1. **Playwright** `runall-logs-content-scroll.playwright.test.js` — 断言末行可见、内容区不溢出面板
2. **Go 静态断言** `ui_test.go` — `.logs-content` 含 `min-height: 0`，不含 `min-height: 280px`

---

## 价值流影响

- **无** `value-stream.yaml` 业务字段变更。
- 影响 runAll 本地开发体验流（状态页 → 查看服务日志），不改变 API 语义。

## 领域概念清单（轻量）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | runAll 运维 UI | 纯布局修复，无后端领域模型变更 |
| 实体 | LogPanelView（UI） | 日志展示区域，只读 |
| 领域事件 | 无 | |

---

## 不在范围

- 服务列表底部裁切（已有独立 spec `2026-06-02-runall-services-list-scroll-clipping-design.md`）。
- 日志面板打开时服务列表抖动（已有 `2026-05-29-runall-logs-panel-layout-jitter-design.md`）。
