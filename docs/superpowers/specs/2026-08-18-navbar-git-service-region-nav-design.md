# 设计：导航栏代码仓库按区域资源跳转

- **Status:** accepted（/goal 自动采用）
- **Date:** 2026-08-18
- **Architecture artifacts:** 不新增。消费既有 v85 多区域 gitService（`billing_gitlab_region` + `billing_tenant_gitlab_resource`），无新服务/表/事件。

## 问题

Navbar 用 VIP1 拦截「代码仓库」→ `/pricing/`。普通会员获赠仓库、或多区域实例时行为错误。

## 决策

1. **门禁**改为「当前租户是否有仓库资源」（`disk_gb>0` 或 `traffic_prepaid_gb>0`，且 `provisioning_status` 非 `not_purchased`），与购买/赠送来源无关。
2. **GET** `/api/billing/gitlab-resources/tenant_id/{tid}/`（无 `region`）在既有 `gitlabResourceView` 上增加 `resources[]`：`region` / `region_name` / `gitlab_web_url` / `provisioning_status` / `disk_gb`。不泄漏 PAT。
3. **UI**
   - 0 条：`<a data-testid="nav-git-service" href="{route.fullPath}">`（留在当前页）
   - 1 条：直链 `gitlab_web_url`，`target=_blank`
   - ≥2 条：下拉（复用公司切换器模式：真实 `<a>` 菜单项、overlay 关闭、单击开/双击关）
4. **VIP1 角标**仅 `membershipTier==='vip1'`；不再改写 href。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 继续 VIP 门禁 + 价格页 | 与赠送资源冲突 |
| 无资源隐藏入口 | 用户要求留在当前页，入口仍可见 |
| 新 API `/gitlab-nav-targets/` | 无 region 的 GET 已存在，扩展字段即可 |

## Consequences

- Playwright 单区域 SSO 仍点 `nav-git-service` 直链。
- 多区域需先开下拉再点 `nav-git-service-region`。
- 无资源点击会刷新当前页（真实 href），可接受。
