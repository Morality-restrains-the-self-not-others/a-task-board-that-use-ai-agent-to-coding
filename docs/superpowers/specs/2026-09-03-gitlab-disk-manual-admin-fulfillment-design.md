# GitLab 磁盘购买改为系统管理员手动发货开通

- **日期**: 2026-09-03
- **状态**: accepted（/goal 覆盖 USER GATE，自动采用）
- **迭代**: gitlab-disk-manual-admin-fulfillment
- **作者**: cursor
- **意图**: `docs/intents/backend/gitlab_disk_manual_admin_fulfillment.intent.md`
- **python_api_approval**: not_applicable（无新增 Python HTTP 接口；扩展既有 Go `taskBill` + `taskFE`）

## 背景

用户在价格管理/购买链路看到 GitLab 磁盘可下单支付。产品要求：**购买后必须由系统管理员手动发货开通**。现场体感是「付完就自动开通」。

代码里已经有一半「人工履约」骨架：

- `resourceRequiresManualFulfillment(gitlab_disk)` = true（留言、施工说明）
- `markOrderPaid` 把磁盘行写成 `provisioning_status=pending_admin`
- SystemAdmin「开通实施」走 `POST /api/tenant/{tid}/billing/gitlab-resources/provision/`
- 租户 GitLab 连接页文案「等待管理员开通实施」

真正自动发货的是另一条隐式路径：`regionResourceView`（租户或管理员 GET 配额）在 `pending_admin` 且 `disk_used_bytes<=0`、区域不是 `pending_node` 时，**后台 goroutine 调用 `ensureTenantGitlabGroupForRegion`**，直接在 GitLab 上建组/设限额。组已经存在，徽章却仍是「待开通」。腾讯云已部署区域（`infra_status=ready`）会稳定走这条自动发货。

## 对当前架构的理解

根据 `docs/architecture/` current（v129）：

- 共有 2 个 **current** 视图：`enterprise-landscape`、`application-integration`（**v129** ✅）。
- 业务层：计费购买、GitLab 区域选购（ADR-0014）、阿里云目录先行 + 人工建节点（v128 / ADR-0057）。
- 应用层：`taskFE` OrderCreate / SystemAdminGitlabTenantPanel；`taskBill` 资源订单、`billing_tenant_gitlab_resource`、hybrid/ensure 组。
- 技术层：腾讯云 GitLab CE 已部署；阿里云区域多为 `pending_node`。
- 上次交付版本 **v129**（2026-09-03）是 runAll 金丝雀重启，与本需求无关。

📋 架构版本历史（近端）：

- v129 (2026-09-03) ✅ current — runAll 金丝雀平滑重启
- v128 (2026-09-02) archived — 阿里云 GitLab 区域目录先行 + 人工建节点
- v127 (2026-09-02) archived — 部署 9999 源码编译

本次需求在 v129 上设计，**不新增架构版本**（见「架构变更影响」）。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 8 files / 0 nodes（图索引面为 JS/TS/Python，不含 `taskBill` Go 符号）。
- CRG unavailable for Go symbol explore（graph 无 Go 节点）。源码阅读：`regionResourceView`、`ensureTenantGitlabGroupForRegion`、`handleAdminProvisionGitlabResource`、`markOrderPaid` gitlab_disk 分支、`adminGrantResources`、`SystemAdminGitlabTenantPanel.vue`。
- 爆炸半径：所有 GET 配额视图（账单/连接页/管理端查询）都会触发 ensure；管理端赠送另有一条 ensure（保留）。

## 决策锁定（goal-mode 自主）

