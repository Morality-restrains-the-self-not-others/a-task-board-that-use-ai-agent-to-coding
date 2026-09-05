# 设计文档：Cloud 域 Phase 3g+3h — CPA 与 Aliyun SDK 查询 API 迁入 Go

**日期：** 2026-07-07  
**状态：** 已设计（v10 target 架构待交付）  
**前置决策：** Phase 3g（CPA + ActiveMethod → taskCloudService）已确认；本次扩展 **Phase 3h** 将 Aliyun SDK 查询链全量迁入 Go。  
**动机：** 消除 Cloud 域 split-brain（网关误路由 Go stub 导致 zones 404）；统一 Cloud Platform 聚合真源；将 Aliyun 出站 I/O 从单线程 Django 剥离。

---

## 架构基线确认

| 维度 | 内容 |
|------|------|
| **当前 target** | v9 — CloudServerConfig / instruct_worker 已设计迁 Go |
| **分裂现状** | CPA：Django ORM 真源 + Go SQLite stub；查询 API：Django `cloud/providers/aliyun/`；网关 `cloud-platform/*/cloud/*` 曾误打 Go |
| **应用层** | `taskCloudService (:8018)` 已有 CloudServerConfig SSOT；Aliyun 查询仍经 `saas-backend` |
| **SDK 资产** | monorepo `sdk/ecs-20140526`（Go v7 模块）；Python 参考实现 `cloud/providers/aliyun/` |

📋 **相关版本历史：**

- v9 (2026-07-06) 🎯 target — Cloud 域 Phase 3a–3f；CPA 标注 stub
- **v10 (2026-07-07) 🎯 target — 本设计：CPA + Aliyun 查询 API Go SSOT**

---

## 问题分析

### 触发 incident（zones 404）

```text
GET /api/tenant/{tid}/cloud-platform/{auth}/cloud/zones/?region_id=cn-hongkong
  → APISIX task-cloud-service (:8018)
  → Go handleCloudPlatformDetail（本地 SQLite 无 Django 授权行）
  → 404 {"error":"authorization not found"}
```

**根因：** 网关路由与数据真源不一致；Go 仅有 CPA stub，未实现 zones 语义。

### 为何须将 Aliyun SDK 一并迁入 Go

| 仅迁 CPA（3g） | CPA + SDK 查询（3g+3h） |
|----------------|-------------------------|
| zones/regions 仍 Django 双跳 | 单服务内：读 CPA → 调 Aliyun |
| 网关须维护 Django 子路由 | `cloud-platform/*/cloud/*` 统一 :8018 |
| incident 类路由漂移风险仍在 | Cloud 域查询面收敛 |
| 假死收益有限（低频 CRUD） | **假死收益中等**（实例列表/询价可并发阻塞 Django） |

---

## 范围

### Phase 3g — 授权真源（前置，本迭代一并交付）

| 模块 | Django 现状 | Go 目标 |
|------|------------|---------|
| `CloudPlatformAuthorization` CRUD | ViewSet + PostgreSQL/SQLite | `task_cloud.db.cloud_platform_authorizations` |
| `CloudPlatformAuthorizationActiveMethod` | ORM + toggle-active | 新表 `cloud_platform_authorization_active_methods` |
| `verify-credentials` | STS GetCallerIdentity | `aliyun-sdk-go` 同等校验 |
| `toggle-active` / `active-list` | Django 函数视图 | Go handlers |
| Kafka `CLOUD_PLATFORM_AUTHORIZATION_CREATED` | model.save 发事件 | Go 创建 AK 后发 Kafka |

### Phase 3h — Aliyun SDK 查询 API（本次扩展）

**租户路径**（`/api/tenant/{tid}/cloud-platform/{auth_id}/cloud/...`）：

| # | 方法 | 路径 | Django 实现 | Go handler |
|---|------|------|-------------|------------|
| 1 | GET | `cloud/regions/` | `tenant_cloud_regions` | `handleTenantCloudRegions` |
| 2 | GET | `cloud/zones/` | `tenant_cloud_zones` | `handleTenantCloudZones` |
| 3 | GET | `cloud/bandwidth-limitation/` | `tenant_cloud_bandwidth_limitation` | `handleTenantCloudBandwidth` |
| 4 | GET | `cloud/available-instances/` | `tenant_cloud_available_instances` | `handleTenantCloudAvailableInstances` |
| 5 | GET | `cloud/instance-price/` | `tenant_cloud_instance_price` | `handleTenantCloudInstancePrice` |
| 6 | GET | `cloud/instance-details/` | `tenant_cloud_instance_details` | `handleTenantCloudInstanceDetails` |
| 7 | GET | `cloud/debug/describe-image-support-instance-types/` | debug 端点 | `handleTenantCloudDebugImageSupport` |

**全局路径**（设置页 `useSetDefaultConfigForm` 等）：

| # | 方法 | 路径 | 说明 |
|---|------|------|------|
| 8 | GET | `/api/cloud/regions/?platform_type=` | 从 Referer/用户解析 tenant，按激活授权查地域 |

