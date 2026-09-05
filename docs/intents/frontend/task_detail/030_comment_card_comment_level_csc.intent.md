# 意图：评论卡片运行态按钮请求评论级服务器配置

## 背景与目标

任务详情评论执行细节里的 Workbench / 刷新状态 / 停止服务器 原先只带 `task_id`。后端 `loadCloudServerConfigForRuntime` 在任务级 CSC 已有 `instance_id` 时不会改读评论行，导致评论卡按钮打到任务级或「最新带 instance 的行」，多评论并行会点错机器。

目标：评论卡片上的 compute 请求一律带该评论 `comment_id`；后端在 `comment_id` 非空时**只读该评论 CSC**，即使 instance 为空也不回退其它行。

## 范围与边界

- 范围内：`workbench-link`、`server-runtime-status`、`stop-vm`、`server-content`；页面级与评论级 `container-task-ui-context` **必须**带 `comment_id`（无评论 id 时前端不发请求，保持 unregistered）。
- 范围内：评论执行细节 `ServerConfigRuntimeStatusSection` 经 `bindCommentRuntimePanel` 绑定 `comment.id`。
- 范围外：空评论列表 fallback 不挂运行态按钮；不改 start-vm 选镜像流程。`comment_id=''` 行仅作任务硬件模板。

## 约束与风险

- 无 `comment_id` → 400「缺少评论ID」，禁止 ForRuntime / 任务级降级。
- 评论卡无参轮询须沿用上次显式 `comment_id`。
- 点击事件不是评论 id，不得当成 `comment_id` 发送，也不得清空后改打任务级。

## 验收标准

1. 评论卡 Workbench / 刷新 / 停止 的 URL 或 body 含 `comment_id=<该评论 id>`。
2. 双 CSC（任务级 `mock-task`、评论 `c2` 为 `mock-cmt`）时，`comment_id=c2` 返回评论实例。
3. `POST stop-vm` 带 `comment_id` 且该评论无 instance → 400「未提供实例ID」，不停任务级机器。
4. 不带 `comment_id` → 400「缺少评论ID」，禁止任务级/ForRuntime。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 评论卡读取/打开该评论 CSC | — | — | — | — | 只读查询与按已有 stop-vm 路径停机；不新增领域事件 |
| 停止评论级云主机 | — | — | `handleStopVmNative`（既有 stop-vm） | 既有 SSE 运行态 | 无新增事件：仅改 CSC 选行，停机发布沿用既有路径 |

## 实施计划

1. `resolveScopedCloudServerConfig`：有 `comment_id` 只读该评论行。
2. 前端 `withCommentIdQuery` / `stopVmBodyWithCommentId` + `bindCommentRuntimePanel`。
3. 回归：Go 双 CSC + stop-vm；Vitest URL/body。

## 变更记录

- 2026-08-13：评论卡 compute 请求锁定评论级 CSC。
- 2026-08-13：与 031 对齐——禁止任务级运行态回退；无 comment_id 不再 ForRuntime。
