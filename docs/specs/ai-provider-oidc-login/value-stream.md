# Value Stream: AI Provider OIDC 登录

> Derived from design: `docs/specs/ai-provider-oidc-login/design.md`

## Value Summary
厂商和平台管理员可通过标准 OIDC 协议（Authorization Code Flow + PKCE）直接登录 AI Provider 镜像市场，无需强制依赖主站 SSO bridge。

## Related Value Streams

- **sso-connection-refused-fix**: 扩展 — 基于已修复的 ai-provider 端口绑定和配置路径，本次在其之上新增 OIDC 登录入口
- **taskauth-oidc-issuer-fix**: 依赖 — taskAuth OIDC Provider 基础设施（issuer 可达性、SSL 修复）是 OIDC RP 的前置条件
- **oidc-sso-404-fix / oidc-slo-logout-sync**: 扩展 — 共享 taskAuth OIDC 基础设施，但 AI Provider 作为独立 RP 接入

## End-to-End Flow

```
用户打开镜像市场 (8010)
  → 点击「OIDC 登录 (厂商)」或「OIDC 登录 (管理端)」
  → 302 重定向到 taskAuth /oauth2/authorize (PKCE + state)
  → taskAuth 认证用户 (已有登录态直接通过，否则走 taskAuth 登录流程)
  → taskAuth 302 回 AI Provider /api/auth/oidc/callback/?code=...&state=...
  → AI Provider 后端: POST taskAuth /oauth2/token (code → id_token)
  → 验证 id_token (RSA 签名 / iss / aud / exp / nonce)
  → 提取 email claim → 匹配 Vendor/PlatformStaff
  → 签发 AI Provider JWT (vendor/staff) → 302 回前端 #token=...
  → 前端存储 token → 用户进入厂商门户/管理端
```

## Value Increments

### Increment 1: OIDC RP 核心 — 后端 authorize + callback (Thin Slice)
**Value to user:** Vendor 或 Admin 可通过 OIDC 获取 AI Provider JWT（需先有 Vendor/Staff 记录）
**Scope:** `oidc_rp.py` + `views_oidc.py` + `urls.py` + `settings.py` 配置
**Depends on:** taskAuth OIDC Provider 运行中

**Steps:**
1. OIDC RP 配置就绪（client_id, secret, issuer, endpoints）
2. `GET /api/auth/oidc/authorize/?role=vendor|admin` — 构建 authorize URL (PKCE + state)，302 到 taskAuth
3. `GET /api/auth/oidc/callback/` — 交换 code → id_token，验证签名和 claims，匹配 Vendor/Staff，签发 JWT

### Increment 2: taskAuth OIDC Client 注册
**Value to user:** OIDC 流程端到端可走通
**Scope:** 在 taskAuth 配置中注册 ai-provider 为 OIDC client
**Depends on:** Increment 1

**Steps:**
1. `conf/auth/task-auth/config.yaml` 新增 bootstrap client
2. taskAuth 重启后识别新 client

### Increment 3: 前端 OIDC 登录入口
**Value to user:** 用户可在 UI 上点击按钮触发 OIDC 登录
**Scope:** VendorPortal.vue + AdminPortal.vue 增加按钮 + callback fragment 处理
**Depends on:** Increment 1, 2

**Steps:**
1. VendorPortal.vue: 添加「通过 OIDC 账号登录」按钮
2. AdminPortal.vue: 添加「通过 OIDC 账号登录」按钮
3. 处理 callback fragment (`#token=...`)

### Increment 4: E2E 测试
**Value to user:** 质量保障
**Scope:** Playwright E2E 测试覆盖 OIDC 登录全链路
**Depends on:** Increment 1, 2, 3
