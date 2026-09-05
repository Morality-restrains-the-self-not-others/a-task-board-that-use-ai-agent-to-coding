# Navbar 无仓库跳价格页 — 角色权限分析

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-navbar-git-empty-to-pricing-design.md`

## 结论

无新 endpoint、无新角色。沿用既有：未登录隐藏入口；已登录可见；列表 API 仍为租户作用域 GET。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 空列表 `<a href=/pricing/>` | 已登录任意会员 | System（公开价格页） | 读/导航 | `isUserAuthenticated` | ✅ | 价格页本就公开；保留 accessCode 与「价格」链一致 |
| `GET /api/tenant/{tid}/billing/gitlab-resources/` | 租户成员 | Tenant | 读 | 既有鉴权 | ✅ | 本迭代不改 |
| 下拉外链 `gitlab_web_url` | 已登录且有资源 | Tenant GitLab | 读 | 仅渲染已购/获赠行 | ✅ | 不变 |
| 拉取失败 fail-open 留当前页 | 已登录 | — | 导航降级 | 前端 status=`error` | ✅ | 避免误转化 |

无 IDOR：不根据 userId 拼他人租户资源；tid 来自当前会话租户。价格页无租户写操作。
