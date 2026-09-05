# 设计：租户订单列表按交易单号 / 商户单号查询

- **日期**: 2026-08-29
- **入口**: `/tenant/:tenant/billing/orders/`
- **架构变更**: 无（不新增服务/聚合/消息/表；不更新 `docs/architecture/`）
- **TraceId**: 无（需求为功能增量，非报错排障）

## 背景

租户订单列表页仅有状态筛选与 `order_id` 深链。用户从微信支付凭证拿到「微信支付交易单号」（`wechat_transaction_id`）或「商户订单号」（`out_trade_no`）后，无法在本页定位订单。

管理端已有「交易单号查询」，且 **GET** `/api/tenant/{tid}/billing/orders/?order_number=` 已等值匹配展示号、主键、`payment_ref`、`out_trade_no`、`wechat_transaction_id`，并强制 `tenant_id = tid`。缺口在租户前端未暴露该参数。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 7 文件，风险分 0。
- 调用链：`BillingOrders.fetchOrders` → `GET /api/tenant/{tid}/billing/orders/` → `handleListOrders` → `listOrdersByTradeNo` / `listTenantOrders`。
- 爆炸半径：只读列表查询；不改支付入账、不改管理端跨租户查单。
- 结论：复用既有 `order_number` 查询，不新开匿名查单接口。

## 方案（已采纳）

在租户订单列表增加两个精确查询输入（交易单号、商户单号），提交后走既有租户列表 GET 的 `order_number`。

1. **UI**：标题下、状态筛选旁（或上方）两个输入框 +「查询」+「清空」。
   - 交易单号：微信支付交易单号（如 `4500000359202608221274536815`）
   - 商户单号：微信商户订单号（如 `WX878981209491800064`）
2. **请求**：`GET /api/tenant/{tid}/billing/orders/?order_number=<trim 后的值>&limit=&offset=`，可叠加既有 `status`。
   - 仅填交易单号 → 该值作为 `order_number`
   - 仅填商户单号 → 该值作为 `order_number`
   - 两项都填 → 以交易单号为准（后端 OR 匹配本就覆盖两类凭证；避免双值歧义）
   - 两项皆空 → 不传 `order_number`，保持原分页列表
3. **归一化**：提交前 trim，去掉首尾 `` ` `` / `'` / `"`（Excel 导出）。空提交本地提示，不发带空 `order_number` 的请求。
4. **结果**：命中则列表仅含本租户匹配行；未命中空态「暂无订单记录」；超长由后端 400。
5. **隔离**：后端已有 `tenant_id` 约束；不得查到其他租户订单。本增量不按 `user_id` 再滤（与现网列表一致：有 `billing:view` 的成员看本租户订单）。
6. **防重放**：查询为只读 GET；查询按钮用 `createClickGuard` 同步门闩，注释 `Anti-Replay-OK: 只读列表查询`。
7. **OpenAPI**：更新 `order_number` 说明，写明匹配微信支付交易单号与商户订单号。

## 非目标

- 不调用微信 QueryOrder 兜底（管理端跨租户才兜底；租户侧保持本地等值）。
- 不模糊搜索、不按金额/时间扫单。
- 不新增匿名/跨租户查单。
- 不在列表行展示完整微信单号（避免表格过宽；详情 JSON 已有字段）。
- 不改支付入账写凭证路径。

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/tenant/{tid}/billing/orders/?order_number=` | 已登录 + 该租户 `billing:view` / `billing:manage`（网关 PDP，与现列表相同） | 仅 `tenant_id=tid` |

## 事件

纯查询，无新业务状态。**书面例外：不投递 MQ。**

## Python 新接口

无。落点为既有 Go `taskBill` + Vue `taskFE`。
