# 设计文档：大流量护栏（开发期落地 + 上线前清单）

**日期：** 2026-07-15  
**状态：** 已采纳（goal-mode 自动决策，开发模式）  
**动机：** 静态分析识别的容量风险；当前为开发环境，优先加护栏且不阻塞本地开发体验；结构性改造登记为上线前项。

## 架构影响

**无拓扑变更**（不新增服务/不改表所有权边界），故**不**新增 `docs/architecture/` vN ArchiMate 视图。变更落在既有 APISIX codegen、各 Go SQLite 打开路径、taskSSE、taskAuth、taskAIComment、go_relayToTrae。

## NOW（本迭代实现）

| ID | 项 | 验收 |
|---|---|---|
| N1 | APISIX 接线 `rateLimit.globalPerMinute` → global_rules `limit-req` | codegen 后 `global_rules` 含 limit-req；单测断言 |
| N2 | Go 域库统一 WAL + busy_timeout=30s + MaxOpenConns≤4 | project/task/cloud/ai-comment/credential |
| N3 | Django `sqlite_pragmas` busy_timeout 与 OPTIONS 对齐为 30000ms | pragma + 既有测试更新 |
| N4 | taskSSE 全局连接上限（默认 500，可配） | 超限 503；单测 |
| N5 | AI instruct chunk SSE 批处理（~80ms / 2KiB） | error/done 立即刷；单测或逻辑测 |
| N6 | forward-auth 结果短 TTL 缓存（默认 2s） | 同 token 重复请求少打 DB；单测 |
| N7 | relay `lifecycleMu` 获取超时 → 503（默认 30s，dev 可配） | 并发 start 不再无限挂起 |
| N8 | Intent + 上线前清单文档 | `docs/intents/backend/high_traffic_hardening.*` |

## PRE-PROD（上线前再做，本迭代只登记）

| ID | 项 | 原因（开发期暂缓） |
|---|---|---|
| P1 | saas / auth / bill 等迁 PostgreSQL | 改动面大，需迁移与 CI |
| P2 | Redis Session / 去掉 Django session/resolve 热路径 | 登录态迁移 |
| P3 | 多 go_relay 分片（打破单 OSJS） | 需部署拓扑 |
| P4 | 通用 circuit breaker + 云 API 客户端超时统一 | 需压测基线 |
| P5 | Redis/Kafka 高可用 | 基础设施 |
| P6 | 存量 django-legacy 热路径继续迁 Go | 既有拆分节奏 |

## 决策摘要

1. **开发期限流**：`globalPerMinute` 默认 3000，足够本地联调；login 仍保留更严 limit。
2. **relay 不分 task 锁**：单进程共享一个 onlineServiceJS，全局 `lifecycleMu` 正确；改为超时快速失败。
3. **SSE 上限 500**：本地多开标签不会先打满 FD；生产可调高。
4. **Auth 缓存 2s**：降低 forward-auth 对 SQLite 的放大，注销延迟可接受。

## 变更记录

- 2026-07-15：初版 — NOW/PRE-PROD 分流并落地 N1–N8。
