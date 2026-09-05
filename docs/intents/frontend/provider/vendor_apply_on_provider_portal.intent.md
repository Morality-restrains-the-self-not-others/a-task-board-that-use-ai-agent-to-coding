# 意图：租户镜像市场承载厂商申请认证，厂商门户不再提供申请面板

## 背景与目标

厂商认证申请在 2026-08-17 迁到 `https://provider.daydaymoney.com/`。2026-09-01 诊断现网镜像市场 SSO 缺失后，**入口改回租户镜像市场**，并从门户撤掉申请面板。未入租户者无法申请。SSO 仍仅 `qualified` 或「审核关闭 + 真实邮箱」。

## 范围与边界

- 范围内：taskFE 镜像市场申请四态与 SSO；taskAiProvider 既有 vendor-application API；从 provider SPA 删除申请面板。
- 范围外：不改运营审核、不改证照直传契约、不让 pending 直接 SSO、不新增 Python 接口。

## 约束与风险

- `provider.*` 直连 taskAiProvider，无 APISIX `X-User-Id`；须用主站 `token`/`userId` Cookie 调 taskAuth forward-auth（内部密钥），禁止把 vendor JWT 转发给 taskAuth。
- 无 Cookie 且无网关头时不得探活 taskAuth（避免未认证请求打内部接口）。
- 前端禁止无触发轮询 vendor-status；状态变化由用户点击刷新或提交后重载。
- 申请表错误须带 `data-traceId`；导航用真实 `<a href>`。
- 无邮箱须引导主站个人资料绑定（`/profile/?sso_error=email_required#rg=profile.email_binding`）。

## 验收标准

1. `https://www.daydaymoney.com/tenant/{id}/image-market` 在 none+有邮箱+审核开时显示「申请」CTA；**点击后**才展开申请表单（默认不展示表单）；pending 显示审核中且无 SSO；qualified 显示「厂商门户（SSO）」。
2. `https://provider.daydaymoney.com/` **不再**展示申请认证面板。
3. 已 qualified（或审核关闭且已绑**可投递**邮箱）时镜像市场显示「厂商门户（SSO）」。
4. 合成邮箱（`sso-<id>@sso.invalid`）不渲染 SSO/表单，渲染「绑定邮箱后进入厂商门户」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 提交/重提厂商申请 | — | — | — | — | 与既有 vendor-application-kyc-docs 一致：无 MQ 订阅方，审核在 AdminPortal 同步完成 |
| 查询申请状态 / 发送验证码 | — | — | — | — | 纯查询或复用 taskAuth 既有短信门禁，无新的领域事实 |

## 实施计划

1. taskFE 镜像市场恢复申请四态（对齐 `taskAiProvider/frontend` 申请契约；禁止拷贝过期 Django 表单、禁止 30s 轮询）。
2. 从 `taskAiProvider/frontend` 删除 `VendorApplyPanel` 及门户入口挂载。
3. 复用既有 Go：`vendor-status` / `vendor-application` / upload-url / 用户级短信；不新增 HTTP 接口。
4. 单测：镜像市场四态 + 合成邮箱绑邮箱链；门户无申请面板。

## 变更记录

- 2026-08-17：申请入口从租户镜像市场迁到厂商门户。
- 2026-09-01：现网 SSO 缺失诊断后，申请入口搬回镜像市场并从门户撤掉申请面板（v125）。合成邮箱仍须先绑定。设计：`docs/superpowers/specs/2026-09-01-image-market-sso-link-missing-design.md`。
- 2026-09-04：镜像市场申请表单改为「点申请 CTA 后再展开」，默认仅展示申请按钮，避免首屏堆叠 KYC/短信表单。
