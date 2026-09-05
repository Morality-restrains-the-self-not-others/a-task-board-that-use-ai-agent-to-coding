# 多区域内网镜像仓库副本 — 镜像提交与同区优先拉取

- **日期**: 2026-08-29
- **作者**: cursor
- **状态**: approved（2026-08-29，总体设计审批）
- **迭代**: multi-region-image-registry-replicas
- **架构**: **v119 🎯 target**（基于 v118 current）
- **ADR**: [ADR-0050](../../adr/0050-multi-region-image-registry-replicas.md) accepted
- **意图**: `docs/intents/backend/multi_region_image_registry_replicas.intent.md`、`docs/intents/frontend/provider/multi_region_image_registry_replicas.intent.md`
- **已拍板**:
  - 副本产生：厂商在各区域仓库推好（或开 ACR/跨地域复制），提交时按区域登记**公网 URL**；平台从已知 ACR/TCR/SWR 规则**推导内网地址**
  - 启机缺失副本：同区内网 → 同区公网 → 回退规范 `image_url`（兼容存量单地址镜像）

## 1. 背景与现状

评论启动云服务器时，UserData 用单一 `container_image_url` 做 `docker pull`。该 URL 来自：

1. 厂商提交镜像版本时的 `ai_provider_vendorcontainerimage.image_url`（一条公网引用）
2. 租户安装时拷贝到 `cloud_tenant_installed_images.image_url`
3. `enrichStartVmPayloadFromInstalledImage` **在解析地域之前**拼 `image_url:version`，之后不再按 `region_id` 改写

同一逻辑容器镜像已经支持**多区域服务器镜像**（CSI）：`ai_provider_containercloudserverassociation` 按 `(platform_type, region)` 绑定 `ai_provider_vendorcloudserverimage`。`resolveCloudServerImageID` 会按启机地域选出 CSI，但 **docker 仓库地址不随地域变化**。

后果：杭州 ECS 仍可能去青岛公网 ACR 拉层，跨区流量贵、慢，且无法使用同区 VPC 内网仓库。

平台控制面**走不进**云厂商 VPC。现有 `shareLib/registryhost` 对 `registry-vpc.*` / RFC1918 **直接拒绝**（厂商解析架构、抽取 skill）。因此平台不能把镜像「代推」进内网仓库。

## 2. 目标

1. **一个逻辑镜像版本** = 规范公网 `image_url` + 零或多条**区域仓库副本**（同一 digest 的多 registry 宿主）。
2. **提交方案**：厂商按已绑定 CSI 的 `(platform, region)` 登记该区**公网**仓库引用；服务端推导内网宿主，落库。
3. **启机**：评论在区域 R 跑服务器时，UserData 的 `container_image_url` **优先同区内网**，否则同区公网，再否则规范地址。
4. **不破坏**：无副本的存量镜像、本机 relay `docker pull`、控制面 OCI 抽取/架构解析，仍只用规范公网 URL。

## 3. 当前架构理解（设计基线）

根据 current 架构稿与运行事实：

- 共有 2 个 current 视图：`enterprise-landscape` / `application-integration`（v118 Git OAuth 资源标记；全站拓扑以 `db/registry.yaml` + table-to-owner 为准）
- 应用层关键组件：`taskAiProvider`（镜像目录/CSI 关联）、`taskCloudService`（已安装镜像、start-vm、UserData 替换）、`taskEvents`（`CLOUD_SERVER_STARTED` → 云厂商 RunInstances）、`taskFE`（评论启机不传 URL）、厂商门户 Vue
- 数据：`ai_provider_vendorcontainerimage.image_url` 单列；`ai_provider_containercloudserverassociation` 已按区域绑 CSI；`cloud_tenant_installed_images` 安装时快照 `image_url`
- 技术层：云厂商区域 Registry（ACR/TCR/SWR）在 VPC 内；平台节点只达公网
- 上次架构版本 **v118 ✅ current**

📋 架构版本历史（最近）：

- v118 (2026-08-29 18:32) ✅ current — Git OAuth 资源使用标记
- v117 — 开放式邀请链接
- v116 — 邮件邀请退订

本次迭代将在 v118 基础上设计。

## 4. 设计

### 4.1 领域模型

