# NFR 澄清：入队前确认工作空间自动调度

增量：任务详情加入队列按钮先读工作空间调度启用态。L2 标准；无新写 API。

## 路径分片键审视

| 路径 | 分片键 | 说明 |
|------|--------|------|
| `GET /api/tenant/{tid}/workspace/{wid}/queue-schedule/` | `tenant_id` + `workspace_id` | 已携带；与 `workspace_schedule_rhythms.workspace_id` PK 对齐，合适 |
| `PATCH /api/tenant/{tid}/workspace/{wid}/todos/{taskId}/` | `tenant_id` + `workspace_id` + `task_id` | 既有入队写路径，本增量不改键 |
| 前端 `/tenant/:tenant/task-detail/:taskId/` | `tenant` + 路由 `workspace` | 任务详情已在租户/工作空间上下文 |
| 前端 `/tenant/:tenant/queue-schedule/?workspace_id=` | `tenant` + `workspace_id` | 设置页既有键 |

无可分片 ID 的新路径。可伸缩性：L0（单次用户点击 GET，无热路径）。升级触发：入队 QPS 需缓存启用态时再评估短 TTL。

## 幂等性审视

| 路径 | 副作用 | 等级 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------|------------|--------------|--------|----------|
| GET queue-schedule | 无（纯查询） | L0 | 双击会重复 GET | 不适用 | 无 | 只读；按钮 `saving` 防连点 |
| PATCH queued_auto_run=true | 有（入队成员行 + TaskQueuedForAutoRun） | L2 | 双击、超时重试 | 同一 task 入队一次 | 点击生成的 `Idempotency-Key`（既有 clickGuard） | 重试回传同一键；服务端 ON DUPLICATE KEY UPDATE |
| 未启用 modal + location.assign | 无服务端写 | L0 | 确认/取消各一次 | 前端导航 | 无 | 取消不写；确认整页跳转 |

前端防重放：加入按钮 `disabled` + `createClickGuard`（仅 PATCH 写路径）。用户点击是触发源，已规划门闩。
