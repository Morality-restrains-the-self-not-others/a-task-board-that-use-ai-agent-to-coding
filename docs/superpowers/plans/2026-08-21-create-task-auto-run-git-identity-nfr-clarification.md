# NFR 澄清：创建任务自动运行 Git 提交身份

- **日期**: 2026-08-21
- **默认等级**: L2 Standard（身份选择非资金路径）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作 |
|------|---------|----------|----------|------|
| `POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/` | tenantId + workspaceId | 是，与任务表租户/工作空间对齐 | L0 本增量不改路由 | 无 |
| work-panel `/tenant/:tenantId/work-panel` | tenantId | 是 | L0 | 无 |
| GET `/api/git-identities/user/{userId}/` | userId | 用户级列表，基数低 | L0 理由：每用户身份条数个位数；升级触发：单用户 >1000 条再考虑 company 过滤索引 | 无 |
| Chrome 插件 POST 同上 | 同 todos | 是 | L0 | 无 |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST 创建任务 + repo_identities | 写任务 + 可能写【自动运行】评论 | 双击/重试 | 同一 Idempotency-Key 或 fork 窗 | 既有 `Idempotency-Key` / fork 去重 | 命中返回已创建任务，不二次写评论 |
| PUT 更新 auto_run/force | 可能再 schedule auto_run | 强制重启按钮 | task_id + 进行中 auto_run 评论 | `findActiveAutoRunAgentByTask` 复用评论 | 复用时 UPDATE JSON，不新建评论 |
| GET git-identities | 无 | — | — | L0 只读 | — |

资金/配额路径：本增量不改扣配额；创建任务配额仍走既有 consume。

## 质量场景

- 刺激：用户 auto_run 未选身份点创建。响应：前端拦截，不发 POST。
- 刺激：绕过前端 POST auto_run 无身份。响应：400 `repo_identities_required`。
- 刺激：无仓项目 auto_run。响应：不要求身份。

## 领域模型影响

身份是自动运行评论的值对象列表，不新聚合。校验是领域服务函数，与 `ValidateRepoIdentitiesForRun` 并列（创建路径不要求 github_user_id）。
