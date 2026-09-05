# Review: 登录后未验证手机号弹窗引导

- Date: 2026-08-25
- Plan: 2026-08-25-login-phone-verify-prompt-plan.md

## CRG

`code-review-graph query` 确认 `useLoginSubmit` 调用方为 Login.vue / AdminLogin.vue；微信路径已单独挂钩 `finishWechatCallbackLogin`。无遗漏调用链。

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | 谓词覆盖 login_methods / has_phone；未验证 confirm→资料深链；稍后/已验证/管理员/模拟登录跳过；微信 fail-open |
| Readability | 深链常量与 email 同构；日志前缀 `[login-phone-verify]` |
| Architecture | 无新 API/事件；复用 profile bind-phone 与 modalService |
| Security | 不打手机号；模拟登录跳过；深链固定 `/profile/` 无他人 user id |
| Performance | 密码登录零额外 RTT；微信复用已拉 profile |

## 安全审计

- [x] 无密钥进代码/日志
- [x] 无新用户输入写路径（绑定仍走存量 SMS）
- [x] 无 SQL
- [x] 无新 XSS 面（文案常量 + 既有 Modal）
- [x] 认证后才读 login_methods / profile
- n/a CORS/CSP/新速率限制

## Intent→Event

书面例外：弹窗纯 UX，无 MQ。绑定沿用存量 bind-phone。

## Log Audit

成功路径：`action=skip|verify|redirect` + reason。无 PII。

## 问题

Critical: 0 / Required: 0 / Nit: 绑定成功后可从 sessionStorage 回到原 next（记 OPT）
