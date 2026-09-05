# 开放式邀请链接（Open Invite Link）— 设计文档

- **日期**: 2026-08-29
- **状态**: accepted（/goal 自动采用）
- **迭代**: open-invite-link
- **作者**: cursor

## 背景

租户「邀请人」页的「复制邀请链接」当前**只能使用一次**：`handleJoin` 在首次成功加入后将 `tenant_invitation.is_accepted=1` 并清空 `invitation_token`。产品要求能创建**开放式邀请链接**：同一 URL 可供**多名**已登录用户加入团队，直到过期或达到人数上限。

前端现文案：「邀请链接有效期为 N 天，且只能使用一次。」（`InviteLinkMethodPanel.vue`）

## 成功标准

1. 管理员在「复制邀请链接」可选择 **单次**（默认，行为与现网一致）或 **开放**。
2. 开放链接：至少两名不同用户可通过同一 `token` 成功 `POST join`；同一用户第二次加入仍失败（「您已在公司中」）。
3. 开放链接支持可选 `max_uses`：`0` = 不限人数（仅受有效期约束）；`N≥1` 达到上限后 validate/join 失败，待处理列表不再展示该 token。
4. 邮箱/电话邀请仍为单次；请求里带 `link_kind=open` 则 400。
5. 待处理邀请列表展示 `link_kind`、已用/上限（开放时）；撤销仍立即失效。
6. 存量邀请无新列语义时等价 `link_kind=single, max_uses=1`（DDL 默认值）。

## 方案（选定）

**扩展既有 `tenant_invitation` 聚合**，不新建邀请服务、不新建公网 path。Owner：`taskTenantService`。

### 数据

`dataMigrate/taskTenantService/012_open_invite_link.sql`：

| 列 | 类型 | 默认 | 含义 |
|---|---|---|---|
| `link_kind` | `VARCHAR(16)` | `'single'` | `single` \| `open` |
| `max_uses` | `INT NOT NULL` | `1` | 单次强制 1；开放时 `0`=不限，`N`=上限 |
| `use_count` | `INT NOT NULL` | `0` | 成功加入次数 |

审计表 `tenant_invitation_redemption`（前缀 `tenant_`）：

- `id` Snowflake PK
- `invitation_id`, `company_id`, `user_id`, `member_id`
- `created_at`
- `UNIQUE(invitation_id, user_id)` — 同链同用户不可重复核销

**冷热**：核销表时间累积型；预估年增量 `< 10万`（每租户开放链 × 加入人数）。本期**不分区**；主键 Snowflake；监控行数，超过 100 万再分区。

### API 契约（扩展既有端点，字段可选）

**POST** `/api/tenant/{tenantId}/accounts/members/invite/`

新增 body（均可选）：

```json
{
  "link_kind": "single | open",
  "max_uses": 0
}
```

规则：

- 缺省 `link_kind=single`，`max_uses` 忽略并落库 1。
- `invite_method` 非 `link` 且 `link_kind=open` → 400 `open_invite_link_only`。
- `open` 且 `max_uses < 0` → 400；`open` 且未传 `max_uses` → `0`（不限）。
- 开放时 `company_member_name` **可选**（作邀请备注/默认展示名）；单次仍建议填写（兼容现 UI 校验）。
- 201 响应增：`link_kind`, `max_uses`, `use_count`。

**GET** `/api/tenant/{tenantId}/accounts/members/validate-invite/?token=`

有效条件增加：未过期、token 非空、且 `(max_uses=0 OR use_count < max_uses)`。  
响应增：`link_kind`, `max_uses`, `use_count`, `remaining_uses`（不限时 `null`）。

**POST** `/api/tenant/{tenantId}/accounts/members/join/`

- 事务内：插入 member → 插入 redemption → 原子 `UPDATE ... SET use_count=use_count+1`  
  `WHERE id=? AND (max_uses=0 OR use_count < max_uses) AND invitation_token IS NOT NULL`。
- `RowsAffected=0` → 400「邀请链接无效或已过期」。
- **single**：保持现行为（`is_accepted=1`，清空 token）。
- **open**：不清空 token，直到 `max_uses>0 AND use_count>=max_uses` 才 `is_accepted=1` 并清空 token。
- 开放加入时成员名：join body 可选 `member_name` → 否则邀请备注 → 否则 `"成员"`。
- 每人每次成功加入仍发布 `MEMBER_JOINED`；payload 含 `invitation_id`, `link_kind`, `use_count`, `invitation_exhausted`（bool）。

**GET** pending-invitations：列表项增 `link_kind`, `max_uses`, `use_count`。开放且未耗尽的行继续出现。

**POST** revoke：不变（清空 token）。

不新增 Python 接口；不新增网关路由。

