# 测试意图：工作面板顶栏合并为单行工具栏

## 对应功能意图

`docs/intents/frontend/work_panel/work_panel_header_single_toolbar.intent.md`

## 用例

| ID | 场景 | 前置 | 步骤 | 期望 |
|----|------|------|------|------|
| T1 | 单行工具栏结构 | 挂载 `WorkPanelHeader` | 查询 `header-toolbar-row` | 含 `flex`+`flex-wrap`；标题区与摘要区为直接子节点；摘要区无 `mt-2`/`w-full` |
| T2 | 右侧操作区 | 有 `tenantId` | 查询 `header-actions-row` | 含 `ml-auto`；内含工作空间切换与「创建任务」 |
| T3 | 搜索仍在标题与工作空间之间 | 有 `tenantId` | 比较 toolbar HTML 序 | `工作面板` < `work-panel-task-search` < `header-workspace-switcher-row` |
| T4 | 无租户不渲染搜索 | `tenantId` 为空 | 查询搜索 | 不存在 `work-panel-task-search` |
| T5 | 标题区不含机器节点文案 | 有 machineSummary | 读 `header-title-row` 文本 | 不含「机器节点」；摘要区含机器摘要与创建任务 |
| T6 | 工作空间失败 traceId | WorkspaceSwitcher 加载失败 | 读 `header-title-row` | 挂载该次请求 `data-traceId` |
| T7 | 机器摘要失败 traceId | `machineSummaryErrorTraceId` 非空 | 读 `header-summary-row` | 挂载该 traceId |
| T8 | 自动调度 x 序 | E2E 工作面板 | 量 boundingBox | 搜索 < 自动调度安排 < 工作空间选择器，且与搜索同一行 |

## 可执行测试

- `taskFE/app/src/views/WorkPanelHeader.legend.test.js`
- `taskFE/app/src/views/WorkPanelHeader.traceId.test.js`
- `taskFE/tests/WorkPanel.auto-schedule-link.playwright.test.js`
