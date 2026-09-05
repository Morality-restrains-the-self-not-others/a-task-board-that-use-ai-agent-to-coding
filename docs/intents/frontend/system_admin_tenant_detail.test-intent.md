# 测试意图：系统管理租户详情页

## 测试目标

证明名称列是可导航链接，详情页并行拉取并渲染三块数据，失败带 traceId。

## 测试分层

| 层 | 位置 |
|----|------|
| 面板 | `taskFE/app/src/components/SystemAdminTenantsPanel.test.js` |
| 详情页 | `taskFE/app/src/views/SystemAdminTenantDetail.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 有 name | `<a href="/system-admin/tenants/c1/">Acme</a>` |
| name 空 | 文本 `—`，无链接 |
| header 200 | 标题含公司名 |
| header 404 | 错误节点 data-traceId |
| quotas 200 | 展示任务帖配额数字 |
| workspaces items | 表格含工作空间名 |
| workspaces [] | 暂无工作空间 |
| orders 200 | 表格含 order_number |
| orders [] | 暂无订单 |
| 返回 | `href="/system-admin/users/?tab=tenants"` |

## 通过标准

上述测例全绿。
