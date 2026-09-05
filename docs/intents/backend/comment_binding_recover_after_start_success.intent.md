# 意图：云主机启动成功后恢复 failed 评论容器绑定

## 背景与目标

任务详情评论执行细节同时出现红色「服务器启动失败」与启动日志「aliyun服务器启动成功！」。

根因（更正）：binding **一开始就不该是 failed**。首轮 `advance` 时评论 CSC 常为 mock（任务级云平台 / start-vm 尚未落库），旧逻辑（OPT-20260812-010）把这当成「未配置云平台」永久失败。随后阿里云 start-vm 成功写入日志，卡片却仍红条。

目标：mock/空平台首轮只保持 `starting`；**仅评论级 start-vm error** 才收口 failed。成功路径与 CSC 升级仍可把历史 failed 拉回 starting（存量防御）。

## 范围与边界

- 范围内：`persistServerSchedulingLogForBinding` 成功路径；`ensureCommentCloudServerConfig` 升级后恢复；任务级 fan-out 排除 failed+success。
- 范围内：前端 `reconcileBindingServerStatusWithStartupLogs` 防御展示。
- 范围外：真正的云 API 失败（无成功日志）仍保持 failed 红条。

## 约束与风险

- 不得把任务级成功误恢复所有 failed binding（可能含真实失败的其它评论）。
- 恢复后状态为 `starting` 而非 `running`（仍须 server_url / reachability）。

## 验收标准

1. 评论级 SSE `status=success`（「aliyun服务器启动成功」）后，该 comment 的 failed binding 变为 `starting`，并有 `server_started` 日志。
2. 任务级（无 comment_id）success 不写 `server_started` 到 failed binding，也不改其 status。
3. 评论 CSC mock→aliyun 升级后，对应 failed binding 变为 `starting`。
4. 前端 failed + 成功日志 → 不渲染 `server-startup-error-banner`，生命周期为「启动中」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 启动成功恢复 failed binding | — | — | `publishTaskSSE` / `ensureCommentCloudServerConfig` | binding 行 status 更新 | 证据豁免：同步状态机，不另发领域事件 |

## 实施计划

1. `recoverFailedCommentBindingToStarting`。
2. 评论级 success 与 CSC 升级调用恢复；任务级 success 不扇出 failed。
3. 前端按启动日志纠正红条。

## 变更记录

- 2026-08-14：任务详情页日志成功与红条失败矛盾；failed 不随 start-vm 成功回写。
- 2026-08-14：根因更正——首轮 mock 不得收口 failed；仅评论级 start-vm error 才标失败。
