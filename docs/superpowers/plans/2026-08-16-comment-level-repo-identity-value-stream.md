# 价值流：评论级仓库身份

> 设计：`docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`

## Related Value Streams

- `comment_clone_progress_under_trace`：克隆进度已在评论执行细节；本流**删除**关联项目上的任务级进度条（旧流范围外写「不移除关联项目进度」→ 本流明确移除）。
- `031_no_task_level_server_config` / ADR-0005：运行态已评论级；本流把身份对齐到同一生命周期。
- `2026-05-28-task-detail-linked-project-name-link`：关联项目名称链接保留（展示）。
- 不冲突：不撤销评论级 CSC。

## 价值阶段

| 阶段 | 类型 | 说明 |
|------|------|------|
| 看清任务绑了哪些仓 | Essential | 关联项目只读 |
| 提交并运行时选定身份 | Core | composer |
| 身份随评论落库并驱动启动 | Core | JSON + 事件 + snapshot |

## 端到端流

Trigger：用户 @镜像 准备运行  
→ composer 按仓库选 Git 身份与授权账号（未绑定则 OAuth）  
→ POST comment 带 repo_identities  
→ 落库 + TASK_COMMENT_IMAGE_MENTIONED  
→ start-vm / snapshot 用该评论身份克隆  
Delivery：该评论容器使用本次选定署名与账号；关联项目不再出现进度/身份控件

## 增量切片

1. **I1 关联项目只读** — 去掉身份/进度/reclone UI + 单测
2. **I2 composer 身份** — 面板 + 草稿 + 提交校验
3. **I3 持久化与 snapshot** — DDL + POST/GET + 事件 + snapshot comment_id
4. **I4 文案/E2E 对齐** — Playwright OAuth 入口改到 composer

字段：`taskTaskService.task_comments.repo_identities_json`
