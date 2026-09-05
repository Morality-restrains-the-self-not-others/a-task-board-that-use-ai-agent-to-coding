# 价值流：编辑/创建非顶层类别须选上层交付物

- **日期**: 2026-07-15
- **设计**: `2026-07-15-edit-parent-deliverable-on-category-change-design.md`

## 增量（单一 MVP）

**VS-1**：用户在详情编辑或创建/编辑弹窗选择非顶层交付物类别时，必须从上一层类别任务中选择上层交付物并持久化 `parent_task`。

### 步骤

1. 打开详情/创建弹窗 → 选择交付物类别
2. 若非顶层 → 展示上层下拉（候选过滤）
3. 选择上层 → 保存/创建 → PATCH/POST 含 `parent_task`
4. 详情只读区展示上层（既有 024）

### 字段

- `taskTaskService.tasks.deliverable_obj_id`
- `taskTaskService.tasks.parent_task_id`

### 测试点

- 详情编辑非顶层必选上层
- 顶层清空 parent
- 创建任务同等
- 弹窗编辑同等
- 候选仅上一 order 层
