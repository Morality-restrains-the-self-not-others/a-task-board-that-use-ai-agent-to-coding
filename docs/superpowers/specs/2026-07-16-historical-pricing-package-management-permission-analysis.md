# 历史套餐管理 — 权限分析

参考设计：`docs/superpowers/specs/2026-07-16-historical-pricing-package-management-design.md`

| 接口 | 角色 | 资源 | 操作 | 既有防护 | 结论 |
|------|------|------|------|----------|------|
| `GET /api/system-admin/pricing-packages/` | superuser | PricingPackage | read | `is_superuser` | ✅ |
| `POST /api/system-admin/pricing-packages/` | superuser | PricingPackage | write | `is_superuser` | ✅ |
| `POST /api/system-admin/pricing-packages/<id>/end/` | superuser | PricingPackage | write | `is_superuser` + 存在性校验 | ✅ 新增 |
| `GET /api/system-admin/pricing-packages/<id>/tenants/` | superuser | BillingAccount | read | `is_superuser` | ✅ |

| 风险 | 评估 |
|------|------|
| IDOR | 低 — 系统级资源，超管可访问任意套餐 |
| 自锁/破坏性 | 中 — 结束在售不可逆改单价；仅关窗口；已锁价租户不受影响 |
| 非超管 | 401/403（既有中间件 + 视图） |

## 验收场景

| 场景 | 期望 |
|------|------|
| 匿名 POST end | 401 |
| 普通用户 POST end | 403 |
| 超管结束在效期内套餐 | 200，`valid_to` 已设 |
| 超管重复结束 | 400 |
