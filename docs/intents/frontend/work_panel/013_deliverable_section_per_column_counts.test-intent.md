# 测试意图：交付物分区标题按进度列展示任务数

## 对应功能意图

`013_deliverable_section_per_column_counts.intent.md`

## 用例

| ID | 场景 | 预期 |
|----|------|------|
| T1 | `countTodosByKanbanColumns` 多列任务 | 返回顺序与 statuses 一致，count 正确（含 null 进度归入首列） |
| T2 | statuses 为空 | 返回 `[]` |
| T3 | 分区标题 DOM | `[data-alias="deliverable-section-count"]` 内有多项 `[data-alias="deliverable-section-column-count"]`，而非单一总数 |

## 自动化落点

- 单元：`task2app/front_project/app/src/utils/workPanelKanbanUtils.test.js`（T1/T2）
- 页面级回归（可选）：work-panel CDP 断言各列 count 与 `progress-column-count` 对齐
