# runAll 日志面板悬浮 + 左边缘拖动调整宽度

**日期**: 2026-06-02
**范围**: `runAll/src/status.html`（单文件 CSS + JS）
**类型**: UI 增强 / 布局修复

## 问题描述

当前 runAll Web UI (`http://localhost:9999/`) 中，点击服务行的「日志」按钮后，日志面板与左侧服务列表采用 flex 并排布局。日志面板占用 ~38% 宽度，导致左侧服务列表被挤压变形。

## 设计要求

1. **悬浮 overlay**: 日志面板覆盖在服务列表上方 (`position: absolute`)，不参与 flex 流
2. **左边缘拖动调宽**: 拖拽面板左侧边缘调整宽度，min 320px / max 75vw / 默认 ~38%
3. **始终靠右**: 面板固定在右侧，不可自由移动位置
4. **高度覆盖服务列表区域**: top/bottom 与 `.workspace` 容器对齐，不遮挡顶部工具条
5. **关闭方式**: Close 按钮 / ESC 键（**不响应**点击面板外部）
6. **切换服务保持宽度**: 点击不同服务的「日志」时，面板宽度保持用户调整后的值

## 技术方案

### 唯一方案：`position: absolute` overlay

纯 CSS + JS 改造，Go 后端零改动。

### CSS 变更

| 当前 | 改为 |
|------|------|
| `.workspace` flex 容器 | 添加 `position: relative`（定位锚点） |
| `.logs-panel` flex 子项 `flex: 0 0 var(--logs-width)` | `position: absolute; right: 0; top: 0; bottom: 0; width: var(--logs-width)` |
| `.pane-divider` 在 flex 流中间 | **移除** `.pane-divider`，改为面板左边缘内置拖拽手柄 |
| `.workspace.logs-open` 触发 flex 并排 | 仅控制面板显示/隐藏 + 过渡动画 |
| `.services-pane` 被挤压 | 始终保持全宽 |

### 拖拽手柄

- 位于日志面板左边缘，宽 6px
- 紫色渐变背景，`cursor: ew-resize`
- 中央有视觉指示条（4px × 24px，`#a78bfa`）
- Pointer Events 处理：`pointerdown` → `pointermove` → `pointerup`
- 实时更新面板 `width` 并通过 `setLogsPanelWidth()` 约束范围
- 与现有 `layoutState.resizing` 机制整合，复用 `pane-resizing` body class

### 开关动画

- 打开：面板从右侧滑入 (`transform: translateX(0)` / opacity，~150ms ease)
- 关闭：滑出 (`transform: translateX(100%)` / opacity → 0)，不保留宽度状态
- 使用 CSS `transition` 属性

### 不变的部分

- 日志面板头部（标题、meta、按钮组）
- 2s 自动刷新轮询 (`fetchLogsOnce`)
- 内容渲染 (`renderLogs`, `normalizeLogRows`)
- traceId 提取 → Grafana 跳转
- Loki / Grafana / Copy / Refresh / Close 按钮行为

## 不影响的部分

- Go 后端 `ui.go` 无变更
- `/api/logs`、`/api/status` 等 API 无变更
- 其他 `.html` / `.go` 文件无变更
- 现有测试无需修改（纯 UI 布局调整，不改变逻辑）

## 价值流影响

无。这是 runAll 开发工具自身的 UI 布局改进，不触及任何产品价值流 (`value-stream.yaml`)。

## 测试策略

- 手动验证：打开 `http://localhost:9999/`，点击服务「日志」按钮，确认面板悬浮 + 可拖动
- Playwright E2E（如需要）：`playwright/front_project/tests/` 下新增布局验证用例
