# 意图：自动运行先建评论再带 comment_id 启机

## 背景与目标

CSC / start-vm 已改为评论级：`finalizeStartVmInGo` 无 `comment_id` 即 400「缺少评论ID」。
`triggerTaskAutoRun` 虽调用 `ensureAutoRunAtComment` 创建「【自动运行】」评论，但丢弃返回的评论 ID，仍按任务级 POST start-vm，导致评论卡显示启动失败。

目标：自动运行（含排队调度）必须先得到评论 ID，再把该 ID 写入 start-vm / start-vm-auto 请求体。

## 范围与边界

- 范围内：`taskTaskService` `triggerTaskAutoRun`、`startQueuedMembership`。
- 范围内：建评失败或评论 ID 为空时**硬失败**，禁止再打无 `comment_id` 的 start-vm。
- 范围外：`@镜像` mention 消费者（已传播 `comment_id`）；前端评论卡手动启机。

## 约束与风险

- 禁止软失败「评论失败仍启机」——评论级 CSC 下该路径必然 400。
- start-vm body 须含 `comment_id` 与 `parent_comment_id`（与 mention 路径对齐）。

## 验收标准

1. `triggerTaskAutoRun` 在 Cloud start-vm 请求体中携带 `ensureAutoRunAtComment` 返回的评论 ID。
2. `ensureAutoRunAtComment` 失败或返回空 ID 时不调用 start-vm，返回错误。
3. 排队 `startQueuedMembership` 同样先建/复用自动运行评论再带 `comment_id` 启机。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 自动运行先建评论再启机 | — | — | `triggerTaskAutoRun` / `startQueuedMembership` | Cloud start-vm（同步 HTTP） | 证据豁免：沿用既有 start-vm 直调；排队成功仍发 `QueuedAutoRunStarted` |

## 实施计划

1. `resolveAutoRunCommentID` 硬失败；`attachStartVmCommentID` 写入 body。
2. `triggerTaskAutoRun` 与 `startQueuedMembership` 共用上述两步。
3. 单测：带 ID 启机；缺 ID 不启机；排队路径带 ID。

## 变更记录

- 2026-08-15：评论卡「CSC 尚未分配」+ 启动日志「缺少评论ID」：`triggerTaskAutoRun` / `startQueuedMembership` 硬失败缺 ID，start-vm body 写入 `comment_id`+`parent_comment_id`；Cloud 禁止将该校验错误扇出到 binding 日志。
- 2026-08-14：评论级 CSC 后，自动运行必须先建评论再带 ID 启机；取消「评论失败仍启机」软失败。
