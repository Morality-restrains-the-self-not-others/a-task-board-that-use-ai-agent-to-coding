# NFR 澄清 — 创建任务可选加入自动调度队列

- 日期：2026-08-27
- 默认等级：L2；云资源启服路径 L3 沿用既有 auto_run / queued dispatcher

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | tenant_id + workspace_id | 是，tenant 为分片键 | L2 | 无 |
| POST/PUT `/api/tenant/{tid}/workspace/{wid}/todos/` | tenant_id + workspace_id | 是 | L2 | 无；`queued_auto_run` 为 body 字段 |
| 前端 `/tenant/:tenant/work-panel/` | tenant | 是 | L1 | 无 |
| 事件 `TaskQueuedForAutoRun` | tenant_id / workspace_id / task_id | task_id 适合入队去重 | L2 | 沿用 membership PK |

无「缺分片 ID」路径。

## 幂等性强制审视

| 路径 | 副作用 | 重复源 | 业务边界 | 幂等键 | 等级 | 重放 |
|------|--------|--------|----------|--------|------|------|
| GET queue-schedule | 无 | — | — | — | L0 | 纯查询 |
| POST 创建 + queued_auto_run | 入队 +（不）启服 | 双击、超时重试 | 同一创建意图 | 既有 `Idempotency-Key` / 创建去重窗；membership `task_id` PK | L2 | 第二次返回已创建任务；enqueue ON DUPLICATE KEY |
| 立即 start-vm 跳过 | 避免双启服 | 同上 | 同一 task | queued 则不调 scheduleTaskAutoRun | L3 防护 | 启服只走 dispatcher `started_via=queued_schedule` |
| Kafka queued_auto_run_scan | 启服 | timer 重叠 | membership 行 | 既有 dispatcher 槽位 | L3 | 沿用 |

资金/云资源：不新增启服入口；queued 路径仍走既有 start-vm + slots。
