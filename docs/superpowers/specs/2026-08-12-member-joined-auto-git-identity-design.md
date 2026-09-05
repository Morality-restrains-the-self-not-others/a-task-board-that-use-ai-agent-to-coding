# 公司成员多 Git 身份 + MEMBER_JOINED 自动建身份

- **Status:** accepted (goal-mode 自动采纳)
- **Date:** 2026-08-12
- **Iteration:** member-joined-auto-git-identity-v77
- **Base architecture:** v76

## Context

People 管理页（`/tenant/{id}/people/manage/`）需要：

1. 一名公司成员可拥有多个 Git 身份（`task_git_identities` 已支持）
2. 创建公司成员成功后向 Kafka 投递事件
3. 消费者自动创建默认 Git 身份
4. 租户管理员 / 小组管理员 / 成员本人可管理该公司下的 Git 身份

## Decision

### 事件

- **复用** `MEMBER_JOINED` / topic `member-joined`（已有 producer-only 配置）
- **扩展发布点**：邀请 join、公开建公司创建者成员、内部 upsert 成员（新建时）
- **payload 契约**：

```json
{
  "member_id": "<snowflake>",
  "user_id": "<user>",
  "company_id": "<tenant>",
  "member_name": "<公司成员名称>",
  "workspace_id": "<optional>",
  "role": "admin|member",
  "source": "invite_join|company_create|internal_upsert"
}
```

### 自动建身份

- Intent：`member_joined/1_create_default_git_identity`（port **18060**）
- 调用 taskTaskService `POST /api/internal/git-identities/ensure-default/`
- 默认字段：
  - `git_user_name` = `member_name`（空则回退 `user_id`）
  - `git_user_email` = `{sha256(member_id)[:16]}.{sha256(company_id)[:16]}@daydaymoney.com`
  - `label` = `system-auto`
  - `is_default` = true（若该公司尚无默认）
- 幂等：同 `user_id+company_id+git_user_email` 已存在 → 成功跳过

### 管理 API（taskTaskService）

| Method | Path | Auth |
|--------|------|------|
| GET/POST | `/api/git-identities/tenant/{tenantId}/member/{memberId}/` | self（目标成员 user）OR `member:manage` OR `group-members:manage` |
| PATCH/DELETE | `/api/git-identities/tenant/{tenantId}/identity/{identityId}/` | 同上（按 identity 归属公司校验） |

### 前端

- `MemberList`：操作列增加「Git 身份」→ 弹窗列表/新增/设默认/删除
- 自助页 `UserGitIdentities` 保持不变

### 数据所有权

- 表 `task_git_identities` 仍属 taskTaskService；不跨服务直写

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 新事件 `COMPANY_MEMBER_CREATED` | 与已有 `MEMBER_JOINED` 重复；conf 已有 topic |
| 在 taskTenantService 内同步建身份 | 违反单服务数据所有权 |
| 仅 invite 路径发事件 | 创建者/内部建成员漏建身份 |

## Consequences

- 需注册 consumer + runAll 条目 + EventTopic
- 存量成员无自动回填（可后续批处理 OPT）
- 组管理员以 `group-members:manage` 码放行（不按单组边界细拆，记 OPT）

## Architecture artifacts

- `docs/architecture/v77-application-integration-20260812-1535-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`

## No-ADR

`No-ADR: covered by existing event/consumer patterns + single-service data ownership rules`
