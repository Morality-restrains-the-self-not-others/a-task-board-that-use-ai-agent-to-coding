# 租户购买 GitLab 资源须按行选择区域

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: tenant-purchase-gitlab-region
- **状态**: accepted（goal-mode 自动采用）
- **意图**: `docs/intents/backend/tenant_purchase_gitlab_region.intent.md`（B-049b，补齐 B-049）
- **python_api_approval**: n/a（扩展既有 Go 下单 API + Vue 购买页）
- **架构**: **不升版**。v85 已完成可插拔多区域；购买页已有「整单共享」区域框，与管理端赠送「按资源行选区」不对齐，且 `createOrder` 只校验非空、不校验启用 slug。
- **ADR**: 沿用 [ADR-0014](../../adr/0014-pluggable-multi-region-gitlab.md)；`No-ADR: covered by existing ADR-0014`

---

## 0. 完成标准（SMART）

1. `/tenant/:tid/billing/orders/create/` 在 VIP1 可见的 **GitLab 磁盘与流量合卡**内有**一个**共用区域下拉（启用区 `name`/`slug`）；未选区不得 POST。
2. 同一订单的磁盘行与流量行写入**同一**所选 slug（合卡共用下拉）；后端仍按行存储 `billing_resource_order_item.region`。
3. `createOrder` 对 GitLab 行调用 `getGitlabRegionBySlug`：空 → `region required`；未知/停用 → `region not found`；禁止静默 `tencent-shanghai-5`。
4. 订单详情展示 GitLab 行的 region。
5. 租户 GitLab 设置页「购买 GitLab 资源」链到订单创建页（真实 `href`），不再链到仅展示价目的 `/pricing`。
6. 不改 VIP1 门槛、不改 1GB 限购、不改支付后 `pending_admin`。

## 1. 问题

公网 SPA 已有整单「GitLab 区域」选择器（`order-gitlab-region`），但：

- 选区不贴在「磁盘」资源上，与刚落地的赠送页心智不一致；
- 磁盘/流量被迫分卡重复选区（2026-08-23 已改为合卡共用下拉）。
- 后端接受任意非空字符串；
- GitLab 设置页购买入口指向 `/pricing`，无法下单选区。

## 2. 方案（已选定）

**购买页按资源卡片内嵌区域选择（对齐赠送页）；API 校验启用 slug。**

拒绝：把区域做成独立 `resource_type`；静默默认区；去掉 VIP1 门槛。

### 2.1 API

既有 `POST /api/tenant/{tid}/billing/orders/` `items[].region`：GitLab 类型必填且须 `is_active=1`。

### 2.2 UI

- 合卡 `data-testid="order-gitlab-resources"`：磁盘数量 + 流量数量 + **一个**区域下拉 `data-testid="order-gitlab-region"`
- 不再提供独立流量区域下拉 `order-gitlab-traffic-region`（2026-08-23 布局调整）
- 列表：`GET /api/billing/gitlab-regions/tenant_id/{tid}/`

### 2.3 事件

下单成功仍走既有 `resource_order_created` 日志；支付发放仍走 `markOrderPaid` + 既有账单 outbox。不新增 Kafka。书面例外见意图文档。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief`：增量图 4 files / 0 新节点。手工调用链：`handleCreateOrder` → `createOrder` → `markOrderPaid`；FE `OrderCreate` / `OrderDetail` / `WorkspaceSettingsGitlabConnection`。无第二套租户下单写入点（直购 `/purchase/` 已强制 region）。
