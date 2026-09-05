# 设计：管理端待分账订单列表

- **日期**: 2026-08-22
- **入口**: `https://www.daydaymoney.com/system-admin/order-records` Tab 栏（当前仅「订单记录 / 退款审批」）
- **架构版本**: v99 target
- **ADR**: 沿用 [ADR-0030](../../adr/0030-admin-only-order-profit-sharing.md)（分账仅平台员工可见）；本增量不新增 ADR

## 背景

运营在订单记录页 Tab 栏只能切「订单记录」与「退款审批」。有推荐关系的已支付订单会写入 `billing_profit_sharing`（默认延迟 8 天再向微信发起），但**没有一张待分账队列**：运营必须在全量订单里逐单展开才能看到 `pending` / `processing` / `failed`。用户期望该 Tab 栏增加「待分账订单列表」。

用户输入无 `data-traceId`。CRG：`code-review-graph update --brief` 增量 19 文件、0 节点、风险 0.00。`codegraph` CLI 可用；无 MCP `codegraph_explore`。既有实现：`processPendingProfitSharings`、`listProfitSharingForOrder`、订单展开分账块。

## 根据当前架构的理解

- 企业景观 current：v98 指令闲置回收；v97 Git OAuth site 寻址仍为正交 target
- 应用层：taskFE 管理端订单页；taskBill 持有 `billing_profit_sharing`；分账执行由 taskEvents timer 调 internal 一次性 API
- 数据：`billing_profit_sharing` 属 taskBill / `task_bill` utf8mb4；状态 `pending|processing|finished|failed`
- 本增量**不改**分账发起、比例、微信调用；只补**管理员只读队列**

## 方案（已采纳）

1. **第三 Tab「待分账」**：`SystemAdminOrderRecords` 增加 `tab=profit-sharing`。默认仍为订单记录；`?tab=refund` 行为不变。
2. **新只读列表 API**（扩展现有 Go `taskBill`，禁止 Python）：
   - `GET /api/system-admin/profit-sharing/`（underscore 别名 `/api/system_admin/profit-sharing/`）
   - 鉴权与订单列表相同：`X-Gateway-Auth-Verified=1` + `authz.IsPlatformStaff`
   - Query：`status`、`limit`（默认 20，最大 50）、`offset`
   - `status`：`pending` / `processing` / `failed` / `finished` / `open`（默认 `open` = pending+processing+failed，即未完成队列）
   - 排序：`settle_after ASC, id ASC`（最早可分账的在前）
3. **响应不含 openid / 微信分账单号**（ADR-0030 延续）。
4. **UI**：新面板 `SystemAdminProfitSharingPanel`：表格 + 状态筛选 + 分页 + 刷新。订单号用既有 `buildSystemAdminOrderRecordsHref` 链到订单 Tab。Anti-Replay-OK：只读列表/刷新/筛选，无写按钮。
5. **网关**：在 `taskGateway/routes/routes.yaml` 新增前缀（priority 高于 django `/api/system-admin/` 通配），`routes-apply` 生成 `apisix.yaml`。
6. **无新 MQ 事件**：纯查询；写入仍由支付完成 `markOrderForProfitSharing`。

### 列表项契约

```json
{
  "items": [
    {
      "id": "string",
      "order_id": "string",
      "order_number": "string",
      "tenant_id": "string",
      "receiver_user_id": "string",
      "total_yuan": "10.00",
      "amount_yuan": "0.50",
      "amount_yuan_cents": 50,
      "status": "pending",
      "settle_after": "RFC3339",
      "settled_at": "",
      "fail_reason": ""
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

## 备选（否决）

| 方案 | 否决原因 |
|------|----------|
| 在现有订单列表加 `?profit_sharing_status=` | 订单列表以 `billing_resource_order` 为行，一单可能无分账行；待分账队列应以分账记录为行 |
| 仅前端过滤已加载订单 | 全量订单分页无法得到「所有待分账」 |
| 管理端手动「立即分账」按钮 | 超出本次「显示列表」范围；写路径需独立 NFR/幂等（记 OPT） |

## 非目标

- 不改微信分账执行、延迟、比例、timer
- 不提供重试/立即分账写接口
- 不展示 openid / wechat_profit_sharing_id
- 不给租户暴露本列表
- 不新增 Python/Django 接口

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/system-admin/profit-sharing/` | 平台员工 | 跨租户只读队列 |
| FE `/system-admin/order-records/?tab=profit-sharing` | 平台员工（既有管理壳） | 同上 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员查看待分账队列 | — | — | — | 纯查询；记录已在支付完成路径写入 |
| 切换待分账 Tab | — | — | — | 纯前端路由 query |

## 领域概念（供 DDD）

- Bounded context：Billing（taskBill）
- Entity：已有 `ProfitSharingRecord`
- 新读模型：`ProfitSharingQueueItem`（列表投影，无 openid）
- 不新增聚合写边界

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：19 files, 0 nodes, risk 0.00
- 相关符号：`handleSystemAdminListOrders`、`listProfitSharingForOrder`、`processPendingProfitSharings`、`SystemAdminOrderRecords.vue`
- 影响面：新增独立 handler，不改 `orderJSON` / 租户 GET；Tab 切换须保持 refund query 兼容

## 架构交付物

- `docs/architecture/v99-enterprise-landscape-20260822-2155-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
- `docs/architecture/v99-application-integration-20260822-2155-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
