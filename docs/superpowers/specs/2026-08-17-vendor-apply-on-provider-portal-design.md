# 2026-08-17 vendor apply on provider portal — design

- **Status:** accepted (goal-mode auto)
- **Date:** 2026-08-17

## Context

认证申请挂在租户镜像市场，申请主体却是主站账号。迁到 `provider.*` 与门户职责对齐。

## Decision

1. 厂商门户未登录卡片增加「申请认证」四态 UI（`VendorApplyPanel`）。
2. `provider.*` 用主站 Cookie 调 taskAuth forward-auth 解析申请人（无 Cookie 不探活）。
3. 用户级 phone-status / send-sms / verify-phone，不再依赖 tenantId。
4. 镜像市场删除申请/审核中/驳回 UI 与 30s 轮询；保留 qualified / 审核关闭时的 SSO。

## Alternatives Considered

- CORS 调 www `/api/ai-provider/*`：需改网关 allow_origins，且手机验证仍绑 tenant 路径。
- 仅做跳转链接到主站独立申请页：用户要求入口在 provider 页面上。

## Consequences

- 正面：无租户用户也可申请；镜像市场只做安装。
- 负面：taskAiProvider 增加对 taskAuth forward-auth 的依赖（已有 SMS gate 同类调用）。
