# 意图：评论级容器绑定与串行/并行调度

## 背景

任务详情评论需独立容器编排门禁：`wait_previous` 串行、`independent` 可并行。OPT-019 起每评论可挂独立 CSC。

## 目标

1. taskCloudService 表 `comment_container_bindings`（comment_id UNIQUE / task 作用域）。
2. API：list/create/advance/complete/**cancel**；workspace `cloud/compute/comment-container-bindings/?task_id=`。
3. 调度：independent → 立即调度（不等待前序）；wait_previous 等前序 completed。
4. **多 CSC**：`cloud_server_configs` 索引 `(workspace_id, task_id)`；唯一键 `(workspace_id, task_id, comment_id)`（`comment_id=''` 为任务级默认）；advance 为每评论 `ensureCommentCloudServerConfig`，并行评论挂接**不同** `csc_id`。
5. 发布 `CommentContainerBindingAdvanced`（publishDomainEvent）。
6. **容器命名**：每条评论绑定独立容器名；任务 ID 未带 `task_` 时为 `task_{任务ID}_{评论ID}`，已带 `task_` 时为 `{任务ID}_{评论ID}`（禁止 `task_task_…` 双前缀）；启动 mock/go_run_container 时传 `comment_id`。
7. **ECS InstanceName**：评论级云主机与容器名相同（`task_{taskId}_{commentId}`）；Describe/heal/orphan **只按该名**查找，不回退任务级/`task-{taskId}`。缺少 `comment_id` 不启机命名、不对账。
8. **@镜像冷启动门禁**：`TASK_COMMENT_IMAGE_MENTIONED` 仅在 `independent`，或 `wait_previous` 且无未完成前序时调用 StartVM。有前序的串行评论由 binding advance 在前序 completed 后经 `ccbStartBinding` → bootstrap start-vm。`start-vm` / `start-vm-auto` 对 `wait_previous` + `waiting_previous` 返回 409。
9. **independent @镜像单一建机**：mention `StartVM` 与 `bootstrapCommentCSCRuntime` 自调 `start-vm` 不得并发 `RunInstances`。`taskCloudService` 按 `company|task|comment` 进程内串行；同评论 CSC 已有非 mock Starting/Running 实例时第二条请求幂等 200（`duplicate_skipped`），且不得覆盖 `start_trace_id`。mention 挂掉时 bootstrap 仍可建机（不因残留 pending start 事件永久跳过）。
10. **终止等待前序**：`waiting_previous` 可由用户 POST `.../{commentId}/cancel/` → `cancelled`；不再参与 advance 调度；不启机。依赖该评论的后续串行仍按「未 completed」阻塞（与 failed 一致）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 评论容器绑定推进 | CommentContainerBindingAdvanced | CommentContainerBindingAdvanced | taskCloudService `publishCommentContainerBindingAdvanced` | 前端刷新 binding；审计 | — |
| 用户终止等待前序 | CommentContainerBindingAdvanced（status=cancelled） | 同上 | `handleCommentContainerBindingCancel` | 前端刷新；不再调度 | 复用 Advanced 事件，避免新 topic |
| 评论 @镜像启机 | TASK_COMMENT_IMAGE_MENTIONED | TASK_COMMENT_IMAGE_MENTIONED（含 `execution_mode` / `has_unfinished_predecessors`） | taskTaskService `buildTaskCommentImageMentionedData` | taskEvents `1_start_vm_for_at_mention`：门禁后 StartVM | 串行有前序时消费成功但不启机 |

## 变更记录

- 2026-07-22：OPT-20260722-038 初版
- 2026-07-23：independent 并行时 CSC 独占，修复两评论复用同一容器连接
- 2026-07-23：容器名统一为 `task_{taskId}_{commentId}`；执行细节摘要展示容器名
- 2026-07-23：OPT-019 多 CSC（workspace_id+task_id 索引 / comment 级唯一）
- 2026-08-13：任务 ID 已含 `task_` 时不再二次拼接（修复 `task_task_…` 双前缀）
- 2026-08-13：Workbench 读当前物理机 CSC（评论级回退），见 `workbench_link_runtime_csc`
- 2026-08-13：`start_trace_id` 按评论持久化；禁止用 task_id 冒充，见 `comment_container_binding_start_trace`
- 2026-08-13：binding 日志 `created_at` 按 UTC 读回并输出 RFC3339 `Z`，见 `comment_container_binding_log_utc`
- 2026-08-16：@镜像冷启动对齐 binding：`wait_previous` 有前序时不得 StartVM；`start-vm` 对 `waiting_previous` 返回 409
- 2026-08-16：评论级 ECS InstanceName 与容器名对齐；查找不再回退任务级/legacy 名
- 2026-08-17：waiting_previous 支持用户终止 → cancelled（API cancel + 执行细节摘要「终止」）
- 2026-08-18：公网 IP 落库不得写入推测性 `server_url`；仅 register-reachability 后升 running（防「服务可用」假阳性）
- 2026-08-18：independent @镜像 mention + bootstrap 双重 start-vm 互斥；同评论已绑定实例幂等跳过且不覆盖 start_trace_id
