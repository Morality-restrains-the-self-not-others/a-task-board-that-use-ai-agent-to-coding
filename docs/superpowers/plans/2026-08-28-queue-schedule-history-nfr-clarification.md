# 自动调度安排 · 调度历史 — NFR 澄清

- 日期：2026-08-28
- 默认：L2；写历史为审计投影，非资金路径

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| 前端 `/tenant/:tenant/queue-schedule/?workspace_id=` | tenant + workspace_id | ✅ | 保持 |
| GET `/api/tenant/{tid}/workspace/{wid}/queue-schedule/` | tid + wid | ✅ | 保持；附加 recent_history |
| GET `.../queue-schedule/history/` | tid + wid | ✅ workspace 为查询键 | 禁止省略 wid；cursor 不跨空间 |
| 内部 append（无 HTTP） | workspace_id 列 | ✅ | 所有 INSERT 带 workspace_id |

可伸缩性：**L2**。表按月分区；查询 `(workspace_id, created_at DESC)`。年增量按「仅状态变化」预估远低于每 30s 扫描。升级触发：单 workspace 热窗 > 100 万行再考虑归档库。

分片键判定：`workspace_id` 合适（历史按工作空间隔离）；`tenant_id` 冗余过滤防串租户。主键含 `created_at` 以兼容分区。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET snapshot / history | 无 | — | — | L0 只读 | — |
| 加载更多 GET | 无 | 连点 | — | L1 锁 | — |
| append on enqueue | 插入历史 | 重复 PATCH 入队 | 每次真实入队状态变化 | 不另建幂等键；membership PK 已防双入队 | 若业务未变化则不会第二次 enqueue 成功路径 |
| append on window flip | 插入 + 更新 last_in_window | timer 30s | 同一 workspace 同方向翻转 | `last_in_window` 比较 | 同态不写 |
| PUT 节奏 | 既有保存 + append | 双击保存 | 一次用户保存 | 既有 Idempotency-Key | 每次成功保存记一行（配置变更审计） |

资金/云资源启服不在本查询路径；start-vm 仍走既有 dispatcher。用户点击加载更多不是写。

## 其他 NFR

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | 同 GET 快照鉴权；message 文本插值防 XSS |
| 可用性 | L2 | append 失败不影响调度；读失败 data-traceId |
| 可观测性 | L2 | `schedule_history_append_failed` / `schedule_history_listed` |
| 性能 | L2 | limit 默认 20、max 50；recent_history 8；无轮询 |
| 冷热 | L2 | 月分区；保留约 90 天热窗 |

## 领域模型影响

新实体 **QueuedScheduleHistoryEntry**（工作空间内时间线，非聚合根独立生命周期）。窗口翻转属于 WorkspaceScheduleRhythm 的衍生事实。
