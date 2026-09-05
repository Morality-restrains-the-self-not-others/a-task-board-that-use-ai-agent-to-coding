# 购买页可选阿里云 GitLab 区域：目录先行 + 人工建节点开通

- **日期**: 2026-09-02
- **状态**: accepted（/goal 覆盖 USER GATE，自动采用）
- **迭代**: aliyun-gitlab-region-manual-node
- **作者**: cursor
- **意图**: `docs/intents/backend/aliyun_gitlab_region_manual_node.intent.md`
- **ADR**: [ADR-0057](../../adr/0057-aliyun-gitlab-region-pending-node.md)（扩展 ADR-0014，不取代）
- **python_api_approval**: not_applicable（无新增 Python HTTP 接口；扩展 taskBill + taskFE）

## 背景

租户 VIP1 购买页 `order-gitlab-resources` 的区域下拉目前只有已部署的腾讯云 GitLab 实例（`tencent-shanghai-5`、`tencent-sh-1`）。用户要求：**选择区域时可点选阿里云上的区域**；平台随后**人工创建服务节点、挂载磁盘，再帮对方开通**。

不自动调阿里云 ECS API 拉起 GitLab（人工履约）。不改 VIP1 / 1GB 限购 / 支付发放。

## 对当前架构的理解

根据 `docs/architecture/` current（v127）：

- 共有 2 个 **current** 视图：`enterprise-landscape`、`application-integration`（**v127** ✅）。
- 业务层：计费购买、GitLab 区域选购（ADR-0014）、部署 9999 源码编译。
- 应用层：`taskFE` OrderCreate；`taskBill` `billing_gitlab_region` + hybrid 开通（Admin API 失败 → `pending_admin`）。
- 技术层：腾讯云 GitLab CE 两套；阿里云仅用于任务容器 ECS，**尚无**阿里云 GitLab 节点。
- 上次交付版本 **v127**（2026-09-02 13:45）。

📋 架构版本历史（近端）：

- v127 (2026-09-02) ✅ current — 部署 9999 编排源码编译
- v126 (2026-09-01) archived — 项目 L2 种下自动运行评论

本次需求在 v127 上设计 **v128 target**。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 3 files / 0 nodes（本会话开始时无本主题脏文件）。
- CRG unavailable for MCP query（无 codegraph MCP）；符号理解走源码：`GitlabRegion.CloudProvider`、`listGitlabRegions`、`ensureTenantGitlabGroupForRegion`、`OrderCreate.vue` 区域 `<select>`、`markOrderPaid` 磁盘行写 `pending_admin`。

## 决策锁定（goal-mode 自主）

| # | 决策 | 选择 | 理由 |
|---|------|------|------|
| D1 | 产品模型 | **共享区域 GitLab CE**（ADR-0014），不是每租户一台 GitLab | 每租户独立 CE 会复制 OIDC/PAT/备份；「为之建节点」指**该阿里云区域的第一套共享实例** |
| D2 | 何时可售 | **目录先行**：节点未部署也可出现在购买下拉 | 用户现在就能选阿里云区域并下单 |
| D3 | 实例创建 | **人工**：运维在阿里云建机、挂盘、部署 gitService，再填 API/token | 用户明确要求手动 |
| D4 | 开通 | `infra_status=pending_node` 时**禁止** Admin API；支付后保持 `pending_admin`；节点就绪后再走既有 admin provision | 避免打到空 URL / 本机 8012 |
| D5 | 区域集合 | seed 常用阿里云地域（杭州/上海/青岛/北京/张家口/深圳/广州/成都/香港/新加坡）；其余走 SystemAdmin 新增 | 下拉可用，不堆全球全量 |
| D6 | slug | `aliyun-{aliyun-region-id}`，如 `aliyun-cn-hangzhou` | 与官方 region id 对齐，可插拔 |
| D7 | OIDC / conf | **节点就绪后再**加 `conf/infra/git-service-<slug>/` 与 OIDC client | 避免空配方与 unauthorized_client |
| D8 | 事件 | 支付命中 `pending_node` 发 `GitlabManualNodeFulfillmentQueued`（Kafka key=`order_id`） | 跨边界副作用；运维队列以库状态为 SSOT，事件做审计/告警 |

### 拒绝的方案

