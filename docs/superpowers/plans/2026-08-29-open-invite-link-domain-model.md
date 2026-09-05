# 开放式邀请链接 — 领域模型

- **日期**: 2026-08-29
- **限界上下文**: Tenant / Membership（taskTenantService）

## 聚合

**Invitation**（根：`tenant_invitation`）

- 身份：`id`（Snowflake）
- 属性：`company_id`, `workspace_id`, `invite_method`, `invite_target`, `invitation_token`, `expires_at`, `is_accepted`, `link_kind` (`single`|`open`), `max_uses`, `use_count`, `company_member_name`, `pending_grants`, `pending_role_names`, `is_admin`
- 不变量：
  - `single` ⇒ `max_uses=1`；首次成功 join 后 token 清空且 `is_accepted=1`
  - `open` ∧ `max_uses=0` ⇒ 仅过期或 revoke 结束
  - `open` ∧ `max_uses>0` ⇒ `use_count <= max_uses`；达到则同 single 结束
  - email/phone 禁止 `open`

**InvitationRedemption**（实体，表 `tenant_invitation_redemption`）

- `id`, `invitation_id`, `company_id`, `user_id`, `member_id`, `created_at`
- 不变量：同一 invitation 同一 user 至多一条

**CompanyMember**（既有）— UNIQUE(user_id, company_id)

## 领域服务

- `CreateInvitation(cmd)` — 校验 link_kind/max_uses
- `ValidateInvitation(companyID, token)` — 未过期且有余量
- `RedeemInvitation(companyID, token, userID, memberName)` — 事务：建成员、核销、计数、事件

## 领域事件

| 事件 | 何时 | 载荷要点 |
|------|------|----------|
| INVITATION_CREATED | 邀请行提交后 | invitation_id, company_id, link_kind, max_uses, invitation_url（token 给投递用，日志指纹） |
| MEMBER_JOINED | 成员提交后 | member_id, invitation_id, link_kind, use_count, invitation_exhausted |

## 仓储端口（逻辑；本期落在现有 SQL handler，不强制新 package）

- `InvitationRepository.Insert / GetByToken / ConsumeUse / Revoke`
- `RedemptionRepository.Insert`

本期不拆新 Go 目录以避免大爆炸；不变量由 handler + 测试锁定，与既有 invite 代码同层。

## 无对应事件例外

validate / pending 列表：纯查询。
