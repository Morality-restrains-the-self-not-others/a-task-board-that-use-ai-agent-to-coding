# 订单 ID → 租户 ID 与分片共片 — 设计

- **Date:** 2026-08-19
- **Status:** accepted（/goal 自动采用方案 B′）
- **Architecture change:** 否（无新服务 / 无新公网 API / 无数据所有权变更；不升级 ArchiMate）
- **ADR:** [ADR-0018](../../adr/0018-order-id-tenant-shard-routing.md)（proposed）
- **关联:** [ADR-0017](../../adr/0017-globally-unique-resource-order-number.md) 的**全局唯一 + 禁止日序号**仍有效；**新单展示号格式**由本设计修订（日期后插入 `tenantId`）。

## Goal / 成功标准

1. 客服/用户咨询只贴展示订单号，**不必另交租户 ID**，系统能解析出租户并定位订单。
2. 技术主键仍是末段 Snowflake；URL 仍为 `/tenant/{tid}/billing/orders/{id}/`。
3. 分片键是 `tenant_id` 列，不按订单 `id` 哈希；Snowflake 位布局不变。
4. 新单 `order_number` 落在 `VARCHAR(64)` 内；存量三节格式仍可展示。
5. 不新增 Python 接口；不新增匿名「只凭订单号查单」公网 API。

## 当前架构理解（基线）

根据 `docs/architecture/` **v85 current**（2026-08-18 15:50）：

- 视图：`enterprise-landscape`、`application-integration`
- 应用层：`taskBill` 拥有 `billing_resource_order`；`taskFE` 展示 `order_number`，链接用 Snowflake `id`
- 技术层：单实例 MySQL `task_bill`，尚未按租户水平分片
- 本次只约定 ID / 展示号 / 查询契约，不改服务拓扑

## 修订后的决策（方案 B′）

用户确认采用：

```text
ORD-{UTC yyyyMMdd}-{tenantId}-{snowflakeId}
```

例：`ORD-20260818-877397588196749312-877596007691485184`

| 字段 | 角色 |
|------|------|
| `ORD` | 固定前缀 |
| UTC 日期 | 人读对账（与 ADR-0017 相同） |
| `tenantId` | 展示号内的租户基因；咨询/分片路由用 |
| `snowflakeId` | **唯一技术主键** = `billing_resource_order.id`；URL path 仍用它 |

生成：`order_number = f(tenantID, orderID, UTC date)`，仍禁止 `SELECT MAX`。

解析（按 `-` 切分）：

| 段数 | 形态 | 租户来源 | 主键 |
|------|------|----------|------|
| 4 | 新单 | 第 3 段 `tenantId` | 第 4 段 |
| 3 | ADR-0017 `ORD-日期-{id}` 或存量 `ORD-日期-NNN` | 不能从号码得出；单库 PK 查 / 分片后定位表或请用户补租户 | 第 3 段（NNN 不是 Snowflake，只能按 `order_number` 查） |

长度：`ORD-` + 8 + `-` + 最多 19 + `-` + 最多 19 = **≤ 52**，`VARCHAR(64)` 足够。

### 咨询场景（只贴订单号）

1. 解析四段号 → `(tenantHint, orderID)`。
2. `WHERE tenant_id = ? AND id = ?`（分片可路由，且防错片）。
3. **校验**：行上 `tenant_id` 必须等于 `tenantHint`；不一致视为不存在（号码被改过中间段）。
4. **授权**：解析出的租户只用于路由/定位，不能代替登录身份。租户用户仍走带 `tid` 的 URL；客服/超管用管理端权限。
5. **不**新增匿名公网「粘贴 ORD 即出订单详情」接口（防枚举）。管理端/内部工具可解析后跳转 `/tenant/{tid}/billing/orders/{id}/`。

### 共片（不变）

```text
shard = hash(tenant_id) % N
```

基因在**展示号**里，不在 64-bit PK 里。共片仍靠列；基因让「只有展示号」时也能先得到 `tenant_id` 再选片，不必定位表（新单）。存量三节号分片后仍要定位表或补租户。

