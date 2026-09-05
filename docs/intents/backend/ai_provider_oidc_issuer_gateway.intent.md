# 厂商门户 OIDC 登录授权跳转须使用公网 gateway issuer

## 意图

从镜像市场「厂商门户（SSO）」进入 `provider.*` 后，点击「通过 OIDC 账号登录」时，浏览器应跳转到 **taskAuth OIDC Discovery 中的 `authorization_endpoint`**（公网 `https://api.<baseDomain>/api/oidc/authorize`），不得跳到 LAN IP（如 `http://183.250.1.132:18081`）或错误路径 `/oauth2/authorize`。

## 背景

- taskAuth `oidc.issuer` = `${subdomains.gateway}`（`conf/base.yaml` → `https://api.daydaymoney.com`）
- Discovery：`authorization_endpoint` = `{issuer}/api/oidc/authorize`
- 缺陷：ai-provider `OIDC_RP_ISSUER` 硬编码 `http://183.250.1.132:18081`，且 RP 默认路径为 `/oauth2/*`

## 验收标准

1. `OIDC_RP_ISSUER` 来自 env > `conf/ai/ai-provider` `oidcRpIssuer` > `subdomains.gateway`，默认解析为 `https://api.daydaymoney.com`（可用 `BASE_DOMAIN` / `PUBLIC_SCHEME` 覆盖）
2. authorize 302 `Location` 以 `https://api.<baseDomain>/api/oidc/authorize?` 开头
3. `Location` 不含 `183.250.1.132`、不含 `/oauth2/authorize`
4. token / JWKS 端点与 Discovery 一致：`/api/oidc/token`、`/api/oidc/jwks`
5. `redirect_uri` 为 `https://provider.<baseDomain>/api/auth/oidc/callback/`（与 taskAuth bootstrapClients 一致；禁止边缘 TLS 下发出 `http://`）



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：OIDC issuer URL 配置，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 厂商门户 OIDC 登录授权跳转须使用公网 gateway issuer | — | — | — | — | OIDC issuer URL 配置，无新增业务事件 |
## 变更记录

- 2026-07-14：修复硬编码 LAN issuer 与错误 `/oauth2` 路径，对齐 gateway Discovery。