### 明确不纳入本迭代（Phase 3i+）

| 模块 | 理由 |
|------|------|
| `start_vm` / `stop_vm` / RunInstances | 已有独立 compute 路径；LCM 复杂度高，单独 WP |
| VPC/VSwitch/SecurityGroup CRUD | 网络编排，依赖多表 |
| `oauth-tokens` / Aliyun OAuth 回调 | OAuth 状态机仍在 Django |
| `/api/cloud/images/`、`server-images/` | 镜像列表混合 TenantInstalledImage，非纯 Aliyun |
| `cloud/regions/`（租户级 `/api/tenant/*/cloud/regions/`） | 旧路由，前端已用 cloud-platform 路径 |

---

## 技术方案

### 1. Go 包结构

```text
taskCloudService/src/
  aliyun/
    client.go          # 从 CPA 记录构建 ECS Client（region 可覆盖）
    regions.go         # DescribeRegions
    zones.go           # DescribeZones
    bandwidth.go       # DescribeBandwidthLimitation（对齐 Python）
    instances.go       # available-instances + instance-details
    price.go           # DescribePrice
    image_support.go   # DescribeImageSupportInstanceTypes
    credentials.go     # STS GetCallerIdentity（verify-credentials）
  provider/
    provider.go        # CloudProvider interface
    aliyun.go
    mock.go            # USE_IN_MEMORY_CLOUD 对齐
  handlers/
    cloud_auth.go      # Phase 3g（扩展现有 cloud_handlers.go）
    tenant_cloud_query.go  # Phase 3h
```

### 2. SDK 依赖

```go
// go.mod
require github.com/alibabacloud-go/ecs-20140526/v7 v7.x.x
replace github.com/alibabacloud-go/ecs-20140526/v7 => ../../sdk/ecs-20140526
```

实现时**逐函数对照** Python 参考（`.ai/03_technical_implementation/07_aliyun_sdk_usage.md`）：

- `cloud/providers/aliyun/region.py` → `aliyun/regions.go`, `zones.go`
- `cloud/providers/aliyun/instance/price.py` → `aliyun/price.go`
- `cloud/providers/aliyun/instance/get_available_instances.py` → `aliyun/instances.go`
- `cloud/providers/aliyun/network/bandwidth.py` → `aliyun/bandwidth.go`

### 3. 响应契约（与前端/E2E 对齐）

| API | 响应形状 | 备注 |
|-----|----------|------|
| regions | `[{id, name}, ...]` | `format_regions_id_name_list` 等价 |
| zones | `[{zone_id, local_name, ...}]` | 与 `ServerConfigHardwarePanel.vue` 解析一致 |
| instance-price | `{price, currency, ...}` | 保留 `X-Cloud-Request-Id` 头 |
| available-instances | 实例类型数组 + diskSupport | Mock 分支保留 |
| instance-details | DescribeInstanceTypes 映射字段 | cpu_cores, memory_gb, gpu_* |
| bandwidth-limitation | `{max_bandwidth, ...}` | platform_type=aliyun 校验 |
| 错误 | `{"status":"error","message":"..."}` | 403/404/400 与 Django 对齐 |

### 4. 权限与租户校验

```text
forward-auth (APISIX) → X-User-Id
Go middleware:
  1. ensureTenantMember(tenantID, userID) — HTTP 调 taskAuth/saas-backend internal（与 Django ensure_tenant_member_response 等价）
  2. resolveCloudAuth(tenantID, authorizationID) — 读本地 CPA 表
  3. _validate_platform_type(query, auth) — platform_type 与授权一致
```

### 5. Mock 模式

环境变量对齐 Django：

- `USE_IN_MEMORY_CLOUD=false` → 真实 Aliyun SDK
- `USE_IN_MEMORY_SERVICES=true` 且 `USE_IN_MEMORY_CLOUD` 非 false → `mock.go` 固定数据（Playwright/E2E）

### 6. 网关路由（目标态）

```yaml
# 从 django-cloud-platform-tenant-apis 移除，并入 task-cloud-service
- /api/tenant/*/cloud-platform/*/cloud
- /api/tenant/*/cloud-platform/*/cloud/
- /api/tenant/*/cloud-platform/*/cloud/*

# Phase 3g：授权 CRUD 也从 django-cloud-tenant-config 迁至 task-cloud-service
- /api/tenant/*/cloud/cloud-platform-authorizations/*
- /api/tenant/*/cloud/toggle-active*
- /api/tenant/*/cloud/active-list*

# 全局 regions（可选同迭代）
- /api/cloud/regions*
```

**优先级：** task-cloud-service `860` 保持不变；删除 `django-cloud-platform-tenant-apis (867)` 避免冲突。

### 7. 切流与删除（对齐 CloudServerConfig 0049 模式）