### 前端

- `InviteLinkMethodPanel`：链接类型单选「单次 / 开放」；开放时「最多人数」可空=不限；开放时成员名称改为可选「邀请备注」；文案改为开放可多人。
- `PeopleInvite.vue`：generate body 传 `link_kind`/`max_uses`；开放不强制成员名。
- `PendingInvitations.vue`：展示已用/上限。
- `PeopleJoin.vue`：validate 展示剩余名额（若有）；join 可带当前用户显示名作 `member_name`。
- 生成/加入按钮沿用 `createClickGuard` + `Idempotency-Key`。

### 并发

同一开放链接并发 join：依赖 `UPDATE ... use_count < max_uses` 行锁 + `UNIQUE(user_id, company_id)` 成员表。超额只成功 `max_uses` 人。

## 拒绝方案

| 方案 | 原因 |
|------|------|
| 独立 `tenant_open_invite` 表 | 与现邀请（角色、grants、workspace、revoke）重复 |
| 不消耗 token、永不 `is_accepted` | 无法设上限，待处理列表无法结束 |
| 邮箱/电话也改多人 | 目标是特定收件人；产品只要求链接 |
| 新 Kafka topic `INVITATION_EXHAUSTED` | 副作用已由 `MEMBER_JOINED.invitation_exhausted` 表达 |

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief`：增量无邀请符号。代码理解：`handleInvite` / `handleValidateInvite` / `handleJoin` / `handlePendingInvitations` / `handleRevokeInvitation`（`taskTenantService/src/invite_handlers.go`）；`InviteLinkMethodPanel.vue`；`PeopleInvite.vue`；`PeopleJoin.vue`；`PendingInvitations.vue`。MCP `codegraph_explore` 不可用；CLI `codegraph query handleInvite` 命中上述 handler。

## 🏛️ 架构变更影响

- **迭代版本**: v117 🎯 target
- **迭代名称**: 开放式邀请链接
- **作者**: cursor
- **设计日期**: 2026-08-29 13:10

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v117-application-integration-20260829-1310-cursor.puml` |
| | `docs/architecture/v117-enterprise-landscape-20260829-1310-cursor.puml` |
| ArchiMate 增量 | 同名 `.diff.archimate` |
| ArchiMate 全量 | 同名 `.full.archimate` |
| Mermaid | 同名 `.mermaid.md` |

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v116 → Gap「链接仅单次」+ WP-open-invite-link → Plateau v117；变更拓扑含 taskFE→GW→taskTenant、`tenant_invitation` 列扩展、`tenant_invitation_redemption` |
| **`.full.archimate`** | 变迁后邀请加入拓扑（含 v116 退订组件与本次开放链） |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ/契约 | 发布点 | 消费者 | 例外理由 |
|---------|----------------|--------|--------|--------|---------|
| 创建单次或开放邀请链接 | INVITATION_CREATED | Kafka `invitation-created` | `handleInvite` | 既有邮件/短信；link 渠道无 SMTP | — |
| 用户经链接加入公司 | MEMBER_JOINED | Kafka `member-joined` | `handleJoin` | git identity 等既有 | — |
| 校验邀请 / 列待处理 | — | — | — | — | 纯查询 |

`INVITATION_CREATED` payload 增 `link_kind`, `max_uses`。`MEMBER_JOINED` 增 `link_kind`, `use_count`, `invitation_exhausted`。

## 🐍 Python 新增接口清单与 Go 替代评估

不触发（扩展既有 Go `taskTenantService` + Vue）。`python_api_approval: not_applicable`

## 价值流影响（Step 1 输入）

影响既有人员邀请流（`conf/value-stream.yaml` 中 tenant invite / pending-invitation 相关 step）。新增 stream `open-invite-link`：创建开放链 → 多人 join → 上限/过期。字段：`task-tenant.tenant_invitation.link_kind` / `max_uses` / `use_count`；`task-tenant.tenant_invitation_redemption.user_id`。

## Domain Concept Inventory

- **Bounded Context**: Tenant / Membership
- **Aggregate**: Invitation（根 `tenant_invitation`）+ Redemption 实体
- **Events**: 见上表
- **不改**: 平台注册邀请码（`auth_registration_invite_code` 仍一码一用）

## 可观测性

结构化日志（小写 level）：`invite_created`（含 `link_kind`/`max_uses`，禁止打完整 token）、`invite_join`（`use_count`, `invitation_exhausted`）、`invite_join_rejected`。trace_id 经 tracelog。禁止日志输出完整 invitation_token。

## 回滚

1. 前端隐藏「开放」选项（只发 `single`）。
2. DDL 列可保留（默认 single）；核销表可留空。不删列以免已产生的开放行语义丢失。
