# DDD 轻量建模：闲置复用启动保护 + 孤儿交叉校验

- **限界上下文**: Cloud Runtime（taskCloudService）
- **聚合**: `CloudServerConfig`（按 task 绑定机器）
- **领域服务**:
  - `IdleReuseSelector` — 仅 TrueIdleMachine（全绑定空 server_url + idle_since + started + 非 Starting）
  - `OrphanInstanceReconciler` — 删除前查 CrossCSCOwnership

## 业务意图 → 事件

无新事件（见意图文档例外表）。既有 `CLOUD_SERVER_START_AUTO` / reconcile 路径不变。

## 不变式

1. 无 `idle_since` 的 Running 空 URL 机器 ∉ idle reuse 候选。
2. `instance_id` 被任意同 workspace CSC 持有 ⇒ orphan reconcile 不得 DeleteInstance。
