# 测试意图：厂商门户 OIDC issuer 公网 gateway

对应：`ai_provider_oidc_issuer_gateway.intent.md`

## T1 — conf 解析 issuer

- **给定** 默认 `conf/base.yaml`（`BASE_DOMAIN=daydaymoney.com`）
- **当** `load_ai_provider_listen`
- **则** `oidc_rp_issuer == https://api.daydaymoney.com`，且不含 LAN IP

## T2 — env 覆盖

- **给定** `OIDC_RP_ISSUER=https://api.example.test`
- **当** `load_ai_provider_listen`
- **则** issuer 为该值

## T3 — authorize 视图 Location

- **给定** GET `/api/auth/oidc/authorize/?role=vendor`
- **当** 返回 302
- **则** Location 含 `/api/oidc/authorize`，不含 `/oauth2/authorize` 与 `183.250.1.132`；`redirect_uri` 为 `https://provider.daydaymoney.com/api/auth/oidc/callback/`

## 测例落点

- `task2app/Saas_Ai_Provider/provider/tests/test_port_config_oidc_issuer.py`
- `task2app/Saas_Ai_Provider/apps/marketplace/tests/test_oidc_views.py`

## 变更记录

- 2026-07-14：初版。
