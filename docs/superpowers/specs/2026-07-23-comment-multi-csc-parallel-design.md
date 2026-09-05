# 设计：评论级多 CSC 并行实例（OPT-20260723-019）

**Date**: 2026-07-23  
**OPT**: OPT-20260723-019

## 问题

`cloud_server_configs` 上 `UNIQUE(company_id, task_id)` 迫使一任务一物理 CSC；independent 评论只能逻辑并行、无法挂接不同机器。

## 选定方案

1. **索引**：删除 `idx_csc_company_task`；新增非唯一索引 `idx_csc_workspace_task(workspace_id, task_id)`（按用户要求作为主查找索引）；新增 `UNIQUE(workspace_id, task_id, comment_id)`。
2. **列**：`cloud_server_configs.comment_id TEXT NOT NULL DEFAULT ''`（空串 = 任务级默认 CSC；非空 = 评论级实例）。
3. **调度**：`advance` 对每个可调度 binding 调用 `ensureCommentCloudServerConfig`，为该 `comment_id` 创建/复用独立 CSC（从任务级模板克隆 platform/region/auth，**不**复制 `instance_id`/`server_url`），binding 挂接各自 `csc_id`，独立 `task_{task}_{comment}` 容器名。
4. **独占纠偏**：仅当多个 live binding **误挂同一 csc_id** 时 demote；不同 csc_id 可同时 running。
5. **前端**：有独立 `csc_id` 的评论展示独立实例信息；任务级心跳面板仅挂在「当前执行」评论，避免多实例误显同一连接。

## 非目标（本迭代）

- 自动对每个评论触发完整阿里云 StartVM 计费流水（可后续接 bootstrap）
- 完整跨 CSC 心跳分流 SSE

## 业务意图 → 事件

| 意图 | 事件 | 发布点 |
|------|------|--------|
| 评论容器绑定推进（含独立 CSC） | CommentContainerBindingAdvanced | taskCloudService |

## 架构制品

- `docs/architecture/v54-application-integration-20260723-1705-claude.{puml,archimate,mermaid.md}`
