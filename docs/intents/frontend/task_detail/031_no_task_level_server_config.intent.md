# 意图：去掉任务级 ServerConfig，运行态禁止回退

## 背景与目标

评论卡片按钮已能带 `comment_id`，但后端在缺失时仍走 `loadCloudServerConfigForRuntime`（任务级有 instance 则用之），前端 fallback 面板仍发无 `comment_id` 的任务级请求。双行 CSC 导致点错机器。

目标：运行态 ServerConfig **只有评论级**。`comment_id` 必填；禁止任何回退到 `comment_id=''` 任务级行的降级。

## 范围与边界

- 范围内：`workbench-link`、`server-runtime-status`、`stop-vm`、`server-content`、`container-task-ui-context`、`repo-reclone`；评论区运行态按钮；删除 `loadCloudServerConfigForRuntime`。
- 范围内：容器 inbound 写可达性/心跳须定位评论 CSC（`comment_id` 或从 `container_name` 解析），不得写入任务级行。
- 范围外：`comment_id=''` 行仅作**任务硬件模板**（ensure 评论 CSC 时复制 platform/region/auth），不作为运行实例；不在本意图删除该模板行或改 start-vm 选镜像。

## 约束与风险

- 缺 `comment_id` → 400「缺少评论ID」，不得 ForRuntime / 最新带 instance 行。
- 有 `comment_id` 但无该评论 CSC → 按接口既有 404/200 空态，仍不得改读任务级。
- 前端无评论 id 时不发 compute 请求；空评论 fallback 不挂运行态按钮。

## 验收标准

1. `GET workbench-link|runtime-status|server-content` 无 `comment_id` → 400「缺少评论ID」。
2. 任务级行有 instance、评论行有另一 instance 时，带 `comment_id` 只返回评论实例。
3. `POST stop-vm` 无 `comment_id` → 400；有 `comment_id` 且评论无 instance → 400「未提供实例ID」，不停任务级机器。
4. 源码不再存在 `loadCloudServerConfigForRuntime`。
5. 评论卡按钮 URL/body 必含 `comment_id`；无 id 不请求。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 按评论读取运行态 CSC | — | — | — | — | 只读查询 |
| 停止评论级云主机 | — | — | 既有 stop-vm | 既有 SSE | 无新增事件，仅改选行 |

## 实施计划

1. `resolveScopedCloudServerConfig` 无 comment_id 返回错误；删除 ForRuntime。
2. 上述 handler 缺 comment_id 返回 400。
3. inbound 禁止回退任务级行。
4. 前端必带 comment_id；去掉 fallback 任务级运行态。

## 变更记录

- 2026-08-13：禁止任务级运行态 ServerConfig 与任何回退。
