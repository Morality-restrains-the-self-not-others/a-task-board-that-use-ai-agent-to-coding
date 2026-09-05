# 订单详情页 — NFR 澄清

L2 Standard；支付路径安全按计费域 L3。

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| `GET/POST /api/tenant/{tenant_id}/billing/orders/` | tenant_id | ✅ 租户隔离键，与账单库归属一致 | 无 |
| `GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` | tenant_id + order_id | ✅ tenant 为分片键；order_id 为实体键 | 无新路径 |
| FE `/tenant/:tenant/billing/orders/create/` | tenant | ✅ | 无 |
| FE `/tenant/:tenant/billing/orders/:orderId/` | tenant | ✅ 新增展示路由，键合适 | 实现该路由 |

无可伸缩性 L0 路径。可伸缩性类别：L1（租户级，沿用现表，不新增分片）。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L1 | 单租户读一单，无列表扫描 |
| 安全 | L3 | 计费读路径；错误不暴露内部细节；`data-traceId` |
| 可用性 | L2 | GET 失败可重试；创建失败不跳转 |
| 性能 | L2 | 单次 GET，无 N+1 |
| 一致性 | L2 | 详情以 GET 为准（创建响应不作为唯一真相） |

## 质量场景

- 刺激：创建成功 → 响应：1s 内进入详情且 items 可见。
- 刺激：GET 失败 → 响应：详情页错误文案 + `data-traceId`，不白屏。

## 领域模型影响

无。不改 ResourceOrder 聚合。
