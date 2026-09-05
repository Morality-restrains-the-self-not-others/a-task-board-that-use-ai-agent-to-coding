# 大流量护栏 — 测试意图

| ID | 场景 | 期望 |
|---|---|---|
| T1 | codegen 含 global limit-req | `test_taskgateway_routes_codegen` 断言 global_rules |
| T2 | Django sqlite pragma | `test_sqlite_pragmas` busy_timeout=30000 |
| T3 | SseHub/server 超限 | taskSSE 单测：达到上限拒绝新连接 |
| T4 | forward-auth 缓存 | taskAuth：二次请求不重复查失效 token 路径可测缓存命中日志或计数 |
| T5 | AI chunk batcher | 多 chunk 合并后再 publish（单元） |
| T6 | relay lifecycle 忙 | 持锁时 start 超时返回 503 |

## 变更记录

- 2026-07-15：初版。
