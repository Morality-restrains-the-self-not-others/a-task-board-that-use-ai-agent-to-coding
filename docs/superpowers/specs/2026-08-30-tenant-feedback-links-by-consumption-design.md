# 意见与建议链接（按租户累计消耗可见）— 设计文档

- **日期**: 2026-08-30
- **状态**: accepted
- **迭代**: tenant-feedback-links-by-consumption
- **作者**: cursor
- **架构**: v121 🎯 target（基于 v120 current：任务与项目内容历史版本）

## 背景

租户控制台侧栏（`tenant-console-sidebar`）现有「项目列表 / 工作面板 / 镜像市场 / 人员管理 / 设置 / 资源与订单」，**没有**「意见与建议」。平台超管需要配置多组外链（问卷、社群、专属支持等），展示在租户侧栏「意见与建议」子菜单中。

可见度按**当前租户各类资源的历史累计消耗**做「达到即可见」（≥）：一组可绑多种资源阈值，**必须全部达到**才显示该组；消耗更高的租户会同时看到阈值更低的组（嵌套包含，不是互斥档位）。同一租户全体成员看到同一套链接（不分付费方 / 实际使用方）。

金额是资源种类之一（租户合计已消费），不是唯一口径。资源种类会继续增加（任务帖、GitLab 磁盘/流量、未来新品类）。

## 成功标准

1. 超管在系统管理可 CRUD **链接组**（组名、排序、启用）及组内 **阈值列表 + 链接列表**。
2. 租户侧栏新增一级「意见与建议」；子菜单**按组名分段**，段下为该组链接（真实 `<a href>`，新标签打开）。
3. 可见判定：对组内每个 `(resource_kind, min_quantity)`，租户累计消耗 `>= min_quantity`（AND）。阈值列表为空 → 所有租户可见。
4. 同一资源种类上，阈值更低的组是更高消耗租户的**子集可见**：例如帖数 200 的租户同时看到「帖 ≥ 0」与「帖 ≥ 100」两组。
5. 不同资源种类的组互相独立：高流量不自动解锁高磁盘组。
6. 前端不得自行按消耗过滤；服务端只返回可见组。
7. 不新增 Python 接口。

## 方案（选定）

Owner：**taskBill**（消耗数字与计费资源种类的唯一真相；配置与可见性求值同库）。

### 可见性规则（SSOT）

```
visible(group, tenant) <=>
  ∀ t ∈ group.thresholds:
    tenantLifetimeConsumed(t.resource_kind) >= t.min_quantity
```

- 未配置的 `resource_kind` 不参与 AND（不是 0 门槛）。
- 租户没有该种类记录 → 消耗视为 `0`。
- `min_quantity` 允许 `0`：表示「该种类消耗 ≥ 0」，即不额外限制（与「不配这一条」等价；超管 UI 可不鼓励重复配置）。
- **不按用户个人用量、不按账期重置。**

「消耗多的能看见消耗少的链接」由上述谓词自然成立：组 B 的每条阈值都 ≥ 组 A 的对应种类时，满足 B 必满足 A。种类集合不同时两组不可比，按各自 AND 独立显示。

### 资源种类目录（可扩展）

首期内置（与 `taskBill` `ResourceType*` + 金额对齐）：

| `resource_kind` | 单位 | 租户累计口径 |
|-----------------|------|----------------|
| `task_post` | 帖 | 历史已消耗任务帖数量（授权 − 剩余，含赠送/购买） |
| `gitlab_traffic` | GB | 历史累计已用流量 |
| `gitlab_disk` | GB | **当前已用容量**（磁盘是存量资源，不做 GB·月积分；与「累计消耗」最接近的可核验数） |
| `consumed_amount` | 分（人民币分） | 该租户账本**已消费金额**合计（与超管「消费情况」同计价单位，但是**租户合计**不是按用户） |

新增计费 `resource_type` 时：在目录表登记 `kind / 显示名 / 单位 / 取数实现`，超管下拉自动出现。禁止在链接组里写未登记 kind。

### 数据（`dataMigrate/taskBill/074_feedback_link_groups.sql`）

表名前缀 `billing_`。主键 Snowflake。年增量远小于 10 万，**不分区**。

**`billing_feedback_link_group`**

| 列 | 说明 |
|----|------|
| `id` | Snowflake PK |
| `name` | 组名（子菜单分段标题） |
| `sort_order` | 升序 |
| `enabled` | 0/1；禁用后租户不可见 |
| `created_at` / `updated_at` | DATETIME |