| 概念 | 说明 |
|------|------|
| **逻辑镜像版本** | `ai_provider_vendorcontainerimage` 一行；规范 `image_url` 供控制面拉 manifest / 抽 skill |
| **区域运行环境** | 已有 association：`(container_image, platform, region) → CSI` |
| **区域仓库副本** | 同一逻辑版本在某 `(platform, region)` 的公网引用 + 推导出的内网引用 + 可选 digest |
| **安装快照** | 租户安装时拷贝副本表到 `registry_replicas_json`，启机不直连 ai-provider 副本表（单表所有权） |

键与 CSI **对齐**：`(container_image_id, platform_type, region)`。无 CSI 的区域不必登记副本（该区本就不能启机）。有 CSI 无副本 → 走规范 URL 回退。

### 4.2 数据

**新表** `ai_provider_containerimage_registry_replica`（owner：`taskAiProvider`，`dataMigrate/taskAiProvider/`）

| 列 | 说明 |
|----|------|
| `id` | BIGINT PK AUTO_INCREMENT（与现网 marketplace 表一致；行数随镜像×区域，非时间累积） |
| `container_image_id` | 逻辑版本 |
| `platform_type` | 与 association 相同（`aliyun` / `tencentcloud` / …） |
| `region` | 云区域 ID（如 `cn-hangzhou`） |
| `public_url` | 厂商登记的公网引用（host/repo:tag 或 @digest） |
| `intranet_url` | 服务端推导；未知规则时为空 |
| `digest` | 公网 manifest digest（best-effort；拉不到则空） |
| UNIQUE `(container_image_id, platform_type, region)` | 每区至多一条 |

字符集 `utf8mb4` / `utf8mb4_unicode_ci`。无分区（小表）。

**已安装镜像** `cloud_tenant_installed_images` 增列：

- `registry_replicas_json` TEXT NOT NULL DEFAULT `'[]'`
  - 元素：`{platform_type, region, public_url, intranet_url, digest}`
  - 安装时从 catalog 拷贝；后续目录变更不自动回写已装行（与 skill/auto_run 快照同语义）

规范 `image_url` 列保留，不作破坏性迁移。

### 4.3 提交方案（厂商门户）

现有两步不变：

1. **添加/编辑版本**：规范公网 `image_url`（必须平台可达；内网地址仍 400，见既有 private_registry）
2. **区域运行环境**：每区选 CSI（已有）

新增：在同一「区域运行环境」保存路径上增加可选字段 `registry_public_url`。

- 空：该区无副本，启机回退规范 URL
- 非空：必须通过 `RejectPrivateRegistry`（禁止把 VPC 地址当「公网」提交）
- 服务端 `SuggestIntranetRegistryRef(public_url)` 写 `intranet_url`
- 可选 HEAD/GET 公网 manifest 记 `digest`（失败不阻断保存）
- Upsert 副本行，与 association 同键；清 CSI（csiID=0）时**同时删**该区副本

公开目录 / 安装 payload：`runtime_environments[]` 每条附：

```json
"registry_replica": {
  "public_url": "registry.cn-hangzhou.aliyuncs.com/ns/img:tag",
  "intranet_url": "registry-vpc.cn-hangzhou.aliyuncs.com/ns/img:tag",
  "digest": "sha256:…"
}
```

无副本时省略或 `null`。

**内网推导（SSOT：`shareLib/registryhost`）** — 只改 host，保留 path/tag：

| 公网 host | 内网 host |
|-----------|-----------|
| `registry.{region}.aliyuncs.com` | `registry-vpc.{region}.aliyuncs.com` |
| `swr.{region}.myhuaweicloud.com` | `swr.{region}.inner.myhuaweicloud.com` |
| TCR 等有稳定规则的再补进 `registry_cases.json` | 无规则 → `intranet_url=""`，启机用该区 `public_url` |

未知云不发明映射。推导结果只读展示给厂商，不接受手填内网 URL（避免控制面误把 VPC 当地址去探测）。

### 4.4 启机拉取（评论运行服务器）

改 `start-vm` / `start-vm-auto` 顺序：

