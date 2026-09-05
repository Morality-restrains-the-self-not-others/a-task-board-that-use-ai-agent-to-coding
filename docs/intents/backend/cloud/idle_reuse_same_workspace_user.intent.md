# 意图：闲置机器复用仅限同工作空间同镜像调用人员

**Status:** superseded  
**Superseded by:** [ADR-0013](../../../adr/0013-remove-idle-machine-reuse.md)（2026-08-16 去掉跨任务闲置复用）

跨任务闲置复用已拆除。同评论 inflight 附着与闲置自动回收仍有效。下文为历史目标，不再实施。

## 目标

1. Idle reuse 候选须与目标请求的「镜像调用人员 / 评论人员」ID 相同（同 workspace 已有）；**不以**任务 `owner_id` 为准。
2. 排队调度启动经同一复用门禁，不得跨调用人员抢机。
3. 复用时解绑源任务 CSC，并通知 TTS dequeue 释放排队槽。
4. 启动时将调用人员持久化到 CSC `image_invoker_user_id`（body `image_invoker_user_id` / `comment_created_by_id` 或认证用户）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 同镜像调用人员闲置复用 | — | — | taskCloudService `bindSharedMachineToTask` | CSC 绑定迁移；SSE reuse；notify TTS dequeue 源任务 | 无对应事件：内部绑定迁移；出队沿用既有 TTS dequeue 路径 |

## 变更记录

- 2026-08-16：ADR-0013 拆除复用实现；本意图 superseded
- 2026-07-22：初版（owner_id）
- 2026-07-22：改为评论人员 / 镜像调用人员（CSC `image_invoker_user_id`）
