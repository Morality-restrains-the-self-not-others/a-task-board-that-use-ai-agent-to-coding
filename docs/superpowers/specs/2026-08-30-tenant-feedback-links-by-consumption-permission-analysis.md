# 意见与建议链接 — 角色权限分析

- **日期**: 2026-08-30
- **设计**: `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-design.md`
- **结论**: 绿灯 ✅ — 无新角色；租户 GET 必须走 v72 `page ⊃ ui_region`（禁止仅粗码）；超管 CRUD 沿用平台员工门禁

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/feedback-link-groups/` | 平台员工（`super_admin` / `employee`） | Platform 配置 | read | 无（新路径） | ⚠️ | `X-Gateway-Auth-Verified` + `authz.IsPlatformStaff` |
| GET `/api/system-admin/feedback-resource-kinds/` | 同上 | Platform | read | 无 | ⚠️ | 同上 |
| POST `/api/system-admin/feedback-link-groups/` | 同上 | Platform | write | 无 | ⚠️ | 同上 + `Idempotency-Key` |
| PUT `/api/system-admin/feedback-link-groups/{id}/` | 同上 | Platform | write | 无 | ⚠️ | 同上 |
| DELETE `/api/system-admin/feedback-link-groups/{id}/` | 同上 | Platform | write | 无 | ⚠️ | 同上 |
| GET `/api/tenant/{tenantId}/billing/feedback-links/` | 租户成员（须持 region） | Tenant `nav.feedback.main` | read | 无 | ⚠️ | `RequireRegionView("nav.feedback.main", tenantId)`；禁止仅 `feedback:view` |
| 超管页「意见与建议链接」 | 平台员工 | Platform UI | CRUD | SystemAdmin 壳 | ✅ | 现有系统管理路由守卫 |
| 租户侧栏「意见与建议」 | 持 `page:nav.feedback`（回退 `feedback:view`） | Tenant UI | view | 无 | ⚠️ | `TENANT_CONSOLE_NAV` 增 `nav.feedback`；访问管理可关掉 |

**v72 对照**：新增 page `nav.feedback`、region `nav.feedback.main`。叶子：UI `TenantConsoleFeedbackNav.Main`；API `GET /api/tenant/{tenantId}/billing/feedback-links/`。内置 `tenant_admin` / `member` / `group_admin` 默认绑定该 page（产品：全员同套链接）。`tenant_admin` 亦由 032 全 page 绑定覆盖，本迁移对 member/group_admin 显式 INSERT。

粗码 `feedback:view` 仅作侧栏过渡回退与访问管理勾选并集，**不是** GET 的唯一门禁。

## 角色与权限建模

无新角色。新增：

```
role: (existing) tenant_admin / member / group_admin
permissions: page:nav.feedback → region:nav.feedback.main:view
scope: tenant
```

平台：无新 platform 码；写接口 `IsPlatformStaff`。

## 安全审查结论

- [x] **IDOR**: 租户 GET 的 `tenantId` 必须与网关租户上下文一致；消耗投影只读该 tenant 的 grant/gitlab/账本
- [x] **权限提升**: 普通成员不能写超管 CRUD（403）；不能靠伪造消耗 JSON 解锁链接（服务端过滤）
- [x] **跨租户泄露**: 消耗 SQL 全部 `tenant_id=?`；响应不含阈值
- [x] **403 vs 404**: 无 region → 403；组 id 不存在 PUT/DELETE → 404
- [x] **user_id 注入**: 可见性不按 user；忽略 body user_id
- [x] **敏感操作**: URL 仅 https；日志不打超管 token
- [x] **并发**: 超管保存整组替换；幂等键同意图回放
- [x] **密钥**: 无新密钥

## 权限测试清单

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 超管 GET 组 | super_admin | GET groups | 200 |
| 租户成员 PUT | member | PUT groups | 403 |
| 无 region 租户 GET | 已登录无 `nav.feedback.main` | GET feedback-links | 403 |
| 有 region 成员 GET | member + page | GET | 200，无 thresholds |
| 访问管理关掉 page | 自定义角色无 nav.feedback | 侧栏 | 不显示「意见与建议」 |

## 设计文档补丁

设计「权限」节改为：租户 API **RequireRegionView(`nav.feedback.main`)**；侧栏 `page:nav.feedback` 优先、粗码 `feedback:view` 回退。种子见 `dataMigrate/taskAuth/052_nav_feedback_page.sql`。
