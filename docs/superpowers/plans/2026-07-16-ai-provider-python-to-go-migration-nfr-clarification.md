# NFR 澄清：ai-provider Go 迁移

**日期：** 2026-07-16  
**默认级别：** L2 Standard；鉴权相关 L3  

| 属性 | 级别 | 说明 |
|------|------|------|
| 可用性 | L2 | 单进程；健康检查与 Django 同 path |
| 性能 | L2 | 优于单线程 WSGI；proxy 超时 30s |
| 安全 | L3 | JWT/SSO/OIDC 契约不变；禁止日志泄露 secret/token |
| 可观测性 | L2 | tracelog + 入站/出站请求日志 |
| 兼容性 | L3 | API/JSON/ID string 严格兼容 |
| 可维护性 | L2 | DDD 分层对齐 taskGitOauth |
| 数据完整性 | L3 | 同库；不改 schema；事务写审批历史 |
