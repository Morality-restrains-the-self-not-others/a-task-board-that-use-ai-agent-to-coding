# 工作面板：进度状态变更后在同交付物横轴内纵向移动

- **日期**: 2026-07-15
- **状态**: approved（goal-mode 自动采用）
- **范围**: `task2app/front_project` 工作面板看板 UI；不新增后端 HTTP 接口
- **架构影响**: 无（纯前端展示；复用既有 `progress_column_id` / `deliverable_obj_id` PATCH）

## 1. 问题与目标

当前看板每个交付物纵列仅一个任务区（`deliverable-single-lane`），卡片上改「进度状态」只 PATCH 数据，**视觉上不换位**。

目标（对齐 2026-07-13 设计方案 B）：

1. **横轴** = 交付物类别分栏
2. **纵轴** = 进度列（`.progress-lane` 自上而下）
3. 卡片下拉修改进度状态 → 自动移入**同一交付物分栏**内对应进度纵轴
4. 栏内跨进度拖拽 → PATCH `progress_column_id`；跨交付物分栏拖拽 → 另写 `deliverable_obj_id`

## 2. 方案选型

| 方案 | 结论 |
|------|------|
| A. 仅乐观更新本地数组、不恢复纵轴 UI | 否 — 用户明确要求「不同纵轴移动」 |
| B. 恢复交付物栏内纵向进度分区 | ⭐ 采用 — 与既有设计/CSS/Playwright 选择器一致 |
| C. 改回纯进度横向看板 | 否 — 会丢掉交付物横轴聚合 |

## 3. 交互

| 操作 | 效果 |
|------|------|
| 改卡片「进度状态」下拉 | PATCH `progress_column_id` + `order`；`tasks-updated` → 刷新后卡片出现在同交付物栏的目标 `.progress-lane` |
| 同栏跨进度拖拽 | PATCH `progress_column_id` + `order` |
| 跨交付物栏拖拽 | PATCH `deliverable_obj_id`（可兼带目标进度列） |

## 4. 实现要点

| 文件 | 变更 |
|------|------|
| `DeliverableKanbanBoard.vue` | 每交付物列内按 `taskStatuses` 渲染 `.progress-lane`；任务按 `filterTodosByKanbanColumn` ∩ 交付物列过滤 |
| `TaskPanel.vue` | Sortable 容器恢复读 `data-progress-column-id`；`buildDraggedTaskPatchPayload` 传入目标进度列 |
| intents `004` | 回写「纵=进度」验收；废弃「单任务区」条款 |
| Playwright | 面包屑用例期望存在 `.progress-lane`；补「下拉改进度后卡片换纵轴」用例 |

## 5. 非目标 / 架构

- 不新增 API、不改进度体系模型、不改 Archimate（无服务边界变更）
- 无新领域事件（沿用既有 Todo PATCH → 既有终态释放等消费者）

## 6. 验收标准

1. 每个 `.deliverable-column` 内有多个 `.progress-lane`（与进度体系列数一致）
2. 修改进度状态下拉后，卡片离开原进度纵轴、进入同交付物栏目标纵轴
3. 跨交付物拖拽仍更新 `deliverable_obj_id`
4. 跨进度拖拽仍更新 `progress_column_id`
