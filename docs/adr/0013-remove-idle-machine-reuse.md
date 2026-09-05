# ADR-0013: 去掉跨任务闲置机器复用

- **Status:** accepted
- **Date:** 2026-08-16
- **Author:** Trae AI 团队
- **Deciders:** 用户确认「去掉闲置复用」；同评论 inflight 附着保留

---

## Context

工作区曾提供 `prefer_idle_reuse`：`start-vm` / `start-vm-auto` 在同 workspace 扫描闲置 CSC（空 `server_url` + `idle_since` + 非 Starting + 同 `image_invoker_user_id`），把源任务的 `instance_id` 绑到目标任务。

[ADR-0007](0007-task-level-running-comment-server-count.md) 之后运行态只在**评论 CSC**，任务级行是模板+计数。复用实现仍读/写任务级 `last_runtime_status` / `instance_id`，现网评论机几乎不会被判可复用；误绑也到不了目标评论卡。评论级 `InstanceName` 之后，即便修绑定也必须改名，复杂度高、收益低。

终态迁机（`migrate-container-off-instance`）同样先走闲置复用再冷启动，会把兄弟任务绑到错误身份的机器上。

同评论 inflight 附着（本评论已有 Starting/Running 实例时跳过二次冷启动）仍是正确的幂等，与跨任务复用不是同一机制。

## Decision

**We will** 拆除跨任务闲置复用，保留同评论 inflight 附着与闲置自动回收：

1. `start-vm` / `start-vm-auto` 门禁只做授权白名单 + `tryAttachSameTaskInFlightMachine`；无附着则冷启动
2. 迁机一律 `start-vm-auto` / mock 冷启动，不再 `tryReuseIdleMachine*`
3. `prefer_idle_reuse` 读写恒 false；**2026-08-20 起列已删除**（`dataMigrate/taskCloudService/029_remove_prefer_idle_reuse_column.sql`，见 OPT-20260818-064），Go/OpenAPI/前端不再出现该键
4. 闲置回收（`idle_recycle_minutes` / `idle_since` / recycle worker）不变
5. 孤儿对账 `skip_owned`（他 CSC 仍持有则不删）不变

## Alternatives Considered

### Alternative 1: 按评论 CSC 重做复用（OPT-20260816-063 原方案）

- **Pros:** 保留「少开机器」；对齐评论级运行态与 InstanceName
- **Cons:** 需改绑目标评论 CSC、解绑源评论、`ModifyInstanceAttribute` 改名、排队 dequeue；与多评论独立机身份冲突，串台风险仍在
- **Why rejected:** 用户要求去掉闲置复用，而不是修到评论级

### Alternative 2: 仅默认关闭开关，保留代码路径

- **Pros:** 改动面小，可随时打开
- **Cons:** 死代码仍按任务级绑机会在误开时串台
- **Why rejected:** 开关打开即是错误架构；必须删实现

### Alternative 3: 连同评论 inflight 附着一起删

- **Pros:** start-vm 路径更短
- **Cons:** 同评论重试会二次 RunInstances，触发 supersede / Initializing 误杀
- **Why rejected:** inflight 附着是幂等，不是跨任务复用

## Consequences

### Positive

- 每条评论冷启动自己的实例，InstanceName / 容器名 / 评论卡一一对应
- 不再把闲置机写回任务级模板
- 策略 UI 只保留回收与授权白名单，语义清晰

### Negative / Trade-offs

- 同 workspace 闲置机不会被新评论/新任务领养，ECS 用量上升，依赖回收分钟数回收

### Mitigations

- 闲置自动回收仍按 `idle_recycle_minutes` 工作
- 同评论重试走 inflight 附着，避免双开
- 存量 `prefer_idle_reuse=1` 先由 `dataMigrate/taskCloudService/024_disable_idle_reuse.sql` 清零，再经 `029_remove_prefer_idle_reuse_column.sql` 删列

## References

- [ADR-0007: 任务级 CSC 只标记运行中机器数与容器数](0007-task-level-running-comment-server-count.md)
- `docs/intents/backend/cloud/idle_reuse_same_workspace_user.intent.md`（superseded）
- `docs/intents/backend/cloud/idle_reuse_boot_guard_orphan_cross_check.intent.md`（复用部分 superseded；孤儿交叉校验仍有效）
- OPT-20260816-063
