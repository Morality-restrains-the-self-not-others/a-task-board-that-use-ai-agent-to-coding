# NFR 澄清：任务/项目数据 GET 与 GitLab 探活分离

- 日期：2026-08-31
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`
- 价值流：`docs/superpowers/plans/2026-08-31-task-detail-gitlab-probe-split-value-stream.md`
- 默认：性能 L3（详情读路径）；资金不适用

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 级别 | 动作 |
|------|---------|----------|------|------|
| GET `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/` | tenant_id + workspace_id + task_id | 是 | L2 | 保持路径；禁止无 task_id 扫表 |
| GET `/api/projects/tenant_id/{tid}/{projectId}` | tenant_id + project_id | 是 | L2 | 保持 |
| POST `/api/projects/validate-git-repos/tenant_id/{tid}/` | tenant_id | 探活按用户 token，URL 列表有界 | L1 | body urls 限条数（既有）；不用 user_id 当分片键 |
| FE `/tenant/:tid/workspace/:wid/task-detail/:taskId` | tenant + workspace + task | 是 | L0 路由 | — |

升级触发：validate-git-repos 单请求 urls > 50 再分页。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 级别 |
|------|--------|------------|--------------|--------|----------|------|
| GET 任务 / GET 项目 | 无 | — | — | — | — | L0 纯查询 |
| POST validate-git-repos | 无系统事实（只读远端） | 页面加载/重试 | 同用户同 URL 列表 | 无写库 | 重复 POST 只再探一次 | L0 无写副作用 |
| 前端探活按钮/自动 | 无写 | 双击 | 同 URL | Anti-Replay-OK：只读探活 | 可并行，不挡详情 | L0 |

禁止用 tenant_id / user_id 作事件消费幂等键（本增量无消费者）。

## 质量场景

- 刺激：GitLab `dial timeout`。响应：任务 GET P99 < 200ms（本机 05:27 基线 12–19ms），HTTP 200。
- 刺激：探活 POST 超时。响应：详情已渲染；badge 失败 + `data-traceId`。

## 领域模型影响

无新聚合。读模型 `GitRepoProbeStatus` 仅前端状态 + 既有 POST 响应。不引入领域事件。
