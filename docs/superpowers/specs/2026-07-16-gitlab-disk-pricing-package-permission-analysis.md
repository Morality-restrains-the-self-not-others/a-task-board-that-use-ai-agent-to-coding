# GitLab 磁盘价格套餐 — 角色权限分析

- 日期：2026-07-16
- 基于：`2026-07-16-gitlab-disk-pricing-package-design.md`

## 角色矩阵

| 能力 | 匿名 | 租户成员 | 租户管理员 | 系统超管 |
|------|------|----------|------------|----------|
| 读公开定价（含 GitLab 磁盘价） | ✅ | ✅ | ✅ | ✅ |
| 列/创建价格套餐 | ❌ | ❌ | ❌ | ✅ |
| 查套餐订户 | ❌ | ❌ | ❌ | ✅ |
| 读本租户锁价/套餐 | ❌ | ✅（鉴权后） | ✅ | ✅ |
| 切换套餐（含新锁价） | ❌ | 按既有账单权限 | ✅ | — |

## 改动点审计

| 端点/界面 | 变更 | 权限结论 |
|-----------|------|----------|
| `GET/POST /api/system-admin/pricing-packages/` | 响应/请求增字段 | 保持 `is_superuser` |
| `GET /api/public/product-pricing/` | 响应增字段 | 保持 AllowAny |
| `GET .../billing/accounts/...` 套餐 JSON | 增字段 | 保持租户鉴权 |
| `/system-admin/price-management` | 表单/表列 | 路由 `requiresAdmin` |

## 结论

无新角色、无权限放宽；仅扩展既有超管写路径与公开/租户读路径字段。
