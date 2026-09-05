# 角色权限分析：测试角色 + GitLab 区域访问模式

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-tester-role-gitlab-region-access-mode-design.md`

## 新角色定义

```
role: tester（标志 is_tester，非平台 RBAC）
display_name: 测试
description: 产品权限等同租户；可使用 development 模式 GitLab 区域
permissions: 与 is_tenant 相同 + CanUseGitlabRegion(development)
scope: platform-account-flag（账号级，非租户内角色）
```

层级（账号类别，非 RBAC 树）：

```
super_admin (is_superuser)
  └─ employee (is_staff)
       └─ tester (is_tester ⊂ tenant 能力)
            └─ tenant (is_tenant)
                 └─ user
```

tester **不得**进入系统管理。设置 tester 仅超管/员工经既有系统管理用户 API。

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| PATCH 用户 is_tester | 系统管理 staff | System | write | 系统管理路由 + 超管授权提权 | ✅ | 与 is_tenant 同允许字段表 |
| GET 用户列表 role=tester | 系统管理 staff | System | read | 同上 | ✅ | 过滤器扩展 |
| GET /me is_tester | 登录用户本人 | User | read | 已登录 | ✅ | 只读自身 |
| forward-auth X-User-Is-Tester | 网关 | System | inject | internal secret | ✅ | 仅 0/1 |
| GET 租户区域目录 | 租户成员 | Tenant | read | 租户鉴权 | ⚠️ | 按 is_tester 过滤 development |
| POST 购买 GitLab 资源 | 租户管理员 | Tenant | write | 既有购买门禁 | ⚠️ | 叠加 CanUseGitlabRegion |
| PUT 系统管理区域 metadata | 系统管理 | System | write | 系统管理路由 | ✅ | 可写 access_mode；不过滤 |
| GET 系统管理区域列表 | 系统管理 | System | read | 系统管理路由 | ✅ | 含 development |

## 租户逻辑资源组（元规则 45）

本增量是**平台超管控制台** + 既有租户 GitLab 购买路径的条件过滤，不新增租户页面/ui_region。不登记新 `auth_resource_group`。理由：无新租户菜单；门禁是账号标志 × 区域模式，不是租户内 RBAC。

## IDOR / 提权

- 普通租户不能自助把账号设为 tester（仅系统管理 PATCH）。
- 非测试租户不能通过猜 slug 购买 development 区域。
- tester 不能因标志进入 `/system-admin/`。

## 审计

写路径打 slog：`user_id`（操作者）、`target_user_id` 或 `region_slug`、`access_mode`/`is_tester` 新值；禁止 PII。
