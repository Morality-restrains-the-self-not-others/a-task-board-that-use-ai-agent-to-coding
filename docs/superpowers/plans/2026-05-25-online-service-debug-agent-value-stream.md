# Value Stream: onlineServiceJS DEBUG_AGENT 调试链路

> Derived from design: `docs/superpowers/specs/2026-05-25-online-service-debug-agent-design.md`

## Value Summary

开发与排障人员可以在 `DEBUG_AGENT=True` 的调试模式下，获得 onlineServiceJS 出入站请求的完整上下文，快速定位 relay 直启与跨服务调用问题。

## End-to-End Flow

用户以 `relayToTrae=true` 进入任务详情并点击直接启动  
→ 前端组装默认 env（含 `DEBUG_AGENT=True`）  
→ relayToTrae 将 env 透传到 onlineServiceJS  
→ onlineServiceJS 处理入站请求并发起出站调用时记录完整请求/响应  
→ 调试人员从日志直接定位问题并得到可执行修复线索。

## Value Stages

- Trigger: 用户在 TaskDetail 中走 `relayToTrae=true` 直接启动路径
- Stage 1 (Core value): 启动链路默认注入并透传 `DEBUG_AGENT=True`
- Stage 2 (Core value): onlineServiceJS 记录入站请求完整上下文
- Stage 3 (Core value): onlineServiceJS 记录出站请求完整上下文
- Stage 4 (Essential support): 单测/E2E 覆盖开关与透传契约
- Delivery point: 排障人员可在单次请求生命周期内看到 method/url/headers/body 全量证据

## Wait / Dependency Points

- 依赖 relay 启动请求中的 env 组装与透传一致性
- 依赖 onlineServiceJS 主进程未被 `DEBUG_AGENT` 日志写入逻辑阻塞
- 依赖测试用例覆盖默认值与透传契约，防止回归

## Value Increments

### Increment 1: relay 直启默认开关 (Thin Slice)
**Value to user:** 使用 `relayToTrae=true` 时无需手工配置即可启用调试模式。  
**Scope:** 前端默认 env 增加 `DEBUG_AGENT=True`，并透传到 relay start payload。  
**Depends on:** nothing

### Increment 2: inbound debug visibility
**Value to user:** 调用 onlineServiceJS 的请求与响应可以完整追踪。  
**Scope:** `server.mjs` 增加 debug 中间件记录入站请求/响应完整字段。  
**Depends on:** Increment 1

### Increment 3: outbound debug visibility
**Value to user:** onlineServiceJS 对外调用（SaaS/GitHub/LLM）请求与响应完整可观测。  
**Scope:** 对主要 `fetch` 路径补全 debug 日志，记录 method/url/headers/body 全量信息。  
**Depends on:** Increment 1

### Increment 4: regression guard
**Value to user:** 后续改动不会破坏默认注入与日志契约。  
**Scope:** 更新 `relayToTraeUtils` 单测与 TaskDetail 直启 Playwright 断言。  
**Depends on:** Increment 1, 2, 3