禁止按 `id` 哈希分片。禁止改 `shareLib/snowflake` 位图。

### 查询契约

- 租户面：`WHERE tenant_id=? AND id=?`（path 上的 `tid` 与号码内基因应一致，否则 404）。
- 微信 pending 已存 `tenant_id`，不改 `out_trade_no`。
- 本日不建 `billing_order_locator`；若需支撑海量**存量三节号**的无租户查询，分片当天再补。

## 备选方案

| # | 方案 | 结论 |
|---|------|------|
| A | 展示号不含租户，只靠 URL/PK | 不满足「咨询不另交租户 ID」 |
| B′ | `ORD-{日期}-{tenantId}-{snowflakeId}` | **采用** |
| B | `ORD-{tid}-{date}-{oid}` | 拒绝：日期不在前，对账扫视差于 B′ |
| C | Instagram 低位片号 | 拒绝：破坏统一 Snowflake |
| D | 64-bit PK 内塞完整 tenant | 拒绝：塞不下 |
| E | 按 order `id` 哈希分片 | 拒绝：同租户打散 |

ADR-0017 Alternative 4 原拒绝「号码嵌 tenant」（长度、URL 已有租户）。现产品明确要咨询零补租户，该拒绝被本 ADR **格式条款**取代；**全局唯一、禁止日序号**仍有效。

## 路径分片键审视

| 路径 | 分片键 | 动作 |
|------|--------|------|
| `/api/tenant/{tid}/billing/orders/` | `tid` | 保持 |
| `.../orders/{oid}/` | `tid`；`oid` 为实体键 | SQL 含 `tenant_id`；可与号码第 3 段交叉校验 |
| 管理端粘贴四段 ORD 号 | 从号码解析 `tid` | 内部解析 + 复合查询；非公网匿名 API |
| 微信回调 | pending.`tenant_id` | 不改 |

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：展示号派生与只读定位，无新业务事实。

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 生成带租户基因的展示号 | — | — | — | 沿用 `resource_order_created` 日志 |
| 凭展示号解析租户并定位 | — | — | — | 只读 |

## Domain 概念（供 /6-ddd）

- Entity: `ResourceOrder`（根 PK = `id` Snowflake；归属 `tenant_id`）
- VO: `OrderNumber` = `ORD-{utcDate}-{tenantId}-{orderId}`
- 解析函数: `ParseResourceOrderNumber(s) → (date, tenantID, orderID, format)`
- 分片键: `tenant_id`（列），展示号基因为其拷贝

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | Nodes: 0；Last updated: never |
| 关键发现 | 图未 build |
| 决策影响 | `generateOrderNumber`、`createOrder` / `insertOrderWithRetry`、`loadOrder`、FE 列表换行、`orders_number_test.go` |
| skip 理由 | `unavailable`：图为空 |

`CRG unavailable: graph.db present but 0 nodes.`

## 价值流影响

- 既有计费/下单/支付流；不新增 stream。
- 字段仍为 `task-bill.billing_resource_order.order_number`（值格式变）与 `.id` / `.tenant_id`（角色不变）。
- 测试：生成四段号、解析、基因与行不一致 → 不存在；两租户号不同且各自含自己的 `tenant_id` 与 `id`。

## 🏛️ 架构变更影响

- **不新建** ArchiMate v86。
- 约束 SSOT：ADR-0018（批准后 accepted）；ADR-0017 唯一性条款保留。

## Python 新增接口

不触发。

## 实现范围（仅批准后）

1. `generateOrderNumber(tenantID, orderID int64)` → 四段号；`tenantID<=0` 或 `orderID<=0` 报错。
2. `ParseResourceOrderNumber`：四段 / 三段分流。
3. `loadOrder(tenantID, orderID)`：`WHERE tenant_id=? AND id=?`。
4. 管理端：若有按号码跳转，优先解析四段再进租户 URL。
5. FE：继续 `break-all`（号码约 50 字符）。
6. **不做**：改 PK 类型、改 Snowflake 算法、改 `out_trade_no`、建 locator、改写存量号码。
