# DDD — 导航栏代码仓库区域跳转

## Bounded Context

计费/GitLab 配额（taskBill）只读查询；门户导航（taskFE）展示。

## 值对象

`GitlabNavTarget`：`region`（slug）、`region_name`、`gitlab_web_url`、`provisioning_status`、`disk_gb`。

聚合根仍是租户 GitLab 资源账本 `billing_tenant_gitlab_resource`（tenant_id + region）。不新增表。

## 查询端口

`listTenantGitlabNavTargets(tenantID) []GitlabNavTarget` — 按租户列出可跳转区域。

## 事件

无。纯查询。

## 架构变更影响

无新 Application_Component。复用 v85 区域注册表与租户配额。
