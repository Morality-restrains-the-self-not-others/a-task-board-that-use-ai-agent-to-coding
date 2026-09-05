# 多入口同源 API — 实施计划

- **日期**: 2026-07-11
- **设计**: `docs/superpowers/specs/2026-07-11-multi-entry-same-origin-api-design.md`

## Tasks

- [x] **T1** HK nginx：`location /api/` → `183.250.1.132:18081`；保留 `/` → `:4000`；转发 Host/X-Forwarded-*
- [x] **T2** `conf`：`apiBaseUrl` 空；新增 `publicEntryOrigins`（daydaymoney + IP）
- [x] **T3** Vite：`resolveViteApiBaseUrl` 默认同源 `''`；`server.proxy['/api']` → gateway
- [x] **T4** Django CSRF 消费 `publicEntryOrigins`
- [x] **T5** Gateway CORS 追加 daydaymoney origins + 再生 apisix.yaml
- [x] **T6** 验收：curl 经域名 `/api/...` 得 JSON；config.js 中 API_BASE 为空

## 验证命令

```bash
ssh hk 'sudo nginx -t && sudo systemctl reload nginx'
curl -sI --resolve www.daydaymoney.com:443:127.0.0.1 https://www.daydaymoney.com/api/public/system-feature-policy/
# Content-Type 须为 application/json（非 text/html）
```