1. 校验 payload、鉴权、CSI/`resolveCloudServerImageID`（得到 `resolvedRegion`）
2. **再** `selectContainerImagePullRef(installed, platform, resolvedRegion)` 写入 `body["container_image_url"]`
3. `CLOUD_SERVER_STARTED` / UserData 替换使用该值

选择函数：

```
if replica[platform, region].intranet_url 非空 → 用之
else if replica[platform, region].public_url 非空 → 用之
else → mergeInstalledImageRef(canonical image_url + version)  // 存量行为
```

副本 JSON 优先用安装快照；快照为空数组时行为与今日完全一致。

**本机 relay** `GET /api/internal/image/resolve`：**不**改，仍规范公网 URL（无区域、控制面不能拉 VPC）。

前端评论启机仍**不传** `container_image_url`（现有 Playwright 契约保持）。

### 4.5 接口落点（全部 Go）

| 方法 | 路径 | 服务 | 说明 |
|------|------|------|------|
| POST（扩展） | `/api/vendor/container-images/{id}/cloud-server-image-association/` | taskAiProvider | body 增 `registry_public_url`；响应 association 带推导后的 replica |
| GET | 同上 associations 列表 | taskAiProvider | 每项带 `registry_replica` |
| GET | 公开 catalog / runtime environments | taskAiProvider | 每环境带 replica |
| POST install | 已有 installed-images | taskCloudService | 快照 `registry_replicas_json` |
| POST start-vm* | 已有 | taskCloudService | 按区选 URL |

不新增 Python/Django 接口。🐍 门禁 **not_applicable**。

### 4.6 幂等与审核

- 副本 Upsert：同键重复提交覆盖，幂等键 = `container_image_id + platform_type + region`
- **不**在 submit 审核时强制每区有副本（已拍板回退）
- 审核通过/激活逻辑不变

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 厂商登记/更新某区仓库副本 | `ContainerImageReplicasUpdated` | LogEventBus（与现网 ImageGroup* 同构） | taskAiProvider association/replica upsert 成功后 | 审计/日后投影已装快照 | 证据豁免：taskAiProvider 无 Kafka；登记 `_meta/publish_evidence_exempt.yaml` |
| 厂商清除某区 CSI+副本 | `ContainerImageReplicasUpdated`（payload `cleared: true`） | 同上 | 同上 | 同上 | 同上 |
| 评论启机选用拉取地址 | —（写入既有 `CLOUD_SERVER_STARTED.data.container_image_url`） | 既有云启机契约 | taskCloudService finalizeStartVm | taskEvents RunInstances / UserData | 无新聚合事实；URL 已在现有事件 |

启机选择本身不另发事件。`container_image_url` 语义从「永远规范地址」变为「本机次选用的副本或回退」。

## 6. Domain Concept Inventory（供 /6-ddd）

- **Bounded Contexts**: 镜像市场（taskAiProvider）、云资源启机（taskCloudService）
- **Key Entities**: ContainerImageVersion、RegistryReplica、CloudServerImage、TenantInstalledImage
- **Candidate Aggregates**: ContainerImageVersion（含 association + replica 子实体）；TenantInstalledImage（安装快照）
- **Domain Events**: `ContainerImageReplicasUpdated`；启机沿用 `CLOUD_SERVER_STARTED`

## 7. 价值流影响

现有流：`container-image-creator`、`cloud-server-image`、`vendor-register-cloud-server-image`（planned）、评论 start-vm。

- 在「镜像市场」域增加 step `vendor-declare-registry-replicas`（厂商按区登记公网仓库）
- 修改 start-vm 相关 step：`container_image_url` 按区选择
- 字段：
  - `ai-provider.ai_provider_containerimage_registry_replica.public_url`
  - `ai-provider.ai_provider_containerimage_registry_replica.intranet_url`
  - `task-cloud-service.cloud_tenant_installed_images.registry_replicas_json`（JSON 内嵌 `public_url`/`intranet_url` 在 description 说明，不拆四段）
- 切片与 YAML 由 `/4-value-stream` 落地

Looking at existing value streams, this change affects 镜像市场（厂商提交/区域运行环境）and 云平台启机（评论 start-vm UserData pull）。Full mapping in step 4.

