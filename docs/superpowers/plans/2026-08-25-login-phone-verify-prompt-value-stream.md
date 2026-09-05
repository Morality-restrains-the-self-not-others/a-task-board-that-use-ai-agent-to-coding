# 价值流：登录后未验证手机号弹窗引导

- 日期：2026-08-25
- 关联设计：2026-08-25-login-phone-verify-prompt-design.md

## Related Value Streams

- `2026-08-24-remove-phone-code-login`：登录只保留手机号+密码；本期不恢复验证码登录。
- `2026-07-15-login-phone-otp-enable`：历史 OTP 入口，已被上条移除。
- 邮箱 SSO 引导（`emailBindingDeepLink`）：同构「弹窗/跳转 + `#rg=` 定位」；本期手机绑定复用该深链模式。

Greenfield UX 增量，叠在既有登录成功跳转之后。

## 用户旅程（目标态）

```
访客 ─登录成功─► 凭据落盘
                    │
                    ├─ 已验证手机 / 手机密码登录 / 管理员 / 模拟登录
                    │      └─► 原 redirect（工作面板 / onboarding / next）
                    │
                    └─ 未验证手机
                           └─► 不可跳过提示（仅去验证）
                                 └─► /profile/#rg=profile.phone_binding → 绑定短信 → 原 next
                     （书签进业务页且未验证 → 阻断层仅去验证）
```

## 价值主张

1. 微信/邮箱用户未验证手机不能进工作台等业务页。
2. 绑定成功后回到原 next / 工作面板。
3. 绑定 UI 仍只有个人资料一处，避免第三套表单。

## 增量

| ID | 增量 | 验收 |
|----|------|------|
| VS-1 | 谓词 + 深链 | 单测覆盖 verified / unverified / has_phone |
| VS-2 | 密码登录挂钩 | 未绑定 confirm；已绑定不 confirm |
| VS-3 | 微信回调挂钩 | 无手机弹窗；有手机直跳 |
| VS-4 | 资料页定位 | `data-rg-key=profile.phone_binding`，加载后可滚到绑定区 |

## 反价值

- 每次登录都提示可能烦人 → 绑定后自动消失；允许稍后。
- 多一次 confirm 打断 OIDC → 稍后仍走原 next。
