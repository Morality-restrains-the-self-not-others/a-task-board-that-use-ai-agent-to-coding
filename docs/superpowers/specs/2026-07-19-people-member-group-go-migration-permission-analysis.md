# 角色权限分析 — 人员页成员/邀请/分组迁 Go

日期：2026-07-19  
设计：`2026-07-19-people-member-group-go-migration-design.md`

## 新/改 Endpoint

| 路径 | 方法 | Owner | 鉴权 |
|------|------|-------|------|
| `/api/tenant/{t}/accounts/members/invite/` | POST | task-tenant-service | Gateway JWT；须公司 admin/creator |
| `/api/tenant/{t}/accounts/members/validate-invite/` | GET | task-tenant-service | AllowAny（仅 token 有效性） |
| `/api/tenant/{t}/accounts/members/join/` | POST | task-tenant-service | 已登录用户 |
| `/api/tenant/{t}/accounts/members/company_members/` | GET | task-tenant-service | admin/creator 才有列表；否则 meta.has_permission=false |
| `/api/tenant/{t}/accounts/members/{id}/update_role/` | PATCH | task-tenant-service | admin/creator；禁止改创建者角色 |
| `/api/tenant/{t}/accounts/members/{id}/toggle_status/` | PATCH | task-tenant-service | admin/creator；禁止禁创建者 |
| `/api/tenant/{t}/accounts/members/{id}/` | DELETE | task-tenant-service | admin/creator；禁止删创建者 |
| `/api/tenant/{t}/accounts/members/pending-invitations/` | GET | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/members/{id}/revoke-invitation/` | POST | task-tenant-service | 公司成员（同租户邀请） |
| `/api/tenant/{t}/accounts/members/{id}/resend-invitation-link/` | POST | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/groups/` | GET/POST | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/groups/{id}/` | DELETE | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/groups/{id}/members/` | GET | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/groups/{id}/add_member/` | POST | task-tenant-service | 公司成员 |
| `/api/tenant/{t}/accounts/groups/{id}/remove_member/` | DELETE | task-tenant-service | 公司成员 |
| `/api/internal/tenant/**` | * | task-tenant-service | Internal secret |

## 权限矩阵

| 角色 | 邀请/管人写 | 看成员列表 | 分组 CRUD | validate-invite |
|------|-------------|------------|-----------|-----------------|
| 公司创建者 | ✅ | ✅ | ✅ | ✅（有 token） |
| 公司 admin | ✅ | ✅ | ✅ | ✅ |
| 普通成员 | ❌（写） | ❌（空列表+提示） | ✅（现网同） | ✅ |
| 非成员 | ❌ | ❌ | ❌ | ✅（token） |
| 匿名 | ❌ | ❌ | ❌ | ✅（token） |

## 审计要点

- 403/拒绝打 warn 日志：user_id、company_id、traceId
- 邀请 token 一次性；revoke 清 token
- Internal API 禁止无 secret 公网可达（仅 loopback/网关后）