**`billing_feedback_link_threshold`**

| 列 | 说明 |
|----|------|
| `id` | Snowflake PK |
| `group_id` | FK 语义（无物理 FK） |
| `resource_kind` | 目录 kind |
| `min_quantity` | `DECIMAL(20,6) NOT NULL` |
| `UNIQUE(group_id, resource_kind)` | 一组一种资源一条 |

**`billing_feedback_link`**

| 列 | 说明 |
|----|------|
| `id` | Snowflake PK |
| `group_id` | 所属组 |
| `title` | 子菜单链接文案 |
| `url` | 仅 `https:`（实现时拒绝 `javascript:` / `http:` / 相对路径） |
| `sort_order` | 组内升序 |
| `enabled` | 0/1 |

**`billing_feedback_resource_kind`**（目录 seed）

| 列 | 说明 |
|----|------|
| `kind` | PK，如 `task_post` |
| `display_name` | 超管下拉文案 |
| `unit` | 展示单位 |
| `enabled` | 下线某种类时关 |

聚合根：**链接组**。一次超管保存覆盖该组阈值 + 链接。

### API（Go `taskBill`，Swagger 同步）

**超管（需系统管理员）**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/system-admin/feedback-link-groups/` | 全部组（含禁用）+ 阈值 + 链接 |
| GET | `/api/system-admin/feedback-resource-kinds/` | 目录 |
| PUT | `/api/system-admin/feedback-link-groups/{id}/` | 整组保存；`Idempotency-Key` |
| POST | `/api/system-admin/feedback-link-groups/` | 新建；`Idempotency-Key` |
| DELETE | `/api/system-admin/feedback-link-groups/{id}/` | 删组及子行；`Idempotency-Key` |

**租户（成员即可，见权限）**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/{tenantId}/billing/feedback-links/` | **只返回** `enabled` 且对当前租户可见的组；组内只含 `enabled` 链接。不回传阈值给前端（避免被用来反推消耗）。 |

求值在 handler 内读租户累计消耗 map，过滤后返回。纯查询，无事件。

### 权限

- 超管 CRUD：现有系统管理员门禁（`X-Gateway-Auth-Verified` + `authz.IsPlatformStaff`）。
- 租户 GET：**v72 page ⊃ ui_region**。page `nav.feedback`，region `nav.feedback.main`。Handler 使用 `authz.RequireRegionView("nav.feedback.main", tenantId)`。禁止仅用粗码作为 API 门禁。
- 侧栏：`canSeeMenuKey('nav.feedback')` 优先 `page:nav.feedback`，回退粗码 `feedback:view`。
- 内置角色 `tenant_admin` / `member` / `group_admin` 默认绑定该 page。访问管理可关掉。
- 无 region：侧栏不显示「意见与建议」；直链 GET 403。

详见 `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-permission-analysis.md`。

### 前端

- `SystemAdminSidebar` 运营管理增加「意见与建议链接」；新页 CRUD 组/阈值/链接。写按钮：`createClickGuard` + `Idempotency-Key`。
- `Sidebar.vue` 在「资源与订单」旁增加一级「意见与建议」（带子菜单）。**该文件已接近 500 行门禁，实现时必须先拆出导航段**（例如 `TenantConsoleFeedbackNav.vue`），禁止继续堆行。
- 子菜单：`v-for` 可见组 → 组名（非链接）→ 其下真实 `<a :href="url" target="_blank" rel="noopener noreferrer">`。`Anti-Replay-OK: 真实外链 href，无写请求`。禁止 `@click.prevent` + `router.push`。
- 缩窄态：与「资源与订单」相同，点击展开侧栏再展开子菜单。
- 侧栏加载：进入租户壳时 GET 一次；禁止 `setInterval` 轮询。消耗变化后下次进页/刷新即更新。

### 领域事件

| 业务意图 | 事件名 | topic（约定） | 发布点 | 消费者 |
|---------|--------|---------------|--------|--------|
| 新建链接组 | `FEEDBACK_LINK_GROUP_CREATED` | `feedback-link-group-created` | 超管 POST 提交成功 | 审计/日志；无强制下游写 |
| 更新链接组（含阈值/链接） | `FEEDBACK_LINK_GROUP_UPDATED` | `feedback-link-group-updated` | 超管 PUT 成功 | 同上 |
| 删除链接组 | `FEEDBACK_LINK_GROUP_DELETED` | `feedback-link-group-deleted` | 超管 DELETE 成功 | 同上 |
| 租户拉取可见链接 | — | — | — | 纯查询 |