| # | 决策 | 选择 | 理由 |
|---|------|------|------|
| D1 | 何谓「发货」 | **仅** SystemAdmin 点「开通实施」才创建 GitLab 组并置 `active` | 与现有按钮/API 对齐，不另开发货状态机 |
| D2 | 支付写不写 `disk_gb` | **写**：支付入账额度；GitLab 组仍未建 | 管理员能看见买了多少 GB；租户能看见「已购待开通」 |
| D3 | GET 配额 | `pending_admin` **禁止** `ensureTenantGitlabGroupForRegion` | 这是当前自动发货根因 |
| D4 | 任务帖 / 流量 | 仍自动发放 | 用户只改磁盘购买 |
| D5 | 管理端赠送 | **仍可** ensure 组（管理员就是发货人） | 赠送路径管理员已在场，不必再点第二次开通 |
| D6 | 存量 `active` | 不回滚 | 已自动建组的租户保持可用 |
| D7 | 到期日 | 本增量仍在支付时写 `disk_expires_at` | 改到「开通日起算」另记 OPT，避免和发货门闩搅在一起 |
| D8 | 待开通列表 | 新增 system-admin GET 列表 | 只靠手填租户 ID 无法运营发货 |
| D9 | 事件 | 支付发 `GitlabDiskFulfillmentQueued`；开通发 `GitlabTenantResourceProvisioned` | 跨边界建组必须有 MQ；key 分别为 `order_id`、`tenant_id:region` |
| D10 | API 落点 | 扩展 **taskBill**（Go） | 无新服务；禁止 Django |

### 拒绝的方案

- **只改文案、不关 GET ensure**：组仍会被第一次打开页面的人建出来。
- **支付不写 disk_gb**：开通 API 用 `res.DiskGB` 设限额，会变成 0；管理端看不到购额。
- **支付后同步调 GitLab 再失败才 pending**：这就是要废除的 hybrid 自动发货。
- **新建发货微服务**：配额与开通已在 taskBill。

## 方案

### 1. 支付（`markOrderPaid`）

磁盘行保持：

- `disk_gb += quantity`
- `provisioning_status='pending_admin'`（INSERT；ON DUPLICATE 若已是 `active` 则**保持 active**，只加额度——续费/加购已开通租户不应被打回待开通）
- **不**调用 `ensureTenantGitlabGroupForRegion`

续费决策（锁定）：已 `active` 的租户再买磁盘 → 仍加 `disk_gb`，状态保持 `active`；管理员无需再点开通。ensure 组/限额由开通路径或赠送路径负责；**active 续费**允许在支付成功后 **只更新 GitLab `repository_size_limit`**（已有组），不新建组。若更新失败只 warn，不改 `pending_admin`。

首次购买（行不存在或原 `not_purchased`/`pending_admin`）→ 必须人工开通。

支付成功后对每个 `gitlab_disk` 行发布 `GitlabDiskFulfillmentQueued`（Kafka `gitlab-disk-fulfillment-queued`，key=`order_id`）。`pending_node` 区继续额外发既有 `GitlabManualNodeFulfillmentQueued`。

任务帖、流量发放逻辑不变。

### 2. GET `regionResourceView` / `listGitlabResourceViews`

ensure 条件改为：

```
DiskUsedBytes <= 0
&& status != not_purchased
&& status != pending_admin    // 新增
&& !pending_node
```

即：**只有 `active`（以及历史空 status 被 COALESCE 成 active 的存量）才允许 GET 侧 ensure。** 为避免 COALESCE 把 pending 吃掉，读库已用 `COALESCE(provisioning_status, 'active')`——`pending_admin` 必须显式写入（支付路径已写）。

GET 仍返回 `provisioning_status`、`disk_gb`，供连接页展示「等待管理员开通实施」。

### 3. 管理员开通（既有 POST）

`POST /api/tenant/{tenant_id}/billing/gitlab-resources/provision/` 仍是唯一建组入口（购买路径）：

- `pending_node` → 409（既有）
- `ensureTenantGitlabGroupForRegion` 成功 → `provisioning_status=active`
- 发布 `GitlabTenantResourceProvisioned`（topic `gitlab-tenant-resource-provisioned`，key=`{tenant_id}:{region}`）
- 前端既有 clickGuard + Idempotency-Key 保持

无消费者自动化开通（禁止 timer/worker 扫表去建组）。

### 4. 待开通队列（新 Go 接口）

`GET /api/system-admin/gitlab-resources/pending-fulfillment/?limit=`

