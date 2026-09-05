# NFR 澄清：gitOauth → taskGitOauth

**日期：** 2026-07-15  
**默认等级：** L2 Standard；认证/凭据相关升至 **L3**

| 类别 | 等级 | 场景 / 度量 |
|------|------|-------------|
| 安全性 | L3 | Fernet 互操作；审计无明文 token；Bridge Secret 校验；日志脱敏 |
| 可用性 | L2 | health 含 DB；进程崩溃可被 runAll 拉起；access 缓存降低 Provider 限流 |
| 性能 | L2 | access-for-user P95 < 2s（缓存命中 < 50ms）；并发 refresh 串行化同 uid |
| 可维护性 | L2 | OpenAPI；与 Python 契约测试对照 |
| 可观测性 | L2 | 结构化日志 + tracelog；换票失败带 provider_key/uid（无 secret） |
| 兼容性 | L3 | URI/JSON/表结构零破坏；存量 SQLite 直接使用 |
| 可移植性 | L2 | modernc.org/sqlite；无 CGO |

## 对领域模型影响

- 需保留 access token 进程内缓存（非持久化实体）
- Credential 聚合绑定状态机不变：pending → active | failed
