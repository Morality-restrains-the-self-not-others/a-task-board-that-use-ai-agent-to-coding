# 测试意图：交付物过滤栏下拉展示任务编号

## 对应功能意图

`016_deliverable_filter_dropdown_task_no.intent.md`

## 测试目标

确认过滤栏与上层交付物下拉的 option 文案带人读任务编号，且 value 仍为任务 id。

## 测试分层

| 层 | 落点 | 覆盖 |
|----|------|------|
| 单元 | `workPanelDeliverableContent.test.js` | `#N 标题` 格式化 |
| 单元 | `workPanelDeliverableAggregation.test.js` | 内容对象携带 `workspace_seq` |
| 组件 | `DeliverableBreadcrumb.test.js` | option 文案 |
| 组件 | `CreateTaskParentDeliverableField.test.js` | 上层交付物 option 文案 |

## 用例矩阵

| ID | 场景 | 预期 |
|----|------|------|
| T1 | 内容有 workspace_seq=12、标题 hello | option / label 为 `#12 hello` |
| T2 | 内容无序号 | option 为标题，不含 `#` |
| T3 | 同标题不同序号 | 两条 option 文案可区分，value 仍为各自 id |
| T4 | 上层交付物候选有序号 | option 为 `#N 标题` |

## 数据与环境

- 不依赖真实账号；Vue 组件用 `@vue/test-utils` mount。
- todos 夹具显式带 `workspace_seq`。

## 通过标准

上述单测全绿；过滤仍按 content.id 工作。
