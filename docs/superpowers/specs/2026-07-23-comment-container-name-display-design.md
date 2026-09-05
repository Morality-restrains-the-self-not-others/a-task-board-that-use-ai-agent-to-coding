# 设计补丁：评论执行细节展示容器名 + 命名规范

- **日期**: 2026-07-23
- **状态**: approved（goal-mode 自动采用）
- **迭代名**: `comment-container-name-display`
- **基线**: `2026-07-22-comment-execution-details-design.md` + comment_container_bindings 编排

## 问题

任务详情「执行细节」摘要仅显示依赖模式与「容器 运行中」，未展示该评论绑定的容器名；命名亦未统一为 `task_{任务ID}_{评论ID}`。

## 方案（采纳）

1. **命名**：`buildCommentMockContainerName` → 任务 ID 未带 `task_` 时为 `task_{taskId}_{commentId}`，已带 `task_` 时为 `{taskId}_{commentId}`（禁止双前缀）；create binding 即写入；JSON 返回 `container_name`（与 `mock_container_name` 同值）。
2. **启动**：`go_run_container` / `mock_run_container` / gateway 接受 `comment_id`，命名同上；无 comment 时保留任务级 `taskId_{taskId}` 兼容。
3. **UI**：`TaskDetailCommentExecutionDetails` summary 展示容器名 badge（`data-testid=comment-execution-container-name`）；无 API 值时前端按同规则推导。

## 非目标

本补丁不强制为每条评论 provision 独立物理 CSC（仍见 OPT 多实例供给）。
