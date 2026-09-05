# [运行时] 容器 step 日志「错误: Error code: 404」——网关瞬时 404 被当成永久失败

## 现象

任务详情评论执行日志出现：

```text
[17:33:16] step 1: 错误: Error code: 404
```

来源：容器 Trae agent 第一步 LLM 调用失败；`trajectory_recorder.compute_step_delivery_summary` 写成 `错误: {error}`，经 job-stream 推到评论区。无 `data-traceId`（容器 stdout 不是前端请求错误节点）。

示例页：`…/task-detail/task_876757038493888512/`

## 根因

1. OpenAI SDK 把 HTTP 404 格式化为 `Error code: 404`（body 为空时无后续 JSON）。
2. SaaS/APISIX 在上游重启、路由尚未注册时常返回空 404 / `404 page not found` / `404 Route Not Found`，与「模型不存在」不是同一类错误。
3. `retry_utils._should_retry_api_error`（HEAD）把 **全部 4xx 除 429** 视为永久失败，因此网关 404 **不重试**，`BaseAgent` 立刻 `step.error` 并结束 job。
4. 同类缺口：`saasPostJson` 对 JSON body 的 404/502 曾写入 `structuredPayload` 从而跳过瞬时重试。

## 修复

- LLM：`404/408/409/425/429` 视为瞬时；仅 `model_not_found` / `the model \`` 等标记为永久 404。
- 容器→SaaS POST：`TRANSIENT_HTTP_STATUS` 含 404/5xx，JSON 网关页也重试。

## 验证

```bash
cd trae-agent && .venv/bin/python -m unittest tests.utils.test_retry_utils -v
cd trae-agent/onlineServiceJS && node --test src/saasTaskCloud.transientRetry.test.mjs
```

现网还需推镜像并重建任务容器（见 OPT-20260816-048）。
