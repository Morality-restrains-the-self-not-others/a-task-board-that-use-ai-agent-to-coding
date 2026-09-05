# runAll 服务列表底部无法滚动 — 设计文档

**日期:** 2026-06-02（2026-07-15 补充：页头滚轮转发）  
**范围:** `http://localhost:9999/` runAll Web UI — 服务列表底部条目滚入视窗  
**文件:** `runAll/src/status.html`（embed 静态页）

---

## 问题描述

用户在 runAll 状态页（`http://localhost:9999/`）无法将**最底部**的服务行滚入可见区域；列表虽已滚动到 `scrollTop` 最大值，末行仍部分或全部处于浏览器视窗下方，无法操作。

**2026-07-15 补充：** 即便布局裁切已修复，用户在页头（observability / 日志收集 / Trace / smoke / 开发工具等 bar）上滚轮时页面无响应，主观为「无法往下滚动」。因 `body { overflow: hidden }`，非 `.services-pane` 区域没有可滚动容器。

## 根因调查

在 `1200×682` 视口下用 Chrome DevTools 复现并采集布局指标：

| 元素 | 指标 | 说明 |
|------|------|------|
| `body` | `overflow: hidden`, `height: 100vh` | 禁止整页滚动 |
| `#observability-bar` | 高度 ~78px | 页头新增，未计入 workspace 高度 |
| `#dev-tools-bar` | 高度 ~45px | 页头新增，未计入 workspace 高度 |
| `.workspace` | `height: calc(100vh - 120px)` | **固定 magic number 120px 已不足** |
| `.workspace` 底边 | 超出视窗 **~89px** | 主内容区整体被裁切 |
| `.services-pane` | `scrollHeight: 2428`, `clientHeight: 562` | 内部滚动正常 |
| 末行（scrollTop=max） | `bottom: 736.6`, 视窗高 `682` | **仍不可见** |

**机制：** `.services-pane` 的滚动容器高度等于 `.workspace` 高度；而 workspace 底边落在视窗之外。滚动到最大值时，内容底边对齐的是 **workspace 元素底边**（视窗外），而非 **浏览器视窗底边**，导致底部约 50–90px 区域永远不可见。

**触发条件：** 页头在 observability bar、dev-tools bar 加入后，实际占用高度约 **~210px**（含 `body` padding、h1、margin），远超 CSS 中假设的 120px。服务数量越多（当前 ~41 个）问题越明显。

**已排除：**

- `renderStatus()` 2s 刷新导致 scroll 丢失（滚动已到 max 仍不可见）
- 日志面板打开时的 scroll 保存逻辑（日志打开/关闭均复现）
- `services-pane` 未设置 `overflow: auto`（已设置，内部滚动存在）

## 目标

- 任意视口高度下，服务列表滚到最底时，**最后一行（含 uptime 行）完全处于可见视窗内**。
- 页头新增/换行（observability、dev-tools 等）时**不再依赖 magic number**，避免回归。
- 日志分栏打开/关闭、面板拖拽宽度时行为不变。
- 仅改 `status.html` 内联 CSS/JS；不改动 API / Go runner。
- **2026-07-15：** 页头滚轮可驱动服务列表；`.services-pane` 滚动条可见。

## 非目标

- 服务行虚拟滚动、移动端响应式重构。
- 修改 `body` 为整页可滚动（会破坏固定分栏 + 日志面板布局）。

---

## 方案对比

### A. Body 纵向 Flex 填充（推荐）

将 `body` 改为纵向 flex 容器，页头块 `flex-shrink: 0`，`.workspace` 使用 `flex: 1; min-height: 0` 占满**剩余视口**，移除 `height: calc(100vh - 120px)`。

```css
body {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  padding: 24px;
}
.observability-bar,
.dev-tools-bar,
h1 { flex-shrink: 0; }
.workspace {
  flex: 1 1 auto;
  min-height: 0;
  margin-top: 20px;
  /* 移除 height: calc(100vh - 120px) */
}
```

| 优点 | 缺点 |
|------|------|
| 自适应任意页头高度，未来加 bar 无需改数字 | 需验证极矮视口（<400px） |
| 与现有 `.services-pane { min-height: 0; overflow: auto }` 语义一致 | 改动 ~10 行 CSS |
| 与 2026-05-29 日志面板 scroll 修复兼容 | |

### B. 增大 magic number（不推荐）