## 8. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：114 nodes / 1012 edges / 18 files；last_updated 2026-08-29T13:01:08；branch main。索引面远小于 monorepo，主要为 JS/TS/Python/bash 子集 |
| 关键发现 | 图未覆盖 `taskAiProvider`/`taskCloudService` Go 启机与 association 主路径；爆炸半径以源码检索为准：`enrichStartVmPayloadFromInstalledImage`、`resolveCloudServerImageID`、`userdata_replace`、`vendor_container_actions` association upsert、`shareLib/registryhost` |
| 决策影响 | 启机必须在 **CSI 地域解析之后** 才选定 pull URL；association upsert 是副本写入的自然缝 |
| skip 理由 | 图对本次 Go 热路径 `unavailable`（索引文件集不含相关 .go） |

## 9. 备选（已拒绝）

| 方案 | 拒绝原因 |
|------|----------|
| 平台自建多区域 Harbor 代复制 | 控制面进不了 VPC；运维面大；用户已否 |
| 厂商手填内网 URL | 平台无法校验；易把 VPC 地址填进控制面探测路径 |
| 有 CSI 无副本则禁止启机 | 用户选择兼容存量回退 |
| 提交审核强制每区副本 | 同上；抬高上架门槛 |

## 10. NFR 摘要（完整表在 /5-nfr）

- **路径分片**: start-vm 已带 tenant + region；副本表按镜像 ID 不按租户
- **幂等**: 区域副本 Upsert；启机选用纯函数，无额外写（除既有 binding）
- **资金/云资源**: 启机路径保持既有 L3；本增量只改 pull 主机名

## 11. 🏛️ 架构变更影响

- **迭代版本**: v119 🎯 target
- **迭代名称**: multi-region-image-registry-replicas
- **作者**: cursor
- **设计日期**: 2026-08-29 21:24
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v119-enterprise-landscape-20260829-2124-cursor.puml`
  - 🆕 `docs/architecture/v119-application-integration-20260829-2124-cursor.puml`
  - 🆕 `docs/architecture/v119-enterprise-landscape-20260829-2124-cursor.diff.archimate`（增量变迁：v118→v119 变更元素 + Plateau/Gap/WP）
  - 🆕 `docs/architecture/v119-application-integration-20260829-2124-cursor.diff.archimate`（增量变迁视图）
  - 🆕 `docs/architecture/v119-enterprise-landscape-20260829-2124-cursor.full.archimate`（全量拓扑：变迁后完整架构）
  - 🆕 `docs/architecture/v119-application-integration-20260829-2124-cursor.full.archimate`（全量拓扑视图）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v118-*-20260829-1832-cursor.puml` (current)
- **变更明细**: 🟢 副本表 + ContainerImageReplicasUpdated / 🟡 taskAiProvider catalog、taskCloudService 安装快照与 start-vm 选址、registryhost 公网→内网 / 🔴 无

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量模型 — Plateau v118 + Plateau v119 + Gap「单 URL 跨区拉镜像」+ WP-multi-region-image-replicas + 本次 🟢/🟡 元素；视图含 `sourceConnection` |
| **`.full.archimate`** | 全量模型 — 变迁后本迭代拓扑（门户/AiProvider/Cloud/副本表/启机事件）；可导入 Archi |

## 12. 验收（实现阶段）

1. 厂商在杭州运行环境填 `registry.cn-hangzhou.aliyuncs.com/ns/img:tag` → 存 `intranet_url=registry-vpc.cn-hangzhou.aliyuncs.com/ns/img:tag`
2. 填 `registry-vpc.cn-hangzhou...` → 400 无法触及（与今日一致）
3. 评论 `region_id=cn-hangzhou` start-vm → UserData/`CLOUD_SERVER_STARTED` 为 vpc host
4. 无副本的旧镜像 → 仍规范 `image_url:version`
5. relay resolve-image 仍规范公网 URL
6. 单测：推导表、选址优先级、安装快照、association 清 CSI 级联删副本

## 13. 实施计划（批准后）

1. `shareLib/registryhost` 增加公网→内网 + 用例表
2. dataMigrate 新表 + installed 列
3. taskAiProvider upsert/list/catalog
4. taskCloudService 安装快照 + start-vm 选址
5. 厂商门户区域运行环境字段与只读内网展示
6. 事件 + 豁免登记 + 单测
