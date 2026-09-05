# 价值流：终态按评论 CSC 释放

- 日期：2026-08-14
- 设计：`docs/superpowers/specs/2026-08-14-terminal-release-comment-csc-design.md`

## 价值主张

任务进入终态后释放**该任务全部已启动评论机器**，且无运行资源时不产生 DLT。

## 增量（单增量交付）

| Step | 说明 | 验证 |
|------|------|------|
| S1 | list-by-task 返回评论 CSC + 两计数，不含模板行 | Go 单测 |
| S2 | 无评论运行态 → DispatchSuccess | Go 单测（本 DLT 三元组） |
| S3 | 已启动评论 → 每台一条 CLOUD_SERVER_STOPPED（含 comment_id） | Go 单测 |
| S4 | Starting 无 instance → success no-op | Go 单测 |

## 既有流

- `task-management` / todo-manage-status（终态触发）
- 云资源 `CLOUD_SERVER_STOPPED`

## YAML 字段

- `taskCloudService.cloud_server_configs.comment_id`
- `taskCloudService.cloud_server_configs.instance_id`
- `taskCloudService.cloud_server_configs.running_machine_count`
