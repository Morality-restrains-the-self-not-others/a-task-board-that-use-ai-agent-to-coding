# 价值流 — 导航栏代码仓库区域跳转

Mapping the approved design into a value stream.

## Related Value Streams

- `tenant-gitlab-settings-resource-purchase`（2026-07-18）：购买/查询配额。本次是**扩展读取**，不改购买。
- 管理端赠送 GitLab 磁盘（admin grant region）：赠送写入同一张 `billing_tenant_gitlab_resource`，Navbar 必须读到赠送行。

## Increment

用户已登录 → Navbar 拉 GET gitlab-resources（无 region）→ 按 `resources[]` 渲染入口 → 用户打开目标区域 GitLab。

测试点：T1 空列表当前页；T2 赠送单区直链；T3 多区下拉。
