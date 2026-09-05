# 权限分析：编辑时非顶层类别须选上层交付物

- **日期**: 2026-07-15
- **设计文档**: `2026-07-15-edit-parent-deliverable-on-category-change-design.md`
- **结论**: 无新端点；复用 PATCH todos；权限边界充分

## 改动点权限表

| 改动点 | 主体 | 资源 | 操作 | 现有检查 | 缺失? |
|--------|------|------|------|----------|-------|
| 详情编辑改 parent_task | 可编辑任务的成员 | Workspace/Task | write | taskTaskService + 网关 | 否 |
| GET workspace todos（候选） | 同上 | Workspace | list | 既有 list | 否 |
| CreateTaskModal 编辑写 parent | 同上 | Workspace/Task | write | 同上 | 否 |

## IDOR

- 候选列表仅来自当前工作空间 todos；选中的 parent id 由服务端 UPDATE 写入，仍受 workspace 归属约束。
- 不得跨工作空间写入 parent_task。

## 角色

无需新增角色。