- 鉴权：system-admin（与区域管理相同）
- 返回 `provisioning_status=pending_admin` 且 `disk_gb>0` 的行：`tenant_id, region, region_name, disk_gb, updated_at, buyer_note`（note 取该租户该区最近一笔含 `gitlab_disk` 的已支付订单）
- 分页：默认 50，最大 200
- Swagger：实现阶段写入 `taskBill/src/openapi.yaml`

`SystemAdminGitlabTenantPanel`：在查询表单上方列出待开通队列；点一行填入租户 ID + slug，再点既有「开通实施」。

### 5. 管理端赠送

保持 `adminGrantResources` 成功后 ensure 组。赠送不是「用户下单自动发货」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 用户支付含 GitLab 磁盘 | GitlabDiskFulfillmentQueued | Kafka `gitlab-disk-fulfillment-queued` | `markOrderPaid` 提交后 | 审计/告警；运维以 pending_admin 列表为 SSOT | — |
| 管理员开通实施成功 | GitlabTenantResourceProvisioned | Kafka `gitlab-tenant-resource-provisioned` | `handleAdminProvisionGitlabResource` | 审计；无自动消费者 | — |
| 支付含 pending_node 区 | GitlabManualNodeFulfillmentQueued | 既有 | 既有 | 既有 | 不变 |
| GET 配额 / 待开通列表 | — | — | — | — | 纯查询 |
| 任务帖/流量自动发放 | BillingTransactionCreated | 既有 | `markOrderPaid` | 既有 | 非磁盘 |

## 价值流影响（输入给 Step 4）

影响既有流：

- 「租户购买 GitLab 磁盘/流量」（`tenant_gitlab_resource_purchase`）：支付后磁盘变为**待开通额度**，不再隐含建组。
- 「可插拔多区域 gitService」：废除购买路径 hybrid「GET 即 ensure」。
- 「阿里云人工建节点」：叠加关系不变（节点未就绪仍 409）。

新字段：无新表。新查询：`billing_tenant_gitlab_resource.provisioning_status=pending_admin`。

测试文件预期：`gitlab_region.go` 视图测、`order_payment` 事件测、`SystemAdminGitlabTenantPanel` 队列测。

## 领域概念清单（输入给 Step 6）

- 限界上下文：计费 / GitLab 区域履约（taskBill）
- 实体：`TenantGitlabResource`（额度 + 开通状态）、`ResourceOrder`
- 聚合根：`TenantGitlabResource`（tenant_id+region）
- 值对象：`ProvisioningStatus` = `not_purchased` \| `pending_admin` \| `active`
- 不变量：`pending_admin` ⇒ 禁止 GET/支付路径调用 GitLab Admin API；`active` ⇒ 组应存在；`pending_node` ⇒ 任何路径禁止 Admin API

## 角色与权限（摘要）

完整表见 `2026-09-03-gitlab-disk-manual-admin-fulfillment-permission-analysis.md`。

- 租户：可买、可看待开通，不能 POST provision
- 系统管理员：待开通列表 + 开通实施
- 不新增角色

## 🏛️ 架构变更影响

- **迭代版本**: 不升 v130
- **理由**: 无新增/移除 Application_Component、无新数据所有权、无新基础设施。taskBill→Kafka 与 taskBill→GitLab Admin API 的 Rel_Flow 已存在；本增量是**关闭 GET 上的隐式调用**，把建组收敛到既有 provision 接口。current 视图 v129 主题是金丝雀重启，复制整图只为改一条副作用会造成噪声。
- **python_api_approval**: not_applicable
- **若后续**把「待开通队列」做成独立任务系统，再补 architecture。

## 验收（设计级）

1. 支付磁盘后 `pending_admin` + `disk_gb` 增加；GitLab 上无新 `tenant-*` 组。
2. 租户打开连接页/账单 GitLab 卡多次，仍 `pending_admin`，无 ensure 调用。
3. 管理员点开通实施后组存在且状态 `active`，并发出 `GitlabTenantResourceProvisioned`。
4. 管理端可见 pending 列表，无需事先知道租户 ID。
5. 任务帖/流量支付仍立即可用。
6. 已 `active` 租户加购只加额度，不被打回 pending。
7. `pending_node` 开通仍 409。
