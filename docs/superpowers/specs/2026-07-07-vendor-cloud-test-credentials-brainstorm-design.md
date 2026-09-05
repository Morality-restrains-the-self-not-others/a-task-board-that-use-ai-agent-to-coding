# 镜像厂商云平台测试密钥 — 头脑风暴设计

- **日期**: 2026-07-07 22:15
- **状态**: shipped（goal-mode 交付）
- **页面**: http://183.250.1.132:8010/ 厂商门户
- **python_api_approval**: revised-to-go（用户 2026-07-07 选择改选 Go 方案）

---

## 1. 问题陈述

镜像厂商通过主站 SSO 进入 **Saas_Ai_Provider（:8010）厂商门户**，需要登记「云平台服务器镜像」并为容器镜像配置「各区域运行环境」。这些操作依赖调用云厂商 OpenAPI 拉取：

- 地域列表（`GET /api/vendor/cloud-server-images/regions/`）
- 镜像列表（`…/images/`）
- 实例规格（`…/instance-types/`）

**现状**：后端 `_get_cloud_credentials(platform_type)` 仅从 **进程环境变量** 读取全局密钥（`ALIYUN_ACCESS_KEY` / `ALIYUN_SECRET_KEY`），厂商门户 **没有任何 UI/API** 让厂商自行配置测试用 AccessKey。

后果：

| 场景 | 厂商体验 |
|------|----------|
| 运维未在 ai-provider 进程注入全局 AK | 区域列表为空 / API 400「未配置阿里云凭证」 |
| 多厂商共用平台 AK | 厂商只能看到平台账号下的镜像，**无法登记自己账号里的自定义镜像** |
| 主站租户 CPA 已有「测试密钥」 | 厂商门户与主站 **完全隔离**，租户 CPA 不可复用 |

主站 `CloudPlatformAuthorizationRow.vue` 已有「测试密钥」交互，但属于 **租户/公司** 维度（taskCloudService :8018），与 **厂商** 维度无关。

---

## 2. 当前架构理解（基线）

根据 `docs/architecture/` 与 VERSION_HISTORY：

- **架构视图（current 基线仍为 v1 `.puml`；最新已交付 target 为 v10）**
- **业务层**：Developer、Platform Admin、Enterprise Customer；镜像厂商通过镜像市场维护产品与版本
- **应用层**：
  - `saas-backend` Django (:8001) — 主站、租户镜像市场入口
  - `ai-provider` Django (:8010) — 镜像市场 / 厂商门户 / 运营后台
  - `task-cloud-service` Go (:8018) — 租户 CPA + 云查询 SSOT
  - `task-gateway` APISIX — 路由
- **厂商门户数据流**：主站 SSO bridge → ai-provider 换票 → Vendor Bearer → 厂商 CRUD 镜像
- **云凭证现状（ai-provider）**：
  - `utils._get_cloud_credentials` → 环境变量（厂商不可配）
  - `cloud_sdk/aliyun/part2.get_cloud_credentials_from_settings` → `conf/conf.yaml` 的 `cloud_platforms`（平台级，未接 VendorPortal）
- **上次架构大版本**：v10 — Cloud CPA + Aliyun Query 迁入 taskCloudService

📋 **近期架构演进**：

- v10 ✅ — CPA + Aliyun 查询 Go SSOT
- v9 🎯 — Cloud 域 Go 真源 Phase 3
- v8 🎯 — 任务域 Go 真源

本次迭代在 **v10 基础上** 为 ai-provider 增加 **厂商级云测试凭证**，不改动 taskCloudService 租户 CPA。

---

## 3. 目标与非目标

### 目标

1. 厂商在 :8010 门户内 **自助配置、校验、更新、删除** 各云平台测试 AccessKey（Phase 1：阿里云）。
2. 登记云服务器镜像、配置区域运行环境时，**优先使用厂商自己的凭证** 调用 OpenAPI。
3. 凭证 **加密落库**、响应 **脱敏**，与主站 CPA 安全约束对齐。
4. 未配置时给出 **可操作的引导**（跳转「云平台测试密钥」Tab）。

### 非目标（本迭代不做）

- 不将厂商凭证并入主站 CPA / taskCloudService（租户与厂商边界不同）。
- 不支持 OAuth 类云授权（仅 `access_key`，与现有 CloudSDKProxy 一致）。
- 不替厂商创建/销毁 ECS 实例（仅查询 regions/images/instance-types）。
- Phase 1 不实现腾讯云等（CloudSDKProxy 尚未实现则返回「暂不支持」）。