投递走 taskBill 既有 `publishEvent` / outbox 模式；payload：`group_id`、操作者、变更摘要（不含完整 URL 列表以外的密钥；URL 为公开配置）。幂等键 = `group_id` + 事件类型 + 版本/`updated_at`，禁止用 `tenant_id`。

### 行数 / 安全

- 侧栏拆分见上。
- URL 服务端校验 scheme=`https`。
- 日志不打印完整超管会话 token。

## 拒绝方案

| 方案 | 原因 |
|------|------|
| 前端按消耗过滤 | 可伪造；链接会泄露给未达阈值租户 |
| 按登录用户个人消耗 | 已否决；全员同租户同套链接 |
| 一组一种资源 | 已否决；一组多种 AND |
| 严格 `>` | 已否决；改为 **≥ 达到即可见** |
| 新独立微服务 | 消耗与目录都在 taskBill，拆服务无边界收益 |
| Django 配置页 | Go-first；Python 门禁不触发 |

## Domain Concept Inventory

- **Bounded Context**: Billing / 平台运营配置（taskBill）
- **Key Entities**: FeedbackLinkGroup（聚合根）、FeedbackLink、FeedbackLinkThreshold、FeedbackResourceKind
- **Candidate Aggregates**: FeedbackLinkGroup
- **Domain Events**: 见上表
- **消耗快照**: 非新实体；从既有 grant/usage/账本投影为 `map[kind]quantity`

## 价值流影响

现有 `value-stream.yaml` 无「意见与建议」流。建议 Step 4 新增：

- **domain**: 组织与成员（或独立「平台运营」）
- **stream**: `tenant-feedback-links`
- **steps（planned）**: `admin-configure-groups`、`tenant-sidebar-visible-links`
- **fields**: `task-bill.billing_feedback_link_group.name`、`task-bill.billing_feedback_link.url`、`task-bill.billing_feedback_link_threshold.resource_kind`、`task-bill.billing_feedback_link_threshold.min_quantity`

不改现有 billing 下单/用量步骤。测试点：超管 CRUD + 租户侧栏可见性矩阵。

## 🕸️ Code Review Graph 分析

- `code-review-graph status`：图存在（约 114 nodes / 18 files），MCP 命名空间本会话不可用。
- CLI `search` 参数与当前版本不兼容（`unrecognized arguments: --brief`）。
- 静态落点：`taskFE/app/src/components/Sidebar.vue`、`SystemAdminSidebar.vue`、`tenantConsoleNav.js`；`taskBill/src/orders.go`（`ResourceType*`）、消耗汇总（任务帖回放、`gitlab_disk_used_gb`、流量账本、user-recharge-consumption 为**用户**维，本功能需**租户**维合计）。
- `CRG unavailable: MCP not registered; graph too small to cover taskBill Go symbols.`

## 🐍 Python 新增接口

**不触发。** 全部新 HTTP 落 Go `taskBill`。`python_api_approval: not_applicable`

## 🏛️ 架构变更影响

- **迭代版本**: v121 🎯 target
- **迭代名称**: 意见与建议链接（租户累计消耗 ≥ 可见）
- **作者**: cursor
- **设计日期**: 2026-08-30 09:22
- **新增文件**（每个视图四类伴生，缺一不可）:
  - `docs/architecture/v121-application-integration-20260830-0922-cursor.puml`
  - `docs/architecture/v121-enterprise-landscape-20260830-0922-cursor.puml`
  - 各视图 `.diff.archimate`（增量变迁）+ `.full.archimate`（本迭代全量拓扑）+ `.mermaid.md`
- **已有文件（未修改）**: v120 current（任务与项目内容历史版本）
- **变更明细**:
  - 🟢 `billing_feedback_link_group` / `_threshold` / `_link` / `_resource_kind`；事件 `FEEDBACK_LINK_GROUP_*`
  - 🟡 taskBill 超管+租户 API；taskFE 侧栏与超管页
  - 🔴 无

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v120 → Gap「侧栏无按消耗分层的意见链接」→ WP → Plateau v121；变更数据流 |
| **`.full.archimate`** | 本迭代完整拓扑（超管/租户 → FE → GW → taskBill → 表与事件） |
