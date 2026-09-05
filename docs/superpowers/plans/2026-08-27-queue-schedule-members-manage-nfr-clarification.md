# 自动调度安排 · 排队任务卡可管理 — NFR 澄清

- 日期：2026-08-27
- 默认等级：L2（Standard）；资金/云资源路径不在本增量直接触发启服（启服仍由既有 dispatcher）

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| 前端 `/tenant/:tenant/queue-schedule/?workspace_id=` | tenant + workspace_id | ✅ 租户+工作空间对齐队列表 | 保持 |
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | tid + wid | ✅ | 保持 |
| PATCH `/api/tenant/{tid}/workspace/{wid}/todos/{taskId}/` | tid + wid + taskId | ✅ task 属 workspace | 禁止省略 wid |
| GET `/api/tasks/search/tenant_id/{tid}/?workspace_id=` | tid；query workspace_id | ✅ tenant 为搜索索引；workspace 过滤 | 前端强制带 workspace_id |

可伸缩性：**L2**。队列按 workspace 查询；搜索已有 tenant 前缀。升级触发：单 workspace 队列成员 > 1 万再分页（当前快照全量，与既有 GET 一致）。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET queue-schedule / search | 无 | — | — | L0 只读 | — |
| PATCH queued_auto_run=true | 入队 | 双击、超时重试 | 同一 task 入队 | 前端 `Idempotency-Key`（点击一次一键）；服务端 membership PK=task_id | 重复入队空操作/已在队 |
| PATCH queued_auto_run=false | 出队 | 双击、超时重试 | 同一 task 出队 | 同上 | 重复出队空操作 |
| 刷新 GET | 无写 | 连点 | — | L1 锁，无 Idempotency-Key（read-refresh） | — |

用户点击是触发源 → 同步门闩 + 写路径 Idempotency-Key（元规则 52）。资金/配额不在本路径；dispatcher 启服不在本增量。

## 其他 NFR

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | 沿用 todos PATCH 鉴权；XSS 用文本插值；无密钥 |
| 可用性 | L2 | 失败展示 data-traceId；确认取消可恢复 |
| 可观测性 | L2 | 前端 console 事件名 `queue_members_join` / `queue_members_leave`（仅 task_id）；后端沿用既有 PATCH 日志 |
| 性能 | L1 | 搜索 debounce 250ms、limit 20；无轮询 |

## 领域模型影响

无新聚合。入队/出队仍落 `QueuedAutoRunMembership`。幂等边界 = task_id，与既有 PK 一致。
