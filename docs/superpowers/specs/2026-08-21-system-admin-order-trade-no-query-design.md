# 设计：系统管理员按交易单号查询订单

- **日期**: 2026-08-21
- **入口**: `/system-admin/order-records/` 订单记录 Tab
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）

## 背景

管理端订单记录页已有「粘贴订单号跳转」：仅当输入为 ADR-0018 四段 `ORD-日期-租户-主键` 时，经 `POST /api/system-admin/order-number/parse/` 解析后深链定位。三段号、纯 Snowflake 主键、支付渠道 `payment_ref`（微信支付交易单号等）均无法查询。列表 GET 也没有按展示号过滤。

用户要求：在该页能通过**交易单号**查询。

## 方案（已采纳）

在既有只读列表上增加可选精确匹配查询，不新开匿名查单接口。

1. **GET** `/api/system-admin/orders/?order_number=`（平台员工）与 **GET** `/api/tenant/{tenant_id}/billing/orders/?order_number=`（租户管理员，仅本租户）。
2. 查询值 trim 后长度 1–128；**精确匹配**下列任一字段（OR，参数化 SQL）：
   - `billing_resource_order.order_number`（含四段/三段展示号）
   - `id`（当整段输入可解析为 int64 主键；**不**从三段号拆主键，避免日序号误撞 Snowflake）
   - `payment_ref`（渠道交易号）
3. 禁止 `LIKE` 前缀/模糊扫描。`order_number` 有 UNIQUE；`id` 为主键。命中 0 条返回空列表 `total=0`，不 404。
4. 与 `order_id` 深链并存：`order_number` 非空时优先按交易单号过滤，不再做 `order_id` offset 对齐。
5. 前端将「粘贴订单号跳转」改为「交易单号查询」：不强制先选租户；走管理端跨租户 GET；命中则全部租户视图展示匹配行并展开首条；未命中空态「未找到该交易单号」；空输入本地校验。
6. 保留 `POST .../order-number/parse/`（格式解析，无 DB），本页查询不再依赖它。

## 非目标

- 不提供未鉴权公网「只拿单号查单」。
- 不把 `order_number` 当作分片键或路径参数。
- 不改微信支付 `out_trade_no` 生成。
- 退款审批 Tab 本增量不改（订单 Tab 查询即可定位关联订单）。

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/system-admin/orders/?order_number=` | `IsPlatformStaff` + 网关已验证 | 跨租户 |
| GET `/api/tenant/{tid}/billing/orders/?order_number=` | 该租户管理员 | 仅 `tenant_id=tid` |

## 事件

纯查询，无新业务状态。**书面例外：不投递 MQ。**
