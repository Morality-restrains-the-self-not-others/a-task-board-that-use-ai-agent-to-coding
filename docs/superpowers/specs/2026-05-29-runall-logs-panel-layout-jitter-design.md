# runAll 日志面板布局抖动 — 设计文档

**日期:** 2026-05-29  
**范围:** `http://localhost:9999/` runAll Web UI — 点击服务行「日志」按钮时的页面布局抖动  
**文件:** `runAll/src/status.html`（embed 静态页）

---

## 问题描述

用户在 runAll 状态页点击任意服务的「日志」按钮后，左侧服务列表区域出现明显布局跳动（内容突然上跳、列宽突变），体验不稳定。

## 根因调查（Phase 1）

Playwright 在 `1400×900` 视口下复现并采集布局指标：

| 时机 | services-pane 宽度 | scrollTop | scrollHeight |
|------|-------------------|-----------|--------------|
| 点击前 | 1352px | 435 | 1215 |
| 点击后（立即） | 1019px | **0** | **1439** |

结论：

1. ** abrupt 分栏切换：** 日志面板由 `display: none` 变为 `display: flex`，同时分隔条出现，服务列表可用宽度一次性减少约 333px。
2. **内容 reflow：** 变窄后服务行内 `actions`、`deps` 等元素换行，`scrollHeight` 从 1215 增至 1439。
3. **滚动位置丢失：** `scrollTop` 从 435 变为 0，用户看到列表「跳到顶部」——这是主观「抖动」的主要来源。
4. **Flex 宽度语义不一致：** CSS 变量 `--logs-width: 38%`，但日志面板实际计算宽度为 `320px`（触达 `min-width`），因 flex 子项默认 `flex-shrink: 1` 且打开时未调用 `setLogsPanelWidth()` 将百分比固化为像素。
5. **刷新周期稳定：** 面板打开后每 2s 的 `renderStatus()` 在宽度不变时不会继续改变列宽；但全量 `innerHTML` 替换在滚动位置未保存时可能加剧跳动（当前宽度稳定时 scroll 已处于 0，问题集中在打开瞬间）。

**非根因（已排除）：** observability bar 异步加载（高度在首屏后稳定）；日志 API 返回内容填充（不改变左栏宽度）。

## 目标

- 点击「日志」时，服务列表**滚动位置保持**（或按 scroll anchoring 平滑过渡）。
- 分栏宽度**一次性、可预期**地到位（38% 或用户上次拖拽宽度），无 flex 挤压至 `min-width` 的跳变。
- 打开/关闭日志面板时**无额外 layout shift**（CLS 友好）。
- 不改动 API 与后端；仅改 `status.html` 内联 CSS/JS。

## 方案

### A. 稳定 flex 分栏（推荐，最小 diff）

1. **CSS**
   - `.logs-panel`、`.pane-divider` 增加 `flex-shrink: 0`（`flex: 0 0 auto`），防止被挤压到 `min-width`。
   - `.services-pane` 增加 `scrollbar-gutter: stable`，避免滚动条出现/消失引起水平微跳。
   - 可选：`.workspace` 在 `.logs-open` 时对 `.services-pane` 使用 `overflow-anchor: auto`（默认），配合 JS 保存滚动。

2. **JS — `openLogsPanel`**
   - 打开前记录 `servicesPane.scrollTop` 与 `scrollHeight`。
   - 添加 `logs-open` 后，用 `setLogsPanelWidth(workspace.clientWidth * 0.38)` 将 `--logs-width` 固化为像素（与拖拽逻辑一致）。
   - 布局稳定后（`requestAnimationFrame` 双帧）按高度比例恢复 `scrollTop`：
     `newScrollTop = savedScrollTop / savedScrollHeight * newScrollHeight`（clamp 到合法范围）。

3. **JS — `renderStatus`**
   - 替换 `innerHTML` 前保存 `scrollTop`；替换后恢复（宽度不变时已验证浏览器通常保留，显式恢复更稳妥）。

4. **JS — `closeLogsPanel`**
   - 同样保存/恢复 scroll，避免关闭时反向跳动。

### B. CSS Grid 三列（备选，改动略大）

`workspace` 改为 `grid-template-columns: 1fr 8px var(--logs-width, 0px)`；关闭时 `--logs-width: 0` 且隐藏 divider，避免 `display: none` 触发的 flex 重算。本需求优先 A，若 A 验收不足再考虑。

## 价值流影响

- **无** `value-stream.yaml` 业务流字段变更。
- 影响 runAll 本地开发体验流（状态页 → 查看服务日志），不涉及 Saas/taskAuth 域。

## 测试计划

1. **Go 静态断言**（`runAll/src/ui_test.go`）：`status.html` 含 `flex-shrink: 0`、`scrollbar-gutter`、scroll 保存/恢复相关 snippet。
2. **Playwright**（`runAll/playwright/tests/`）：新用例 `runall-logs-panel-layout.playwright.test.js`
   - 滚动服务列表至中间 → 点击「日志」→ 断言 `scrollTop > 0`（或相对位置误差 < 10%）。
   - 断言日志面板宽度 ≥ 38% × workspace − tolerance，且打开后 4s 内 services 宽度不变。
3. **手工：** `http://localhost:9999/` 点击「日志」/「Close」，目视无跳动。

## 不在范围

- 日志面板内容虚拟滚动、增量 DOM diff 全量重构。
- 分栏宽度动画（可后续增强）。

## 风险

- 极低：仅前端布局；回滚即还原 `status.html`。

---

**预估工作量：** 1 个文件 + 2 个测试文件，约 1–2 小时。
