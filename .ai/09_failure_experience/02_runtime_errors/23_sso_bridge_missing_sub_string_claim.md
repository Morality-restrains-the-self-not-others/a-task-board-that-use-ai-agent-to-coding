# [运行时] 厂商门户 SSO 换票报「bridge 缺少 sub」

## 现象

主站 `…/tenant/:id/image-market` 点击「厂商门户（SSO）」后跳转 `provider.*`，页面/接口提示 **`bridge 缺少 sub`**（换票 `POST /api/auth/sso/exchange/` 400）。

## 环境与上下文

- 主站签发：`task2app/Saas_project/accounts/sso_bridge_token.py` → `"sub": str(subject_id)`
- Provider 换票：`taskAiProvider/infrastructure/store_auth.go` → `ExchangeBridge` → `claimInt64(payload["sub"])`
- 规范：`.ai/03_technical_implementation/11_id_field_string_transit.md`（Snowflake ID 进程间必须 string）

## 根因

主站按 ID 字符串传输规范签发 **string `sub`**；Go `claimInt64` 仅接受 `float64`/`int`/`json.Number`，**不解析 string**，于是误报「缺少 sub」（claim 实际存在，只是类型不被识别）。

## 修复

- `taskAiProvider/infrastructure/config_jwt.go`：`claimInt64` 增加 `string` → `strconv.ParseInt`
- 单测：`TestSSOExchangeStaffStringSub` / `TestSSOExchangeVendorStringSub`
- E2E：`playwright/saas_ai_provider/tests/port8010-vendor-sso-bridge-sub-cdp.mjs`（及配套 `.sh`）

## 预防

- JWT / 跨服务 claim 中的业务 ID 按 string 签发时，解析侧必须同时接受 string（入库前再转 int64）
- SSO / bridge 回归测例须覆盖 **string `sub`**（与主站签发一致），勿只用 JSON number 的 float64 fixture
