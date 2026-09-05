# 多入口同源 API — NFR

- **日期**: 2026-07-11
- **默认级别**: L2；认证相关 CSRF/Host **L3**

| 属性 | 级别 | 说明 |
|------|------|------|
| 安全 | L3 | CSRF Trusted Origins / ALLOWED_HOSTS 显式名单；禁 credentials+`*` |
| 可用性 | L2 | 边缘分流失败时 `/api` 不得回落到 SPA HTML |
| 兼容 | L2 | 多域名+IP；本地 Vite 代理兼容 |
| 性能 | L1 | 额外一跳边缘反代，可接受 |
| 可观测 | L2 | nginx access + 既有 gateway 日志 |

## 对领域模型影响

无新聚合；配置对象 `PublicEntryOrigin` 为运维配置。
