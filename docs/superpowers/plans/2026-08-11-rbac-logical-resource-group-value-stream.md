# Value Stream — tenant-logical-resource-access (v72)

- **Date:** 2026-08-11
- **Stream id:** `tenant-logical-resource-access`

## 端到端步骤

1. **配角色×Region** — 管理员在访问管理勾选 ui_region（或整页 page）→ PUT role resource-groups  
2. **主体获角色** — 既有 member-role / group-role  
3. **PDP** — forward-auth 展开 page→regions → 注入 `region:*` / `page:*`  
4. **消费** — FE `hasRegion`/`hasPage`；BE `RequireRegion` → 403  

## 字段（三节）

| name | description |
|------|-------------|
| `task-auth.auth_resource_group.group_key` | page/region 稳定键 |
| `task-auth.auth_resource_member.member_key` | ui/api 叶子 |
| `task-auth.auth_role_resource_group.role_id` | 角色绑定 |

## 事件

| 意图 | 事件 | 发布 | 消费 |
|------|------|------|------|
| 角色↔RG 变更 | 扩展既有角色更新日志 + 依赖 membership_rev | taskAuth | PDP 缓存失效 |
| 成员角色变更 | TenantRoleChanged | taskTenant | membership_rev |

查询目录：只读例外，无事件。
