# 角色权限分析：系统管理用户表列过滤器

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-system-admin-users-column-filters-design.md`

## 结论

不新增角色或权限码。列过滤只是超管已有列表 GET 的查询参数扩展，沿用 `requireSuperuser`。

## 改动点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/users/` + 列过滤 query | superuser / 平台超管 | System | read | `requireSuperuser` | ✅ 充分 | 不放宽；非超管 403 |
| 前端 `/system-admin/users/` 表头过滤行 | 同上 | System | read | 路由 `requiresAdmin` | ✅ 充分 | 推荐码申请 tab 不渲染该行 |
| taskTenant / taskReferral 批量内部查询 | taskAuth 服务身份 | System | read | 既有 Internal Secret | ✅ 充分 | 仅 enrichment 过滤时对候选 ID 调用 |

## 风险

| 风险 | 判定 |
|------|------|
| IDOR | 无。列表本就是系统级全量用户，过滤不扩大可见范围。 |
| 信息泄露 | 过滤条件可能含邮箱/手机号。日志禁止写 query 原文，只写键名。 |
| 枚举 | 超管本就可枚举用户；不新增对外接口。 |

## 角色建模

无新角色。
