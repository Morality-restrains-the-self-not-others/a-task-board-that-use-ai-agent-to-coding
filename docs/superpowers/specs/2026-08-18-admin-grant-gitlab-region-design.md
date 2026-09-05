# 管理端赠送 GitLab 磁盘须选择区域

- **日期**: 2026-08-18
- **作者**: cursor
- **迭代**: admin-grant-gitlab-region
- **状态**: accepted（goal-mode 自动采用）
- **意图**: `docs/intents/backend/admin_grant_gitlab_region.intent.md`（并补充 B-049）
- **python_api_approval**: n/a（无新增 Python 接口；扩展既有 Go `taskBill` 赠送 API + Vue 赠送页）
- **架构**: **不升版**。v85 已完成可插拔多区域；本次补齐被遗漏的管理端赠送路径，无新服务/新库/新通信方式。
- **ADR**: 沿用 [ADR-0014](../../adr/0014-pluggable-multi-region-gitlab.md)；`No-ADR: covered by existing ADR-0014`

---

## 0. 完成标准（SMART）

1. `/system-admin/grant-points/` 资源类型为 **GitLab 磁盘 (GB)** 时，必须出现 **GitLab 区域** 下拉，选项来自 `GET /api/system-admin/gitlab-regions/` 的启用区域（`name`/`slug`）。
2. 未选区域时前端禁止提交；后端对 `gitlab_disk` / `gitlab_traffic` 缺 `region` 或无效 slug 返回 400，**禁止**回落到 `tencent-shanghai-5`。
3. 赠送成功后 `billing_tenant_gitlab_resource` 按 `(tenant_id, region)` 累加 `disk_gb`（或流量的 `traffic_prepaid_gb`）；`billing_resource_order_item.region` 写入所选 slug。
4. `task_post` 赠送行为不变（不要求、不展示区域）。
5. 回归测试覆盖：缺 region、非法 region、指定区域落库、两区域互不串、前端 POST body 含 region。

## 1. 问题

购买路径（`OrderCreate` / `order_payment`）已强制 `region`，主键为 `(tenant_id, region)`。  
管理端赠送 `adminGrantResources` 对 GitLab 磁盘/流量的 `INSERT` **未写 `region`**，依赖列默认值 `tencent-shanghai-5`。管理员无法把磁盘赠到 `tencent-sh-1` 等其它区。

页面元素：`select` 资源类型仅 `task_post` / `gitlab_disk` / `gitlab_traffic`，没有区选择。

## 2. 方案（已选定）

**在资源行上按类型显示区域选择器，API 增加可选字段 `region`（GitLab 类型必填）。**

拒绝的替代：

| 方案 | 拒绝原因 |
|------|----------|
| 把「GitLab 磁盘-上海五区」做成独立 resource_type | 区域动态，类型会爆炸 |
| 静默使用租户最近购买区 | 违反 v85「无平台默认、须显式选择」 |
| 仅改前端、后端仍默认五区 | 无法保证落库正确 |

### 2.1 API 契约（扩展既有，字段可选以保持 task_post 兼容）

`POST /api/tenant/{tid}/billing/accounts/admin_grant_points/`

```json
{
  "resources": [
    {
      "resource_type": "gitlab_disk",
      "quantity": 1,
      "expires_at": "2026-12-31 23:59:59.000000",
      "reason": "活动补偿",
      "region": "tencent-sh-1"
    }
  ]
}
```

- `region`：GitLab 区域 slug。`gitlab_disk` / `gitlab_traffic` **必填**且须命中 `billing_gitlab_region` 且 `is_active=1`。
- `task_post`：忽略 `region`。
- 内部 `POST /api/internal/taskbill/admin-grant-resources/` 同样解析 `region`（新用户礼包仍只送 `task_post`，不受影响）。
- 错误：`region required` / `region not found: {slug}`，HTTP 400；错误 JSON 沿用既有 `writeErrorJSON`。

### 2.2 落库

- `billing_tenant_gitlab_resource`：`INSERT ... (tenant_id, region, disk_gb, ...)` + `ON DUPLICATE KEY UPDATE disk_gb = disk_gb + VALUES(disk_gb)`（流量同理 `traffic_prepaid_gb`）。
- `billing_resource_order_item.region`：写入 slug（当前硬编码 `''`）。
- 成功后对该 slug 调用既有 `recalcRegionAllocated`。
- **不**套用用户限购 1GB（管理赠送可超限）；**不**改 provisioning 为 `pending_admin`（与现赠送语义一致：直接加配额）。

### 2.3 UI

- 资源类型为 `gitlab_disk` 或 `gitlab_traffic` 时展示「GitLab 区域」`<select>`（`data-testid="grant-gitlab-region"`）。
- 选项：`请选择区域` + `r.name`（value=`r.slug`）。列表 API 与 SystemAdmin GitLab 资源页相同。
- 提交校验：GitLab 行 `region` 非空。

### 2.4 事件

既有 `adminGrantResources` 无独立 Kafka 主题（仅 DB 流水 + 赠送订单）。本增量不新增业务意图，**不新增 MQ**。对照表书面例外见意图文档。

## 3. 🕸️ Code Review Graph 分析

- CRG `update --brief`：增量成功（3 files / 0 nodes 本轮基线）。
- `codegraph query adminGrantResources`：定义 `taskBill/src/admin_grant.go:145`；调用方 `handleAdminGrantResources`、`handleInternalAdminGrantResources`、`referral_commission`（仅 `task_post`）。
- 爆炸半径：`ResourceGrantInput` + 两处 JSON 解析 + GitLab upsert SQL + 订单行 `region`；前端仅 `SystemAdminGrantPoints.vue`。
- 购买路径 `order_payment.go` 已按 region 写入，作为实现样板，不改购买 API。

## 4. 架构判断

无需新 `.puml` / `.archimate`：无新 Application_Component、无新数据所有权、无新 Rel_Flow。数据仍在 `task_bill` / `billing_tenant_gitlab_resource`。
