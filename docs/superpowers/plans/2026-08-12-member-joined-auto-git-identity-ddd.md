# Step 6 — DDD：MEMBER_JOINED 自动 Git 身份

## 限界上下文

| 上下文 | 服务 | 职责 |
|--------|------|------|
| Tenant Membership | taskTenantService | 公司成员生命周期；发布 `MEMBER_JOINED` |
| Task / Git Identity | taskTaskService | `GitIdentity` 聚合；ensure-default + 租户管理 API |
| Event Orchestration | taskEvents | Intent `1_create_default_git_identity` |
| Tenant Console UI | taskFE PeopleManage | 成员行 Git 身份管理 |

## 聚合 / 实体

- **GitIdentity**（taskTask）：`id`, `user_id`, `company_id`, `git_user_name`, `git_user_email`, `label`, `is_default`
- **CompanyMember**（tenant，既有）：触发事件的源实体

## 领域事件

- **MEMBER_JOINED**（复用/扩展 payload）
  - Topic: `member-joined`
  - Key fields: `member_id`, `user_id`, `company_id`, `member_name`, `role`, `source`
  - Consumer effect: EnsureDefaultGitIdentity（幂等）

## 端口

- `EnsureDefaultGitIdentity` — 内部 HTTP
- `TenantGitIdentityManage` — 租户域 CRUD（self/perm）
