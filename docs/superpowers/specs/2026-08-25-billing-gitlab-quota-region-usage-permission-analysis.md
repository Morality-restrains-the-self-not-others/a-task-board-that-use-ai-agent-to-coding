# 角色权限分析 — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-billing-gitlab-quota-region-usage-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET /api/tenant/{tid}/billing/quotas/` | 租户成员（账单页读者） | Tenant | read | 既有 tenant 路径鉴权 + 租户归属 | 否 | 不改 |
| `BillingDashboard` 渲染 used/quota | 同上 | Tenant | 展示 | 前端仅渲染本 `tid` 响应 | 否 | 不展示他租户数据；无新写操作 |
| `gitlab_web_url` 外链 | 同上 | Tenant | 导航 | 真实 `<a href>` | 否 | Anti-Replay-OK：只读外链 |

## 结论

无新 endpoint、无新权限角色。IDOR 面与现网 quotas 相同（路径 `tenant_id`）。本增量不扩大数据暴露：used 字段本就在 quotas JSON 中。