将 `120px` 改为 `220px` 或 `240px`。

| 优点 | 缺点 |
|------|------|
| 一行 CSS | 页头换行/隐藏 observability 时仍会漂移 |
| | 不同 DPI / 字体下可能再次裁切 |

### C. `100dvh` + JS 动态测量页头（备选）

JS 在 load/resize 时测量页头 `offsetHeight`，设置 `--workspace-height`。仅在 A 在极旧浏览器不足时考虑。

### D. 页头滚轮转发到 `.services-pane`（2026-07-15 追加）

保留方案 A，并为 `document` 注册非 passive `wheel`：目标不在其他独立滚动容器（日志/详情）时，将 `deltaY` 写入 `.services-pane.scrollTop`；同时为列表增加显式 scrollbar 样式。

---

## 推荐设计（方案 A + D）

### CSS 变更

1. **`body`**：增加 `display: flex; flex-direction: column;`，保留 `height: 100vh; overflow: hidden; padding: 24px`。
2. **页头元素**（`h1`、`#observability-bar`、`#dev-tools-bar`）：`flex-shrink: 0`（可合并为 `.page-header` 类，但为最小 diff 可用元素选择器或现有 class）。
3. **`.workspace`**：删除 `height: calc(100vh - 120px)`；改为 `flex: 1 1 auto; min-height: 0`（保留现有横向 flex、日志分栏、`--logs-width`）。
4. **`.services-pane`**：保持 `min-height: 0; overflow: auto`；增加 `scrollbar-width` / `::-webkit-scrollbar`。
5. **可选**：收紧 observability / dev-tools bar 的 margin/padding。

### JS 变更

- `findVerticalScrollableAncestor` + `document` `wheel` 转发（见 `status.html` 文末）。
- 现有 `openLogsPanel` / `closeLogsPanel` / `renderStatus` 的 scroll 保存逻辑继续有效。

### 验收标准

| # | 条件 | 通过标准 |
|---|------|----------|
| 1 | 默认视口 `1200×682`，41 个服务 | 滚到底后末行 `getBoundingClientRect().bottom ≤ window.innerHeight` |
| 2 | workspace 底边 | `getBoundingClientRect().bottom ≤ window.innerHeight`（误差 ≤ 2px） |
| 3 | 日志面板打开 | 同上 |
| 4 | 视口 `1400×900` | 同上 |
| 5 | observability bar 加载完成前后 | 无布局跳动导致末行不可达 |
| 6 | 鼠标在 `.page-header` 上滚轮 | `.services-pane.scrollTop` 明显增加 |

---

## 价值流影响

- **无** `value-stream.yaml` 业务字段变更。
- 影响 runAll 本地开发体验流（状态页 → 查看/操作底部服务），与 `runall-observability`、`runall-dev-db-tools` 等 stream 的 UI 入口相关，但不改变 API 语义。
- 测试：新增 Playwright 用例 + 可选 `ui_test.go` 静态断言（body flex、workspace 无 `calc(100vh - 120px)`）。

## 测试计划

1. **Playwright**（`runAll/playwright/tests/runall-services-list-scroll.playwright.test.js`）  
   - 视口 `1200×682`，等待 `#services .service` 渲染  
   - 对 `.services-pane` 执行 `scrollTop = scrollHeight`  
   - 断言最后一个 `.service` 在视窗内（`boundingBox().y + height ≤ viewport height`）  
   - 断言 `#workspace` 底边不超出视窗  
   - 页头 hover + `mouse.wheel` → `scrollTop` 增加  

2. **Go 静态断言**（`runAll/src/ui_test.go`）  
   - `status.html` 含 `flex-direction: column`、`findVerticalScrollableAncestor`、`scrollbar-width: thin`  
   - 不含 `calc(100vh - 120px)`（或 workspace 使用 `flex: 1`）

3. **手工：** 打开 `http://localhost:9999/`，在页头滚轮确认列表跟随；滚至底部确认最后几个服务可点击。

---

## 领域概念清单（轻量）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | runAll 运维 UI | 本地服务编排状态页，无后端领域模型变更 |
| 实体 | ServiceRow（UI） | 列表行，只读展示 runner 状态 |
| 领域事件 | 无 | 纯布局修复 |

---

## 不在范围

- 日志面板内容区独立滚动问题（已有 spec）。
- 服务列表分页或折叠 level。
- 提交/PR（按用户指令另行进行）。
