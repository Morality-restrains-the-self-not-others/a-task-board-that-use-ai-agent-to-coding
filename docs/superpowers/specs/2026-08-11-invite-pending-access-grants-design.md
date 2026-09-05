# 邀请时预授页面/区域权限（含访问管理）

- **Status:** accepted（goal-mode 自动采纳）
- **Date:** 2026-08-11
- **Iteration:** invite-pending-access-grants-v74
- **Based on:** v72/v73 逻辑资源组 RBAC（ADR-0003 / ADR-0004）
- **Architecture impact:** **是** — ArchiMate application-integration v74
- **ADR:** 无新 ADR（扩展既有 page/region 模型到邀请生命周期；`No-ADR: covered by existing ADR-0003/0004`）
- **python_api_approval:** not_applicable（全 Go）

---

## 0. 问题

邀请人页面仅能选「成员 / 管理员」。管理员入职后得 `tenant_admin`（含访问管理）；普通成员入职后无 page/region 授予，须事后在「访问管理」再配。产品要求：**邀请当时即可为受邀人分配访问管理等页面/区域权限**。

## 1. 决策（锁定）

1. 邀请表单增加「页面与区域权限」勾选矩阵（复用访问管理目录与 view/operate 两档）。
2. 快捷项：**授予访问管理** → 勾选 `people.access` 整页（全部 ui_region = operate）。
3. 授予清单以 JSON 存入 `tenant_invitation.pending_grants`。
4. 受邀人 `join` 成功且非管理员时，taskTenantService 调 taskAuth **internal** API 创建自定义访问角色 + 绑定 resource groups，再 `ensureMemberRole`。
5. `role=admin` 时忽略 pending_grants（`tenant_admin` 已覆盖）。
6. 空 grants = 保持现状（仅成员，无自定义访问角色）。

## 2. 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 仅 checkbox「访问管理」 | 实现快 | 无法一次授其它页 | 作快捷项，非唯一能力 |
| B. 邀请时写完整 grants，join 落角色 | 与 PeopleAccess 同源 | 跨服务 internal API | **采用** |
| C. join 后靠 FE 再保存 | 少后端改动 | 不可靠、可绕过 | 拒绝 |

## 3. API / 数据

### 3.1 DDL

`dataMigrate/taskTenantService/009_invitation_pending_grants.sql`：

```sql
ALTER TABLE tenant_invitation
  ADD COLUMN pending_grants JSON NULL;
```

语义：`[{"group_key":"people.access.subject_list","effect":"operate"}, ...]`；NULL/[] = 无预授。

### 3.2 邀请 POST（既有）

`POST /api/tenant/{id}/accounts/members/invite/` body 增可选：

```json
{ "grants": [ { "group_key": "people.access", "effect": "operate" } ] }
```

鉴权不变：`member:manage`。非法 group_key / effect → 400。

### 3.3 Internal 落权

`POST /api/internal/authz/apply-member-grants/`（taskAuth，`X-Internal-Secret`）：

```json
{
  "company_id": "...",
  "display_name": "访问·张三",
  "grants": [ { "group_key": "...", "effect": "view|operate" } ]
}
```

→ 创建自定义角色 + `auth_role_resource_group` + 粗码桥接；返回 `{ id, name }`。

### 3.4 Join

`handleJoin`：读 `pending_grants`；非 admin 且非空 → apply → `ensureMemberRole` → `incrMembershipRev`。

## 4. 前端

- `PeopleInvite.vue`：角色下方嵌入 `InviteAccessGrants.vue`（目录 + 快捷「授予访问管理」）。
- `role===admin` 时灰显矩阵并提示「管理员已拥有全部权限」。
- 提交时附带 `grants`（admin 不传）。

## 5. 事件

`INVITATION_CREATED` / `MEMBER_JOINED` payload 增加可选 `grants`（审计用）；无新业务事件类型。

## 6. 🕸️ Code Review Graph 分析

- 热点：`handleInvite` / `handleJoin` / `saveSubjectResourceAccess` / PeopleAccess 目录。
- 爆炸半径：taskTenantService 邀请表、taskAuth 角色与 resource-group 写入、taskFE 邀请页。
- 风险：join 时 taskAuth 不可达 → 成员已创建但无预授；记 warn + 不回滚入职（可事后在访问管理补授）。

## 7. 测试意图（摘要）

1. 邀请带 people.access grants → DB pending_grants 非空。
2. join 后 member-role 为自定义角色且 PDP 含 `page:people.access` / region 码。
3. admin 邀请忽略 grants。
4. FE：勾选「授予访问管理」后 payload 含 people.access 相关 keys。

## 8. 架构交付物

- `docs/architecture/v74-application-integration-20260811-2100-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
- VERSION_HISTORY v74 target
---

## Role / NFR / DDD（流水线压缩）

- **角色**：邀请者须 `member:manage`；受邀人接受后获得预授 region/page；不可自授。
- **NFR**：L3 安全（授权）；internal secret；grants 上限 200 条；join 调 auth 超时 4s。
- **领域**：Invitation 聚合新增 PendingGrants 值对象；Join 领域服务 ApplyPendingAccessGrants。