---

## 4. 方案对比

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| A. ai-provider 厂商凭证表 | Django CRUD + VendorPortal Tab | 同库简单 | 新增 Python 接口；单线程 WSGI |
| B. 运营 Admin 代填 | Staff 维护 | 实现量小 | 非自助 |
| C. 复用主站 CPA | 读租户 CPA | 复用 verify | 厂商≠租户 |
| **D. taskCloudService 厂商凭证 SSOT（推荐，用户已选）** | Go SQLite 存凭证 + Vendor JWT；ai-provider 云查询走 internal lookup | 复用 verify/Aliyun SDK；无新 Python CRUD；与 CPA 同服务不同表 | 跨服务 vendor_id 引用；需同源代理或 CORS |

**最终方案 D（Go SSOT）**：凭证 CRUD 与 `verify-credentials` 落在 **taskCloudService (:8018)**；ai-provider 仅改造现有 cloud-server-images 读凭证路径，**不新增 Python 业务 REST**。

---

## 5. 领域概念清单（供 /5-ddd 消费）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | `marketplace-vendor`（ai-provider 内） |
| **Entity** | `VendorCloudPlatformCredential` — 厂商×云平台 测试 AccessKey |
| **Aggregate Root** | `Vendor`（1:N credentials，每 platform 至多一条 active） |
| **Domain Event** | （可选 Phase 2）`VendorCloudCredentialVerified` — 审计/通知 |
| **不变量** | 同一 `vendor_id + platform_type` 唯一；secret 永不明文出 API/日志 |

---

## 6. 数据模型（taskCloudService SQLite）

### 6.1 表 `vendor_cloud_platform_credentials`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT PK | Snowflake 字符串 |
| vendor_id | TEXT NOT NULL | 镜像市场 Vendor.id（逻辑 FK，无跨库约束） |
| platform_type | TEXT NOT NULL | 如 `aliyun` |
| secret_id | TEXT NOT NULL | AccessKey ID |
| secret_key | TEXT NOT NULL | 加密存储（AES-GCM 或等同；密钥来自 `TASK_CLOUD_CREDENTIAL_ENCRYPTION_KEY`） |
| remark | TEXT | 备注 |
| last_verified_at | TIMESTAMP | 最近校验成功 |
| last_verify_error | TEXT | 最近失败摘要 |
| is_active | INTEGER | 0/1 |
| created_at / updated_at | TIMESTAMP | 审计 |

**约束**: `UNIQUE(vendor_id, platform_type)`

**与 CPA 关系**: 同库不同表；`cloud_platform_authorizations.company_id` 为租户，`vendor_cloud_platform_credentials.vendor_id` 为厂商，**禁止混用**。

---

## 7. API 设计

### 7.1 公网 API（taskCloudService :8018）

Base path: `/api/vendor/cloud-platform-credentials/`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 列表（脱敏） |
| POST | `/` | 创建 |
| GET | `/{id}/` | 详情 |
| PUT/PATCH | `/{id}/` | 更新 |
| DELETE | `/{id}/` | 删除 |
| POST | `/{id}/verify-credentials/` | 测试密钥（复用 `verifyAccessKeyCallerIdentity`） |

**鉴权 — Vendor JWT Middleware**:

- 校验 `Authorization: Bearer <vendor_token>`（与 ai-provider 签发 JWT 同算法/密钥，配置项 `MARKETPLACE_VENDOR_JWT_SECRET`，与 ai-provider `JWT` 段对齐）
- Payload 须含 `typ=vendor`、`sub=<vendor_id>`；middleware 注入 `vendor_id` 到 request context
- 所有 SQL 带 `vendor_id=?` 过滤

**Swagger**: 在 taskCloudService / 网关 Swagger 聚合中可见（gitOauth 元规则）。

### 7.2 Internal API（ai-provider → Go）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/internal/vendor-cloud-credentials/lookup` | Query: `vendor_id`, `platform_type`；返回 `{secret_id, secret_key}` 供服务端 cloud SDK |

- 鉴权：`X-Internal-Secret`（与现有 internal 约定一致）
- **仅服务端调用**；禁止浏览器直连

### 7.3 前端同源访问（:8010 VendorPortal）

厂商 SPA 当前 `fetch(path)` 为 **同源相对路径**。推荐：

**方案 D1（推荐）— ai-provider 透明反代**:

