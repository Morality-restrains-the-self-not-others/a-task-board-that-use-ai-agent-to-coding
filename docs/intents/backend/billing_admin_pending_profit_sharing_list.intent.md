# 功能意图：管理员待分账订单列表

## 意图

平台员工在系统管理「订单与退款」页打开「待分账」Tab 时，服务端返回未完成（及按筛选）的分账记录队列，按最早可分账时间排序。租户不得调用本接口。

## 角色

- 系统管理员 / 平台员工：跨租户读
- 租户成员：403

## 行为

1. `GET /api/system-admin/profit-sharing/`：网关已验证 + `IsPlatformStaff`；200 含 `items`、`total`、`limit`、`offset`。
2. 默认 `status=open`（`pending|processing|failed`）；也可传单一状态或 `all`。
3. `limit` 默认 20、最大 50；`offset` ≥ 0。
4. 每项含订单号、租户、接收方用户 ID、订单金额、分账金额、状态、`settle_after`、`fail_reason`、`fail_trace_id`（分账失败当次请求的 trace id，可空）。平台员工可见微信单号、`app_id`、`openid`（成对取自接收方登记，与 `wechat_identity` 支付/mp 行一致；无接收方 openid 时 openid 才回退台账快照；始终出键，空串表示未绑定）。不含 `referrer_openid` 键。
5. 未登录 401；非平台员工 403；非法 status 400。
6. 写入仍由既有 `markOrderForProfitSharing`；本意图不新写。

## 非目标

- 不改分账执行、比例、微信回调
- 不提供立即分账/重试

## 业务意图 → 事件对照

**无对应新事件（纯查询例外）**

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 管理员读取待分账队列 | — | — | — | 只读；记录已在支付成功路径写入 |

## 变更记录

- 2026-08-22：新增管理员待分账列表 API
- 2026-08-26：列表项增加 `fail_trace_id`，供失败原因列 `data-traceId`
- 2026-08-26：列表项增加 `app_id` / `openid`，供超管对照「appid 与 openid 不匹配」
- 2026-08-26：`app_id`/`openid` 改为接收方成对取值，禁止与台账 web 登录快照混用
- 2026-08-26：`app_id`/`openid` 改为接收方成对取值，禁止混用台账快照 openid（与 `wechat_identity` 不一致）
