# 编辑/创建：非顶层交付物类别须选上层交付物

- **日期**: 2026-07-15
- **状态**: approved（goal-mode 自动采用）
- **范围**: 前端任务详情编辑身份区 + CreateTaskModal（创建与编辑）+ 保存 payload
- **架构影响**: 无（无新服务/无新表/无新 HTTP path）
- **Python 新接口**: 无

## 问题

交付物详情「编辑」时可改交付物类别，但非顶层类别时不能选择/修改「上层交付物」。创建任务已有该规则；编辑弹窗（`CreateTaskModal` 带 id）刻意隐藏了上层字段。

## 方案（采用）

复用既有规则与组件：

| 规则 | 行为 |
|------|------|
| 顶层类别 | `order` 最小；不展示上层下拉；保存时 `parent_task=""`（清空） |
| 非顶层 | 展示「上层交付物」下拉；候选 = 上一 `order` 层任务；必选后方可保存/创建 |
| 切换类别 | 若当前 parent 不在新候选中则清空 |

### 落点

1. **详情编辑** `TaskDetailTaskIdentityPanel`：编辑态复用 `CreateTaskParentDeliverableField`；只读态仍按 024 展示跳转链接。
2. **`startEdit`**：回填 `parent_task`；拉取/持有工作空间 `todos` 供候选过滤。
3. **`saveEdit`**：PATCH 写入 `parent_task`（顶层显式空串）。
4. **CreateTaskModal**：去掉「仅创建」限制；编辑同样门禁与字段。
5. **`appendParentTaskToCreatePayload`**：顶层时也写入空串，保证编辑清空生效。

### 不采用

- 改 `DeliverableObj` 为树形 `parent_id`（过大，与现 `order` 约定冲突）
- 新后端过滤 API（候选继续前端纯函数）

## Domain 概念（轻量）

- Bounded Context: Task / Deliverable
- Entity: Task（`deliverable_obj_id`, `parent_task`）
- Value: DeliverableCategory（`order` 定层级）

## 价值流影响

- 影响 stream：任务协作 / work-panel 创建与编辑交付物
- 字段：`taskTaskService.tasks.parent_task_id`、`deliverable_obj_id`（语义不变，编辑路径补齐）

## 验收

1. 详情编辑选非顶层类别 → 出现上层下拉，选项仅上一层类别任务。
2. 未选上层 → 保存禁用或报错。
3. 改回顶层 → 上层字段消失，保存后无 parent。
4. 创建任务保持同等规则。
5. CreateTaskModal 编辑模式同等规则。
