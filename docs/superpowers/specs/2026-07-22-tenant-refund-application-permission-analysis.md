# 权限分析：租户退款申请

- 日期：2026-07-22
- 设计：`2026-07-22-tenant-refund-application-design.md`

## 角色 × 操作

| 操作 | 匿名 | 租户成员 | 租户管理员 | is_staff | is_superuser |
|------|------|----------|------------|----------|--------------|
| 查看本租户退款申请 | ❌ | ✅（只读本租户） | ✅ | ❌* | ✅（全量列表） |
| 提交退款申请 | ❌ | ❌ | ✅ | ❌ | ❌（走超管审批页） |
| 审批通过/驳回 | ❌ | ❌ | ❌ | ❌ | ✅ |
| 直接改余额/调支付退款 | ❌ | ❌ | ❌ | ❌ | 仅经 approve 状态机 |

\* staff 不开放审批；与价格套餐一致，仅 superuser。

## 数据边界

- 租户 API：强制路径 `tenant_id` 与 JWT/会话租户一致（既有 billing proxy 模式）
- 申请仅冻结本账户；不可跨租户
- 审批 API：Django `is_superuser` 后转发 internal；taskBill 校验 internal secret

## CRG 触点

- `requireTenantAdmin`（GitLab 购买同构）
- `pricing_views.system_admin_*` 超管模式
- `BillingProxyView` 鉴权头 `X-User-Id`

## 审计

- 申请/审批写 `applicant_user_id` / `reviewer_user_id` + 事件 outbox
- 支付退款 refs JSON 存 application，日志脱敏 provider_ref
