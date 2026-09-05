# 设计：闲置机器复用限定同工作空间同镜像调用人员

**Date**: 2026-07-22  
**选型**: 在 taskCloudService idle reuse 候选过滤中匹配 CSC `image_invoker_user_id`（评论人员 / 镜像调用人员）；排队调度走同一 `start-vm-auto` 门禁自动生效；复用时解绑源任务并通知 TTS 释放排队槽。

## 背景

曾用任务 `owner_id`（TTS HTTP 查找）做门禁，与产品语义不符：应限制为**同一镜像调用人员**（`@installed_image` 评论作者 / 实际启服用户）。排队调度（`started_via=queued_schedule`）同样经 `applyStartVmPolicyGateAndReuse`，须遵循同一原则。

## 规则

1. **复用前提**：同租户、同工作空间，且源 CSC `image_invoker_user_id` 与目标请求的镜像调用人员 ID 相同。
2. **调用人员来源**：优先 body `image_invoker_user_id` / `comment_created_by_id`，否则 `X-Auth-User-Id`（TTS `startVM` 会写入 body）。
3. **持久化**：冷启动 finalize 后 `setCloudServerImageInvokerUserID`；复用绑定写入目标 CSC。
4. **查不到 invoker / 源 CSC 无 invoker**：fail-closed——跳过复用，走冷启动。
5. **排队调度**：不新增平行逻辑；`startQueuedMembership` → `start-vm-auto` 已走复用门禁（传排队启动用户为 invoker）。
6. **先释放再启动**：复用时 `unbindSourceMachineAfterReuse` 清空源 CSC 绑定；并 `notifyTaskServiceDequeueQueued(source, idle_reuse)`。

## 非目标

- 不以任务 `owner_id` 作为复用门禁
- 不跨工作空间复用
- 不为他调用人员强制 hard-release ECS（仅跳过其闲置机）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外 |
|---------|--------|---------|--------|------|
| 同镜像调用人员闲置复用绑定 | — | — | taskCloudService 日志 `idle_reuse_bound` / 既有 SSE | 无对应新 MQ：复用为内部绑定迁移 |
| 源任务排队槽释放 | TaskDequeuedFromAutoRun（既有） | 既有 | TTS dequeue API（Cloud notify） | — |

## 变更记录

- 2026-07-22：初版 owner_id 门禁
- 2026-07-22：改为 CSC `image_invoker_user_id`（评论/镜像调用人员）
