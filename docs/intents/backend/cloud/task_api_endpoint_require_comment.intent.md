# 意图：TaskApiEndPoint 必须含 /comment/{cid}/

## 背景与目标

`docs/skills/saas-container/saas-machine-container.md` 曾写：无 `commentId` 时兼容旧值 `…/task/{taskId}/cloud`。CSC 键是 `(task_id, comment_id)`，无 cid 的前缀无法定位评论级容器。

目标：生成、注入、容器解析 **一律** 使用

`https://<api>/api/tenant/{tid}/workspace/{wid}/task/{taskId}/comment/{cid}/cloud`

禁止再写出或把无 cid 的 `…/task/{taskId}/cloud` 当作合法 `TaskApiEndPoint` / `TASK_API_ENDPOINT`。

## 范围与边界

- 范围内：UserData 占位符 `__TASK2APP_TASK_CLOUD_PREFIX__`；relay `expandEnvForRuntime` / `cloudTokenAPIPrefix` / `statusPushURL`；容器 `buildTaskCloudPrefix` / `taskApiPrefix`；SaaS HTTP inbound（网关 `container-inbound-token`、`parseCloudInboundPath`、`parseContainerAPIPath`、token init）；skill 文档。
- 范围外：浏览器 compute `/cloud/compute/`（ADR-0010 kv `comment_id`）。

## 约束与风险

- 无有效 cid（空或 `-`）：不生成旧 URL；前缀为空或报错。
- 旧 URL 仅可当作 origin + tenant/workspace/task 解析源；有 `COMMENT_ID` 时重建新前缀，无 cid 则失败。
- 路径已带 `comment_id` 分片键；NFR：L1，禁止无 cid 的 TaskApi 前缀。

## 验收标准

1. 有 cid 时前缀必含 `/comment/{cid}/cloud`。
2. 无 cid 时不出现 `…/task/{id}/cloud`（无 comment 段）。
3. 容器 `taskApiPrefix()` 无 cid 抛错。
4. skill 文档不再写「兼容旧值」。
5. 无 `/comment/{cid}/` 的容器 inbound URL 解析失败 / HTTP 404。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 收紧 TaskApiEndPoint 契约 | — | — | UserData / env 生成 | 容器出站前缀 | 只读契约收紧，无新领域事实 |
