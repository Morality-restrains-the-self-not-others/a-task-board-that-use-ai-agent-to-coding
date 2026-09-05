# 设计：评论级容器编排 + 依赖模式持久化（OPT-038/039）

**Date**: 2026-07-22  
**OPTs**: OPT-20260722-038, OPT-20260722-039（附带 OPT-040 接线）

## 选定方案

### OPT-039（依赖模式持久化）
1. `taskTaskService.comments`、`taskAIComment.ai_task_comments` 增加 `execution_mode TEXT NOT NULL DEFAULT 'wait_previous'`（取值 `wait_previous`|`independent`）。
2. Go PATCH：人类评论 TTS；AI 评论 taskAIComment；创建时可选带入。
3. Feed / enrich 回传；前端执行细节内切换并 PATCH。

### OPT-038（每评论一容器 + 调度）MVP
1. **Owner**：taskCloudService 新表 `comment_container_bindings`：
   - `comment_id`, `task_id`, `company_id`, `execution_mode`, `depends_on_comment_id`, `status`（pending|waiting_previous|starting|running|completed|failed|released）, `csc_id`/`mock_container_name` 可空
2. **调度**：`POST .../comment-container/advance`（或创建绑定后自动）：
   - `independent` → 立刻调度（可与其它并行，**不等待前序 completed**）
   - `wait_previous` → 前序 binding `completed` 后才启动
3. **物理容器**：
   - Mock：`go_run_container` 容器名 `task_{task}_{comment}`（每评论唯一）
   - Cloud（OPT-019）：索引 `(workspace_id,task_id)`；`UNIQUE(workspace_id,task_id,comment_id)`；并行评论各自 ensure 独立 CSC。
4. **前端**：当前执行挂完整连接面板；其它 live 评论展示独立容器名 + csc_id。

### 业务意图 → 事件对照

| 意图 | 事件 | 发布点 | 例外 |
|------|------|--------|------|
| 依赖模式变更 | CommentExecutionModeChanged | TTS / taskAIComment | — |
| 评论容器绑定推进 | CommentContainerBindingAdvanced | taskCloudService | — |

## 非目标（本迭代）
- 每个评论自动触发完整阿里云 StartVM 计费流水（CSC 行已就位）
- 完整计费拆分 / 跨 CSC 心跳 SSE

**续见**：`2026-07-23-comment-multi-csc-parallel-design.md`
