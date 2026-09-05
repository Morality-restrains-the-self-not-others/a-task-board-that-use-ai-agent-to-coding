# 意图：容器转发目标解析评论级 CSC

## 背景与目标

任务详情项目文件树报「容器尚未注册可用业务地址，请先完成启动与 exchange-refresh」，但同页「打开容器开发页面」可打开。

根因（traceId `c8049995-5d9c-4b6f-ba47-a6c35412990c`）：

1. 文件树 `GET container-layer-children` 经 gateway 调内部 `container-target`。
2. `handleInternalContainerTarget` 只读**任务级** CSC（`comment_id=''`），该行常是空模板（无 `server_url` / `business_api_endpoint`）→ 409。
3. `container-task-ui-context` / VS Code 链接走**评论级** CSC，已登记 `http://<public_ip>:8765` 与 `container_vscode_url`。

目标：SaaS→容器转发与 UI 上下文使用同一套 CSC 解析；有 `comment_id` 时只读该评论行；无 `comment_id` 时不得用空任务级模板挡住已注册的评论级地址。

## 范围与边界

- 范围内：`taskCloudService` `container-target` / `lookup` / `validate-ai-comment-post` 内部接口；gateway 转发 `comment_id`；`ensure-client-ingress` 无评论 ID 时的 CSC 回退；**taskFE 所有走网关的 `container-layer-*` / `container-job-*` / `container-auto-run-steps` / `container-bootstrap-clone-log` 须显式带 `comment_id`（GET query + POST body）**。
- 范围外：不改浏览器直连容器的 VS Code URL 生成；不改变评论级 CSC 写入路径。

## 约束与风险

- 传入 `comment_id` 时禁止回退其它评论（与 `resolveScopedCloudServerConfig` 一致）。
- 无 `comment_id` 时回退「最新带 instance / 已登记地址的评论级 CSC」，与 workbench-link / runtime 一致；多评论并行时前端应传 `comment_id`。

## 验收标准

1. 任务级 CSC 地址为空、评论级已登记 `server_url` → 无 `comment_id` 的 `container-target` 返回 200 且 `base_url` 来自评论级。
2. 两个评论级 CSC 不同地址时，`comment_id` 命中对应行的 `base_url`/`access_token`。
3. 任务级已有地址时，既有 `container-target` 测例仍 200。
4. 文件树不再因空任务级模板误报「尚未注册可用业务地址」（评论级已登记时）。
5. 两评论绑不同实例时，层图 / git commit / layer-changes 等转发请求带 `comment_id`，命中对应 CSC token（不得打到「最新有地址」的另一台）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 解析容器转发目标 | — | — | — | — | 无对应事件：只读查询 CSC 地址，无业务状态变更 |

## 实施计划

1. 抽取 `loadCloudServerConfigForGatewayForward`：`csc_id` / `comment_id` 优先，否则评论级可达地址，最后才任务级模板。
2. `handleInternalContainerTarget` 与 lookup 默认分支改用该函数。
3. gateway `container-target`/`lookup` 透传 `comment_id`。
4. 回归单测覆盖空任务级 + 评论级已登记、以及 `comment_id` 精确选择。

## 变更记录

- 2026-08-15：修复文件树 409 与「打开容器开发页面」不一致（评论级 CSC vs 任务级空模板）。
- 2026-08-15：OPT-20260815-021 — 其余 container-layer / container-job / auto-run-steps 转发显式带 `comment_id`。
