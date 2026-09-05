# 测试意图：创建任务标题翻译失败对用户可读

| # | 场景 | 期望 |
|---|------|------|
| T1 | fanyi 返回 200 空 body | Go 错误含 `响应为空`，不含 `unexpected end of JSON input` |
| T2 | fanyi 客户端超时 | Go 错误含 `超时`/`请求失败`，不含 JSON parse 文案 |
| T3 | conf `max_tokens=4096` | 实际请求 `max_tokens=128`、`stream=false`、`thinking.type=disabled` |
| T4 | 空 body 经 HTTP handler | 502 body 含「响应异常」+ `trace_id`，不含 `fanyi_agent` / JSON parse |
| T5 | 前端收到 JSON parse dump | `taskTitleTranslationError` 为本地规则提示，仍带 `data-traceId` |
| T6 | 前端收到分类原因「未返回可用译文」 | 红字含该原因 +「本地规则」，不含技术字段 |
| T7 | 前端 AbortError | `taskTitleTranslationError` 为空，不改已有工作分支名 |