- **仅改文案、不 seed 区域**：用户选不到阿里云。
- **下单即调阿里云 RunInstances**：与「手动创建」冲突，且资金路径自动开云主机风险高。
- **每租户独立 GitLab**：超出购买页范围，破坏 ADR-0014 共享实例模型。

## 方案

### 1. 区域目录

`billing_gitlab_region` 新增 `infra_status`：

| 值 | 含义 | 购买 | Admin API |
|----|------|------|-----------|
| `ready` | 实例已部署（存量腾讯云默认） | 可售 | hybrid |
| `pending_node` | 可售、节点未建 | 可售 | **禁止** |

Seed 10 条 `cloud_provider=aliyun`、`is_active=1`、`access_mode=release`、`infra_status=pending_node`、token/API 空、`total_disk_gb=0`（容量检查跳过）。

展示名示例：`阿里云杭州（华东1）`。

### 2. 购买页（OrderCreate）

- `<select data-testid="order-gitlab-region">` 用 `<optgroup>`：腾讯云 / 阿里云（人工开通节点）/ 其他。
- 选中 `pending_node`：提示「该区域 GitLab 节点将由平台在阿里云创建并挂载后开通」。
- 施工留言：磁盘 **或** `pending_node` 区域即显示。
- POST items.region 仍为 slug（不变）。

### 3. 支付与开通

- 磁盘行仍 `pending_admin`（既有）。
- `regionResourceView` / `handleAdminProvisionGitlabResource`：`pending_node` 或缺少 API/token → **不**调用 `ensureTenantGitlabGroupForRegion`。
- 运维：SystemAdmin 区域卡徽章「待创建节点」；填 API+token 后 PUT `infra_status=ready`；再对租户点既有开通。

### 4. 运维清单（节点就绪）

1. 在所选阿里云地域创建 ECS，挂载数据盘到 `GITLAB_HOME`。
2. 按 ADR-0014 增加 `conf/infra/git-service-<slug>/config.yaml` + 边缘 vhost。
3. 部署 GitLab CE；SystemAdmin 填 `gitlab_api_base` / `gitlab_web_url` / PAT。
4. PUT `infra_status=ready`；seed taskAuth OIDC client（`gitlab-git-service-<slug>`）。
5. 对排队租户调用既有 provision API。

本增量**不**自动执行 1–2、4 的 OIDC seed。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 租户支付含 pending_node 区的 GitLab 行 | GitlabManualNodeFulfillmentQueued | Kafka `gitlab-manual-node-fulfillment-queued` | `markOrderPaid` 成功后 | 审计/告警；运维以 `pending_admin`∩`pending_node` 查询为 SSOT | — |
| 运维标记节点已部署 | GitlabRegionInfraMarkedReady | Kafka `gitlab-region-infra-marked-ready` | SystemAdmin PUT infra_status ready | 审计 | — |
| 租户创建 pending 订单 | — | — | `resource_order_created` 日志 | — | 下单不发放配额（既有） |
| GET 区域列表 | — | — | — | — | 纯查询 |

## 价值流影响（输入给 Step 4）

扩展既有「租户购买 GitLab 须按行指定区域」流：下拉增加阿里云组；支付后人工建节点。不新开独立计费流。

字段：`taskBill.billing_gitlab_region.infra_status`、`taskBill.billing_gitlab_region.cloud_provider`、`taskBill.billing_tenant_gitlab_resource.provisioning_status`。

## 领域概念清单（输入给 Step 6）

- 限界上下文：计费 / GitLab 区域目录（taskBill）
- 实体：`GitlabRegion`（+ `infra_status`）
- 聚合：`GitlabRegion`（目录）；`TenantGitlabResource`（配额+开通状态）
- 值对象：`InfraStatus` = ready | pending_node；`CloudProvider` = tencent | aliyun | …

## 🏛️ 架构变更影响

- **迭代版本**: v128 🎯 target
- **迭代名称**: aliyun-gitlab-region-manual-node
- **作者**: cursor
- **设计日期**: 2026-09-02 14:10
- **新增文件**（每个视图四类伴生格式）:
  - `docs/architecture/v128-enterprise-landscape-20260902-1410-cursor.puml`
  - `docs/architecture/v128-application-integration-20260902-1410-cursor.puml`
  - 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**: 🟢 阿里云 GitLab 区域目录（pending_node）+ 人工建节点履约过程；🟡 taskBill/taskFE 购买下拉与开通门闩；🔴 无废弃组件
- **python_api_approval**: not_applicable
