# DDD 建模：登录后未验证手机号弹窗引导

- 日期：2026-08-25
- 关联 NFR：2026-08-25-login-phone-verify-prompt-nfr-clarification.md

## 限界上下文

仍在 **Identity / taskAuth + taskFE Auth UX**。不跨租户、不计费。

## 领域概念

| 概念 | 类型 | 落点 |
|------|------|------|
| PhoneVerifiedStatus | Value Object | `taskFE/app/src/domain/auth/value_objects/phone_verified_status.js` |
| PostLoginPhoneVerifyDecision | Value Object | `{ action: 'redirect'\|'verify', href }` |
| promptPostLoginPhoneVerify | Domain Service | 纯函数：skip + 谓词 + 用户选择 → decision |
| PhoneBindingDeepLink | 契约常量 | `profile.phone_binding` + `/profile/#rg=...` |

后端 `LoginMethod`（phone, is_verified）已存在，本期只读。

## 领域事件（书面例外）

**纯前端引导，不产生新业务事实，无 MQ 投递。**

绑定成功沿用存量 bind-phone（不在本期改 publish）。

## 端口

无新仓储/事件总线端口。UI 适配器：`modalService.confirm`、`window.location.href`。

## 结论

骨架即上述 JS 值对象 + 服务；基础设施（API）零变更。
