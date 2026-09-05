# 账单页 GitLab 磁盘/流量按区域展示已用量

- **日期**: 2026-08-25
- **作者**: cursor
- **迭代**: billing-gitlab-quota-region-usage
- **状态**: accepted（goal-mode 自动采用）
- **意图**: `docs/intents/frontend/billing_gitlab_quota_region_usage.intent.md`（F-100）
- **python_api_approval**: n/a（无 Python 新 API；复用既有 Go `GET /billing/quotas/`）
- **架构**: **不升版**。配额账本、按区存储、用量同步（磁盘 `disk_used_bytes` / 流量 `traffic_used_gb`）已在 v85/ADR-0014 与后续 GitLab 用量门禁落地。本增量只改账单首页展示，不新增服务/表/事件。
- **ADR**: `No-ADR: covered by existing ADR-0014` + 既有 `gitlab_resources[]` 契约

---

## 0. 完成标准（SMART）

1. `/tenant/:tid/billing/`「资源配额」中 **GitLab 磁盘与流量按区域分卡**，不再用一对无区域标签的汇总绿/橙卡作为主展示。
2. 每个区域同时显示 **已用 / 配额**（磁盘 GB、流量 GB），磁盘保留到期日；赠送/购买拆分仍可见。
3. 有 `gitlab_resources[]` 时隐藏无区域汇总卡；无列表时回退到聚合字段，且聚合卡也显示已用量。
4. 用量数字与 GitLab 设置页同一口径（最多 6 位小数、去掉尾随 0）。
5. Vitest 覆盖多区、单区、无列表回退、赠送拆分；不新增写接口。

## 1. 问题

公网账单页用户选中的两个元素是：

- 绿卡「GitLab 磁盘 1 GB 到期：2027-03-18」
- 橙卡「GitLab 流量 1 GB」

问题：

1. **未按区域划分**：两卡读 `gitlab_disk_gb` / `gitlab_traffic_prepaid_gb` 聚合字段；多区时会把不同区的配额叠在一起或只反映最近一区。页底虽有「按区域配额」小字行（OPT-20260818-022），视觉权重远低于这两张卡。
2. **未显示已用量**：后端 `gitlab_resources[]` 已含 `disk_used_gb` / `traffic_used_gb`，设置页已展示「已用 / 配额」，账单首页没用。

## 2. 方案（已选定）

**用按区卡片替换汇总 GitLab 卡，卡片主数字为「已用 / 配额」。**

拒绝：

- A. 只在小字行加用量、保留无区域汇总卡 — 用户点选的就是那两张卡。
- B. 新开用量 API / 轮询 GitLab — 违反「禁止无触发后台轮询」；用量已在 quotas 响应中。
- C. 跨区合计再分摊 — 磁盘/流量按区账本，合计无业务意义。

### 2.1 API（不改契约，只补测）

既有 `GET /api/tenant/{tid}/billing/quotas/`：

| 字段 | 用途 |
|------|------|
| `gitlab_resources[]` | 主展示：`region`/`region_name`/`disk_gb`/`disk_used_gb`/`traffic_prepaid_gb`/`traffic_used_gb`/`disk_expires_at`/`disk_gifted_gb`/`disk_purchased_gb`/`traffic_gifted_gb`/`traffic_purchased_gb`/`gitlab_web_url` |
| `gitlab_disk_gb` / `gitlab_traffic_prepaid_gb` / `gitlab_*_used_gb` | 无列表时的回退 |

Go 单测断言列表项含 `disk_used_gb` / `traffic_used_gb`（字段已由 `regionResourceView` 下发）。

### 2.2 UI

- 任务帖配额卡保持不变。
- `gitlab_resources.length > 0`：按区渲染 `GitlabRegionQuotaCards`（每区：区域名 + 绿磁盘卡 + 橙流量卡）。
- 每卡：`已用 / 配额 GB`、可选进度条（已用/配额，封顶 100%）、到期、赠送/购买、进入仓库链接。
- 保留 `data-testid="gitlab-region-quota-rows"` 以免拆测；新增 `data-testid="gitlab-region-disk-card"` / `gitlab-region-traffic-card`。
- 无 `gitlab_resources`：显示原绿/橙卡，但数字改为 `已用 / 配额`。

### 2.3 事件

纯查询展示。无新 MQ。书面例外见意图文档。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief`：4 files / 0 新节点。调用链：

- FE `BillingDashboard.fetchQuotas` → `GET .../billing/quotas/`
- Go `handleResourceQuotas` → `listGitlabResourceViews` → `regionResourceView`（已含 used 字段）
- 设置页 `useGitlabResourcePurchase.applyView` 已映射 `disk_used_gb`/`traffic_used_gb`（对照口径，不改行为）

同类问题搜索：账单页是唯一「无区域标签的 GitLab 配额汇总卡」。设置页、超管租户面板已按区+已用。

## 3. Python 新 API 门禁

not_applicable — 无 Django/Flask 新路由。
