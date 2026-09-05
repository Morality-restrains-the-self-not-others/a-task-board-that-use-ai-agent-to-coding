# 实施计划 — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **设计 / NFR / DDD**: 见同前缀 specs/plans

不新增 Kafka。事件任务：核对意图例外表 + 下单日志含 region。

## Task 1 — 后端：未知 slug 必须失败（Red→Green）

- 测：`taskBill/src/orders_gitlab_region_test.go`
- 改：`createOrder` 对 GitLab 行 `getGitlabRegionBySlug`，写回 canonical slug
- 验证：`go test ./src -count=1 -run 'CreateOrder_Gitlab'`

## Task 2 — 后端：合法 slug 写入订单行

- 测：指定 `tencent-sh-1` 后 `billing_resource_order_item.region` 匹配
- 改：同上；`resource_order_created` 日志带 regions
- 验证：VIP1 租户下单成功

## Task 3 — 前端：按行选区 + 详情展示 + 购买入口

- 测：`OrderCreate.contract.test.js` 磁盘/流量可不同 region；`OrderDetail.contract.test.js` 展示 region；设置页 href
- 改：`OrderCreate.vue`、`OrderDetail.vue`、`WorkspaceSettingsGitlabConnection.vue`
- 验证：`npx vitest run src/views/OrderCreate.contract.test.js src/views/OrderDetail.contract.test.js src/views/WorkspaceSettingsGitlabConnection.test.js`

## Task 4 — 意图

- B-049b + INDEX；B-049 验收补一条
- OrderCreate ≤ 500 行
