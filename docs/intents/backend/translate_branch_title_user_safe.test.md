# 测试意图：translate-branch-title 上游失败用户可读

| # | 场景 | 期望 |
|---|------|------|
| T1 | 200 空 body | `translateTitleWithFanyiAgent` 报 `响应为空` |
| T2 | 客户端超时 | 不出现 `unexpected end of JSON input` |
| T3 | max_tokens 封顶 | 请求体 128，且 `thinking.type=disabled` |
| T4 | handler 空 body | 502 含「响应异常」+ `trace_id`，不含 `fanyi_agent` / JSON parse |
| T5 | 非中文标题 | 200 `used_ai=false`（回归） |
| T7 | 200 截断 JSON | 不含 `unexpected end of JSON input` |
| T8 | 超时下限 | `fanyiTitleTimeout` ∈ [12s, 20s] |
| T9 | 200 + 空 content + finish_reason=length | 日志含 `finish_reason=length`；502 用户句含「未返回可用译文」，不含 finish_reason |
| T10 | 配置不完整 | 502 含「暂未就绪」 |
