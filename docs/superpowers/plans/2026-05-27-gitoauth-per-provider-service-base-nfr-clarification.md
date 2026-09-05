# NFR 澄清: gitOauth 按 service_provider 路由

> 输入: 设计文档 + value-stream

| 类别 | 等级 | 量化目标 |
|------|------|---------|
| 可用性 | L2 | gitlab-local connection GET 在公网 gitOauth 502 时仍 200 |
| 数据一致性 | L2 | 每个 provider_key 100% 命中对应 service_base（单测断言 URL host） |
| 可维护性 | L2 | 移除 DJANGO_GITOAUTH_BASE；配置单一来源为 gitOauth.*.service.allowedHost |
| 安全性 | L2 | 不变；仍经 bridge secret 调 internal API |