- Django 增加 **单一** catch-all：`/api/vendor/cloud-platform-credentials/` → 反向代理至 `http://127.0.0.1:8018`（保留 Authorization 头）
- 前端路径不变；Go 仍为业务 SSOT
- Python 侧仅为 HTTP 透传，**无业务逻辑**（不计入「新增 Python 业务接口」）

**方案 D2 — 前端直连 :8018**:

- `api.js` 对 credential 路径使用 `TASK_CLOUD_ORIGIN`
- 须在 taskCloudService 配置 CORS `Access-Control-Allow-Origin: http://183.250.1.132:8010`

交付默认 **D1**，避免生产 CORS 配置漂移。

### 7.4 ai-provider 凭证解析链（改造现有端点）

替换 `utils._get_cloud_credentials(platform_type)`：

```python
def resolve_vendor_cloud_credentials(vendor, platform_type) -> tuple[str, str]:
    # 1. HTTP GET internal lookup → taskCloudService
    # 2. dev-only: conf.yaml / env fallback（AI_PROVIDER_ALLOW_PLATFORM_CLOUD_FALLBACK=1）
    # 3. 抛 VendorCloudCredentialMissing（带 code 供前端引导）
```

`VendorCloudServerImageViewSet` regions/images/instance-types **不变 path**，仅改凭证来源。

---

## 8. 前端设计（VendorPortal.vue）

### 8.1 新 Tab：「云平台测试密钥」

位置：与「镜像组」「云平台服务器镜像」并列。

**列表列**: 云平台 | AccessKey ID（脱敏）| 备注 | 最近校验 | 状态 | 操作

**操作**:

- 添加 / 编辑（Modal：platform、secret_id、secret_key、remark）
- **测试密钥**（与主站 CPA 按钮文案一致）
- 启用/禁用 toggle
- 删除（二次确认）

**空状态**: 说明文案 + 「添加测试密钥」主按钮。

### 8.2 联动提示

在「云平台服务器镜像」登记 Modal、以及「各区域运行环境」展开面板中：

- 若 `regions` API 返回 `vendor_cloud_credential_missing` → 顶部 Alert：「请先在「云平台测试密钥」配置阿里云 AccessKey 并测试通过」+ 一键切 Tab。

### 8.3 组件拆分（遵守 500 行规则）

从 VendorPortal 抽出：

- `VendorCloudCredentialsTab.vue`
- `VendorCloudCredentialFormModal.vue`

---

## 9. 安全与合规

| 项 | 措施 |
|----|------|
| 存储 | secret_key 仅加密字段；禁止明文列 |
| 传输 | 生产 HTTPS；日志 incoming_requests 脱敏 body 中的 secret_key |
| 权限 | 厂商只能 CRUD 自己的凭证；Staff 只读（AdminPortal 可选 Phase 2） |
| 校验 | 阿里云 STS `GetCallerIdentity`（复用 ai-provider 现有 alibabacloud SDK） |
| 与主站 CPA 关系 | **独立**；厂商 AK 不写入 taskCloudService |

---

## 10. 价值流影响

受影响 `conf/value-stream.yaml` 流：

| 流 | 影响 |
|----|------|
| **云平台集成** (`cloud-compute` / CPA 相关) | 无直接变更（租户 CPA 仍在 :8018） |
| **ai-provider SSO / 厂商门户** | 新增步骤：厂商配置测试密钥 → 登记云镜像 → 配置区域运行环境 → 提交审核 |
| **镜像市场安装** | 间接：厂商能正确登记 runtime_environments 后，主站 dev-catalog / 上架镜像数据更完整 |

**新增价值流切片（建议 Step 3 写入）**:

```yaml
- name: vendor-cloud-test-credentials
  domain: 镜像市场
  steps:
    - name: vendor-configure-cloud-credential
      description: 厂商门户配置并 verify 云平台测试 AccessKey
    - name: vendor-register-cloud-server-image
      description: 使用厂商 AK 拉取 regions/images 并登记 CSI
```

**Fields（示例）**:

- `taskCloudService.vendor_cloud_platform_credentials.vendor_id`
- `taskCloudService.vendor_cloud_platform_credentials.secret_id`
- `taskCloudService.vendor_cloud_platform_credentials.last_verified_at`

**测试影响**:

- 新增 `taskCloudService/src/vendor_credential_handlers_test.go`
- 新增 ai-provider 集成测：`test_resolve_vendor_cloud_credentials_via_go.py`（mock internal lookup）
- 新增 Playwright `django8010-vendor-cloud-credentials.playwright.test.js`
- 现有 vendor SSO E2E 可 mock STS 或使用 `USE_IN_MEMORY_CLOUD` 等价开关