```bash
# 1. 数据
python3 scripts/migrate_django_to_go.py --service cloud  # CPA + ActiveMethod

# 2. 验证
curl :8018/api/tenant/.../cloud-platform/.../cloud/zones/?region_id=cn-hongkong

# 3. 网关切流 + apisix reload

# 4. Django migration DROP cloud_cloudplatformauthorization* 表

# 5. 删除 cloud/views/tenant_cloud_platform_views_part*.py 公网视图
#    保留 internal read stub 直至所有调用方改 HTTP client（若有）
```

**失败策略：** taskCloudService 不可达 → 502；**禁止**静默回退 Django ORM。

---

## 价值流影响（`cloud-integration`）

| Step | 变更 |
|------|------|
| `platform-authorization` | 执行体 Django → taskCloudService |
| `platform-verify-credentials` | 同上 |
| `cloud-compute` | authorization 解析本地化 |
| **新增** `cloud-platform-query-regions-zones` | regions/zones/price/instances E2E |
| `mock-authorization-id-lookup-guard` | 仍有效；mock-auth 规则不变 |

**测试迁移：**

- `cloud/view_test/TenantCloudPlatformPermission_test.py` → Go 集成测 + 网关 E2E
- Playwright mock 路由可保留至 Go Mock 模式稳定

---

## 领域概念清单（供 /6-ddd）

| 有界上下文 | 实体/聚合 | 事件 |
|------------|-----------|------|
| **Cloud Platform** | `CloudPlatformAuthorization`（聚合根）、`ActiveMethod`（实体） | `CLOUD_PLATFORM_AUTHORIZATION_CREATED` |
| **Cloud Query** | 无状态查询服务（应用服务） | — |
| **Container Runtime** | `CloudServerConfig`（已有 Go SSOT） | 通过 `authorization_id` 引用 CPA |

---

## 🐍 Python 接口审批

**状态：scoped-down** — 本设计**不新增** Python HTTP 接口；删除 Django 公网 `tenant_cloud_*` 与 `CloudPlatformAuthorizationViewSet` 路由。

---

## 实施切片（垂直交付顺序）

| 切片 | 内容 | 验收 |
|------|------|------|
| **3g-1** | CPA + ActiveMethod 表与 CRUD + verify + Kafka | 授权设置页 CRUD |
| **3g-2** | toggle-active / active-list | 激活互斥 |
| **3h-1** | regions + zones | 项目详情页硬件 Tab 地域/可用区 |
| **3h-2** | available-instances + instance-details + bandwidth | 实例筛选 |
| **3h-3** | instance-price | 询价展示 |
| **3h-4** | debug image-support + 网关切流 + Django 删除 | E2E 全绿 |

预估：**2–3 周**（含对照测试与切流）。

---

## 🏛️ 架构变更影响

- **迭代版本**: v10 🎯 target
- **迭代名称**: Cloud Platform CPA + Aliyun Query API Go SSOT
- **作者**: claude
- **设计日期**: 2026-07-07 10:16
- **新增文件**（每视图三类伴生格式）:
  - 🆕 `docs/architecture/v10-enterprise-landscape-20260707-1016-claude.{puml,archimate,mermaid.md}`
  - 🆕 `docs/architecture/v10-application-integration-20260707-1016-claude.{puml,archimate,mermaid.md}`
- **变更明细**:
  - 🟡 [MODIFIED] taskCloudService — CPA SSOT + Aliyun SDK 查询 provider
  - 🟡 [MODIFIED] APISIX — `cloud-platform/*/cloud/*` + 授权 CRUD → :8018
  - 🔴 [DEPRECATED] Django `cloud/providers/aliyun/` 查询路径（交付后删除）
  - 🔴 [DEPRECATED] Django `tenant_cloud_platform_views_*`（交付后删除）
  - ✅ [CLOSED] Gap: Cloud split-brain — WP Phase 3g+3h

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| **Plateau v9** | CloudServerConfig Go SSOT；CPA/查询仍 Django |
| **Plateau v10** | CPA + Aliyun 查询全在 taskCloudService |
| **Gap** | 网关双路由 / Go stub 无 SDK / Django 出站 Aliyun |
| **WorkPackage** | 3g 授权 → 3h 查询 → 网关切流 → Django 删表 |

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| Go/Python SDK 响应字段漂移 | 契约测试对照 JSON fixture；保留 Playwright |
| Aliyun API 限流/超时 | Go context timeout + 与 Django 相同默认值 |
| 租户校验 internal 依赖 | 先 HTTP 调 saas-backend internal；后续可迁 taskAuth |
| 切流窗口数据不一致 | migrate 脚本 + 行数校验 |

---

## 验收标准

1. 项目详情页 `cloud/zones`、`cloud/regions` 经网关 :8018 返回 200
2. 授权 CRUD / verify / toggle 不经 Django ORM
3. `USE_IN_MEMORY_CLOUD=true` 时 Playwright 无需改 mock 即可绿
4. Django 表 `cloud_cloudplatformauthorization` 已 DROP
5. `value-stream.yaml` 字段归属与真源一致
