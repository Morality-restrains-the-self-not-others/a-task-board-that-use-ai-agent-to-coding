# 意图：compute 转发 comment_id 进 path

## 背景与目标

评论级 CSC 要求每次容器 compute 调用带 `comment_id`。此前放在 query，与 kv-last path 约定不一致，网关不解析该键。目标：浏览器 URL 使用 `/comment_id/{id}/`，query 不再带 `comment_id=`。

## 范围与边界

- 范围内：taskFE compute helper、网关 `parseContainerComputePath`、Cloud `commentIDFromComputeRequest`、`ParseConventionPath` 识别 `comment_id`。
- 范围内：规整 SaaS inbound skill（原 Django `machine_container.md`）。
- 范围内：容器 → SaaS `TaskApiEndPoint` 推荐 `…/task/{taskId}/comment/{cid}/cloud`（位置段）；网关 / AgentSupport / Credential / UserData 前缀与 `taskApiPrefix()` 对齐。
- 范围外：内部 `container-target` query。

## 验收标准

1. 执行日志 / 层图 / job redo / 文件树 / Workbench / runtime-status URL 含 `/comment_id/{cid}/`，且不含 `[?&]comment_id=`。
2. 网关解析 func-first 与 kv-last 两种带 `comment_id` 的 path。
3. 无 `comment_id` 时前端不发评论级 compute 请求。
4. `docs/skills/saas-container/` 可定位容器 inbound 与 SaaS inbound 两份 skill。
5. 有 `commentId` 时 UserData / `taskApiPrefix()` 产出 `…/task/{taskId}/comment/{cid}/cloud`；APISIX 与 `parseCloudInboundPath` **只**接受该形态，拒绝无 comment 段的 `/task/{id}/cloud/`。