---

## 11. 🐍 Python 新增接口清单与 Go 替代评估

> **用户审批**: revised-to-go（2026-07-07）

### 拟新增接口（Go — taskCloudService）

| # | 方法 | 路径 | 归属 | 说明 |
|---|------|------|------|------|
| 1–6 | CRUD + verify | `/api/vendor/cloud-platform-credentials/…` | taskCloudService :8018 | 见 §7.1 |
| 7 | GET | `/api/internal/vendor-cloud-credentials/lookup` | taskCloudService internal | ai-provider 云 SDK 读凭证 |

### Python 侧变更（非新增公网 CRUD）

| 变更 | 类型 | 说明 |
|------|------|------|
| `resolve_vendor_cloud_credentials()` | 修改现有 utils | internal HTTP → Go |
| `VendorCloudServerImageViewSet` | 修改现有 3 个 action | 使用 vendor 凭证 |
| `/api/vendor/cloud-platform-credentials/*` 反代 | 可选透传路由 | D1 同源方案；无业务逻辑 |

### 选型结论

- **最终选择**: Go taskCloudService SSOT + ai-provider internal 消费 + 同源反代
- **Python 专项审批结论**: **scoped-down** — 不新增 Python 业务 CRUD；仅改现有 cloud-server-images 凭证链 + 可选 HTTP 反代

---

## 12. 实施计划（垂直切片）

### Phase 1 — 阿里云 MVP（本设计范围）

1. taskCloudService：migration + vendor credential handlers + Vendor JWT middleware + 单元测试
2. taskCloudService：internal lookup + 复用 `verifyAccessKeyCallerIdentity`
3. ai-provider：internal lookup 客户端 + 改造 `resolve_vendor_cloud_credentials`
4. ai-provider：D1 反代路由（或 APISIX 规则）
5. VendorPortal 新 Tab + 组件拆分 + 联动 Alert
6. Playwright E2E + Swagger

### Phase 2（后续）

- 腾讯云等 CloudSDKProxy 扩展
- AdminPortal 只读审计视图
- 可选 domain event 通知运营「厂商已校验 AK」

---

## 13. 验收标准

- [x] 厂商 SSO 登录后可见「云平台测试密钥」Tab
- [x] 可添加阿里云 AK/SK，点击「测试密钥」返回 caller_identity
- [x] 配置成功后，「云平台服务器镜像」登记流程可拉取 regions/images
- [x] 未配置时错误信息引导至该 Tab（非 cryptic 500）
- [x] API 响应与日志不含明文 secret_key
- [x] 厂商 A 无法读写厂商 B 的凭证

---

## 14. 🏛️ 架构变更影响（审批通过后写入 v11 target）

- **迭代版本**: v11 🎯 target
- **迭代名称**: 镜像厂商云平台测试密钥
- **变更视图**: `application-integration`（ai-provider 组件与数据流）；`enterprise-landscape` 若需标注新 DataObject
- **变更明细**:
  - 🟢 [NEW] taskCloudService `vendor_cloud_platform_credentials` 表 + Vendor JWT API
  - 🟢 [NEW] internal lookup 供 ai-provider 云 SDK
  - 🟡 [MODIFIED] ai-provider 云 SDK 凭证解析链（Go lookup + dev fallback）
  - 🟡 [MODIFIED] ai-provider → Go 反代（同源）
  - 🟡 [MODIFIED] VendorPortal 前端 Tab
- **伴生文件**（审批后生成）: `.puml` + `.archimate`（含 Plateau v10→v11 变迁视图 + sourceConnection）+ `.mermaid.md`

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v10 | 厂商门户云查询依赖平台 env AK |
| Plateau v11 | 厂商自管测试凭证 → taskCloudService SQLite → ai-provider internal lookup → CloudSDKProxy |
| Gap | 厂商无法自配云测试密钥 |
| WorkPackage | WP-v11-vendor-cloud-credentials |

---

## 15. 开放问题（可选确认）

1. **生产是否允许平台级 fallback AK？** 建议默认关闭，仅 dev/staging 可开。
2. **每个云平台是否允许多条凭证？** 建议 Phase 1 每 platform 一条（唯一约束），与 CPA 类似。
3. **运营是否需要在 AdminPortal 查看厂商凭证状态（不含 secret）？** 建议 Phase 2。
