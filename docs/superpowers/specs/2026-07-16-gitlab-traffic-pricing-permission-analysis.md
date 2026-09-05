# GitLab 流量费定价 — 角色权限分析

- 日期：2026-07-16
- 基于：`2026-07-16-gitlab-traffic-pricing-design.md`

## 结论

| 能力 | 角色 | 变化 |
|---|---|---|
| 创建/结束价格套餐（含新价目字段） | 系统管理员 `is_superuser` | 无新权限点；沿用既有 `/api/system-admin/pricing-packages/` |
| 查看公开/模板价 | 匿名/登录用户（既有公开接口） | 响应多一个字段，无权限收紧或放宽 |
| 租户查看当前/可切换套餐 | 租户成员（既有 billing API） | 展示新字段与说明文案 |

无新 endpoint；无角色矩阵变更。
