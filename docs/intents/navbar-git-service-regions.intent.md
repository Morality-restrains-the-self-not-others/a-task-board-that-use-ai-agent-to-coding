# 功能意图：Navbar「代码仓库」展示已购 GitLab 区域下拉

## 意图

已登录用户在顶部导航悬停或单击「代码仓库」时，展开下拉菜单，列出当前租户已购买或获赠的 GitLab 资源区域；每项为真实外链（`gitlab_web_url`），新标签打开。无已购区域且列表已成功拉取时入口为真实 `<a href=/pricing/>`（保留 accessCode）；拉取失败则留在当前页。

## 验收

1. `GET /api/tenant/{tid}/billing/gitlab-resources/`（无 `?region=`）返回 `{ tenant_id, resources: [{ region, region_name, gitlab_web_url, provisioning_status, ... }] }`
2. `GET .../billing/gitlab-resources/?region=<slug>` 仍返回单区详情（兼容设置页）
3. `resources` 仅含已购/获赠行（`disk_gb > 0 OR traffic_prepaid_gb > 0`）
4. 前端 ≥1 个带 `gitlab_web_url` 的区域时：悬停或单击展开下拉；双击收起；菜单项 `target=_blank`
5. 0 个可跳转区域且列表 ready：`<a href=/pricing/>`（可带 accessCode），不出现空下拉；unknown/error 时 href 为当前页

## 设计说明

- 列表路径为轻量摘要，不做磁盘刷新 / 租户组开通副作用
- 下拉交互遵循前端规范：单击展开、双击收起；另支持悬停展开以匹配导航场景

## 变更日期

2026-08-19；2026-08-29 无资源就绪改跳价格页

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 查询租户已购 GitLab 区域列表 | — | — | — | Navbar 渲染下拉 | 纯查询，无状态变更 |
