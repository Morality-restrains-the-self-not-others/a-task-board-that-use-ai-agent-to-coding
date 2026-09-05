# 登录后未验证手机号弹窗引导 — 设计

- **Date:** 2026-08-25
- **Status:** accepted（/goal 自动采用）
- **Iteration:** login-phone-verify-prompt

## Context

用户在 `/auth/login/` 登录后可直接进入工作面板/onboarding，微信/邮箱账号常未绑定手机。支付与安全校验已有 `PhoneVerificationGate`，但登录当下没有引导，用户不知道要去个人资料绑定。

现有能力：

- 登录响应 `user.login_methods[]` 含 `method_type` / `is_verified`
- `GET /api/accounts/users/profile/` 返回 `has_phone` / `phone_masked`（`has_phone` = 存在未作废 phone login_method）
- 个人资料 `UserProfilePhoneBindingPanel` 已实现绑定+短信验证
- 邮箱引导先例：`/profile/?sso_error=email_required#rg=profile.email_binding`

## Architecture understanding

根据 `docs/architecture/VERSION_HISTORY.md`：最新已设计版本为 v107（镜像/技能 ID 化，target）。认证边界仍是 taskFE SPA → taskAuth（登录/profile/bind-phone）。**本期不新增服务、表、协议或 MQ**，不发布新架构视图。

## Decision

登录成功、凭据落盘之后、`window.location.href` 跳转之前：

1. 判定是否已验证手机（见下方谓词）。
2. 未验证 → `modalService.alert`：标题「请验证手机号」，仅「去验证」（无「稍后再说」、无关闭叉）。关闭/拒绝仍去绑定页。
3. 「去验证」→ `/profile/#rg=profile.phone_binding`；原 next 写入 sessionStorage，绑定成功后回跳。
4. 已验证 → 原 `resolvePostLoginRedirectUrl` 落点。
5. **硬门禁**：未验证不能进工作台等业务页；`PhoneVerifyAccessGate` 在已登录会话上阻断非豁免路由。

覆盖入口：客户密码/令牌登录（`useLoginSubmit`，`adminLogin=false`）与微信回调（`Login.vue` `handleWeChatCallback`）。

### 已验证谓词

```
userHasVerifiedPhone(source):
  if source.login_methods is array:
    return some(m => m.method_type === 'phone' && m.is_verified === true)
  if typeof source.has_phone === 'boolean':
    return source.has_phone === true
  return false
```

绑定路径只在短信验证成功后写入 `is_verified=1` 的 phone method，故 `has_phone` 与「已验证」在现网等价。微信回调只有 profile，用 `has_phone`。

### 跳过弹窗

- `adminLogin === true`
- `loginMethod === 'phonePassword'`
- 模拟登录（`sessionStorage.impersonatorAccountBackup`）
- 谓词为 true
- 判定异常：fail-open 并打 `[login-phone-verify]` 日志

### 不采用的方案

| 方案 | 拒绝原因 |
|------|----------|
| 登录页内嵌完整绑定表单 | UserProfilePhoneBindingPanel 已承担绑定；弹窗内再做一套易漂移 |
| 拦截直至绑定（硬门禁） | 需求是「引导」；OIDC next / onboarding 不能被永久卡住 |
| 新 taskAuth API `phone-verification-status` | 登录响应与 profile 已够用 |
| 复用计费 `PhoneVerificationGate` | 依赖 tenant billing 状态与支付 SMS，登录时可能无租户 |

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 已执行（16 files，risk 0.30）。
- `codegraph query useLoginSubmit`：定义 `taskFE/app/src/composables/auth/useLoginSubmit.js`；调用方 `Login.vue`、`AdminLogin.vue`。
- 微信路径不经过 `useLoginSubmit`，必须在 `handleWeChatCallback` 单独挂钩，否则扫码登录漏提示。
- `resolveWechatSessionIdentity` 已 GET profile，可把 `has_phone` 带回调用方，避免二次请求。

## Python 新 API 门禁

not_applicable — 无 Django/Flask 新路由。

## 架构变更

无。不新增 `.puml` / `.archimate`。

## Observability

前端：`[login-phone-verify] action=prompt|skip|verify|redirect reason=...`（不打印手机号明文）。不新增后端埋点。
