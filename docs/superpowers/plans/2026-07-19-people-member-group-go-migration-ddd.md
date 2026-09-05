# DDD 领域建模 — 人员组织（taskTenantService）

日期：2026-07-19

## 限界上下文

**Company Membership（组织成员）** — 新建服务 `taskTenantService`。

## 聚合 / 实体

| 类型 | 名称 | 不变式 |
|------|------|--------|
| Aggregate Root | CompanyMember | (user_id, company_id) 唯一；创建者不可删/禁/改角色 |
| Entity | Invitation | token 唯一可空；accepted 后 token 清空；须 workspace_id |
| Aggregate Root | CompanyGroup | 属单一 company |
| Entity | CompanyGroupMember | (group_id, user_id) 唯一；user 须为公司成员 |

## 值对象

- Role：admin | member（展示层另有「创建者」）
- InviteMethod：email | phone | link
- DisplayName：member_name → profile → user_id 前缀

## 领域服务

- `AdminPolicy.RequireCompanyAdmin(user, company, creatorID)`
- `InvitationService.Issue / Resend / Revoke / Accept`
- `GroupMembershipService.Add / Remove`

## 端口

| Port | 适配器 |
|------|--------|
| MemberRepository | SQLite |
| InvitationRepository | SQLite |
| GroupRepository | SQLite |
| CompanyDirectory | Django HTTP（exists + creator_id） |
| WorkspaceAccess | taskProjectService HTTP |
| EventPublisher | Kafka（INVITATION_CREATED, MEMBER_JOINED） |
| IdentityEmail | taskAuth / Django（可选） |

## 领域事件

见设计文档 §5；查询路径无事件（例外已声明）。
