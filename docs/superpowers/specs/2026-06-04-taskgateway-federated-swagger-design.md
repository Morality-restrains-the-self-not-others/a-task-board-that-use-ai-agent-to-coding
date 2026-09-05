# 设计：taskGateway 分服务 Swagger 门户（开发环境）

**日期**：2026-06-04  
**状态**：已实现（D1–D4，2026-06-04）  
**关联**：`docs/superpowers/specs/2026-06-02-taskgateway-apisix-design.md`、`taskGateway/routes/routes.yaml`、`conf/task-gateway/config.yaml`

**用户决策（头脑风暴确认）**：

| 项 | 选择 |
|----|------|
| 文档形态 | **分服务门户**（非单一合并 OpenAPI） |
| 环境 | **仅本地/开发**（`docs.enabled` 默认关，生产不生成 docs 路由） |

---

## 1. 背景与问题

### 1.1 现状

| 组件 | OpenAPI / Swagger | 经 taskGateway 可达性 |
|------|-------------------|----------------------|
| django (saas-backend) | drf_yasg：`/swagger/`、`/swagger?format=openapi` | 默认 `/*` 需 `auth_mode: token`，未登录无法打开 UI |
| git-oauth | drf-spectacular：`/api/swagger/`、`/api/schema/` | 无专用路由，`/api/swagger/` 误打到 django upstream |
| task-auth (Go) | 无 | 登录等路径走 taskAuth upstream，Django Swagger 不包含 |
| task-sse / task-container-gateway (Go) | 无 | 仅少量路由经网关 |

浏览器生产路径已统一为 `https://<gateway>:8443/api/...`，但**无法在网关入口调试全部转发 API**，且各服务文档分散、与路由表脱节。

### 1.2 目标

1. 开发者在 **`https://<publicBase>/gateway/docs/`** 看到所有已注册 upstream 的文档入口。
2. 每个 upstream 的 Swagger UI **经网关反代**，「Try it out」请求走真实网关路径（含 forward-auth、路由拆分）。
3. **新增 upstream** 时，只要在 `routes.yaml` 声明 `docs` 块并跑 `routes-apply`，门户与代理路由**自动生成**（无需手改 APISIX 片段）。
4. 生产/CI 发布路径**不暴露** docs（配置开关 + codegen 不产出 docs 路由）。

### 1.3 非目标（V1）

- 单一合并 OpenAPI（多服务 path 去重、auth_mode 标注）→ 留 V2 可选。
- 为所有 Go 服务一次性补齐完整 OpenAPI（按 upstream 分阶段）。
- 在 Swagger UI 内模拟 forward-auth / 自动注入 `X-User-Id`（仍由用户填写 `Authorization: Token`）。
- 替代各服务自身的 schema 维护流程（gitOauth 现有 Swagger 规则仍有效）。

---

## 2. 价值流影响

依据 `value-stream.yaml` 中 **`task-gateway`** stream：

| 影响 | 说明 |
|------|------|
| **受影响 stream** | `task-gateway`（新增 docs 路由 codegen、门户页生成） |
| **交叉依赖** | `taskFE.config.api_base_url`（Try it out 的 server 须与 `publicBase` 一致）；`user-auth`（taskAuth 文档补齐后可在门户列出） |
| **新 step（建议）** | `gateway-docs-portal`（门户 HTML + 路由存在性测试）、`gateway-openapi-upstream-contract`（upstream `docs` 契约 CI） |
| **测试** | 扩展 `tests/test_taskgateway_routes_codegen.py`；可选 `view_test/task-gateway-docs.md` 手测清单 |
| **字段（三步命名）** | `task-gateway.docs.enabled`、`task-gateway.docs.portal_path`、`task-gateway.routes.docs_proxy` |

与 `gitoauth-swagger` 规则：**git-oauth 仍在本服务维护 schema**；网关只代理展示，不生成 gitOauth 契约。

---

## 3. 领域概念清单（供 `/5-ddd`）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | Edge Gateway（文档门户）、各业务服务（契约源） |
| **实体** | `Upstream`（含 `DocsDescriptor`）、`DocsPortal`（聚合索引页）、`ProxiedSchemaRoute` |
| **值对象** | `schemaPath`、`uiPath`、`docsEnabled` |
| **聚合** | `GatewayRouteTable` — docs 路由与业务路由同版本发布 |
| **领域事件** | `GatewayDocsRoutesPublished`（可选，V1 用 Git + codegen 即可） |

---

## 4. 方案对比

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. 声明式反代 + codegen（推荐）** | `routes.yaml` 的 `upstreams.*.docs` 驱动门户与 APISIX 路由 | 与现有 `routes-to-apisix.py` 一致；新服务只改 YAML | 各服务须自备 OpenAPI；UI 静态资源路径需 rewrite |
| **B. 构建时拉取合并 spec** | CI 拉各服务 schema 合并为一个 JSON | 单文件浏览 | 与「分服务门户」决策冲突；合并逻辑重 |
| **C. 独立 docs 微服务** | 新容器聚合爬取 | 灵活 | 多进程、与 YAGNI 冲突 |

**推荐 A**：最小增量，复用 APISIX `proxy-rewrite` + 生成静态门户页。

---

## 5. 目标架构

```text
开发者浏览器
    │  GET https://<publicBase>/gateway/docs/
    ▼
┌─────────────────────────────────────────┐
│  taskGateway (APISIX)                    │
│  docs.enabled=true 时额外路由：          │
│  • /gateway/docs/          → 静态门户页   │
│  • /gateway/docs/{upstream}/* → 反代 UI   │
│  • /gateway/openapi/{upstream}.json       │
│      → 反代 schema + 可选 servers 补丁    │
│  auth_mode: none（仅 docs 路由）          │
└──────────┬──────────────────────────────┘
           │ 按 upstream 转发
     ┌─────┼─────┬──────────┐
     ▼     ▼     ▼          ▼
  django gitOauth taskAuth  …
  :8001   :8002    :8003
```

**门户页**：`routes-apply` 根据 `upstreams` 生成 `taskGateway/generated/docs-portal/index.html`（链接到各 `/gateway/docs/{id}/`）。Compose 将 `generated/docs-portal` 挂载到 APISIX 可读目录，或由轻量 `file-server` 容器提供（实现阶段二选一，优先 APISIX `uri` + `response-rewrite` 返回静态文件若可行，否则同 compose 内 `nginx:alpine` 仅 dev profile）。

---

## 6. 配置契约

### 6.1 `conf/task-gateway/config.yaml`

```yaml
docs:
  enabled: false          # 本地 dev 可 true；生产模板保持 false
  portalPath: /gateway/docs/
  schemaPathPrefix: /gateway/openapi/   # + {upstreamId}.json
```

### 6.2 `taskGateway/routes/routes.yaml` — upstream 扩展

在现有 `upstreams` 块为每个逻辑服务增加 **`docs`**（与 `host`/`port` 同级）：

```yaml
upstreams:
  django:
    host: 10.2.150.89
    port: 8001
    docs:
      uiPath: /swagger/
      schemaPath: /swagger/?format=openapi
  gitOauth:
    host: 10.2.150.89
    port: 8002
    docs:
      uiPath: /api/swagger/
      schemaPath: /api/schema/
  taskAuth:
    host: 10.2.150.89
    port: 8003
    docs: false   # V1 占位；补齐 OpenAPI 后改为 uiPath/schemaPath
  taskSse:
    host: 10.2.150.89
    port: 8798
    docs: false
  taskContainerGateway:
    host: 10.2.150.89
    port: 8014
    docs: false
```

**契约规则（CI 强制）**：

| 规则 | 说明 |
|------|------|
| R1 | `routes.yaml` 中每个 `upstreams` 键必须有 `docs`（对象或 `false`） |
| R2 | `docs` 为对象时必含 `uiPath`、`schemaPath`（以 `/` 开头） |
| R3 | `docs.enabled=false` 时，生成的 `apisix.yaml` **不得**含 `/gateway/docs` 路由 |
| R4 | 新增 upstream 时同步 R1–R2，门户自动出现或显示「未提供文档」 |

**标准路径（新服务推荐）**：`uiPath: /api/swagger/`、`schemaPath: /api/schema/`（与 gitOauth / drf-spectacular 默认一致）。

---

## 7. 路由与 codegen

### 7.1 `routes-to-apisix.py` 扩展

当 `conf.task-gateway.docs.enabled` 为 true 时，在业务路由之前注入（高 `priority`，如 `2100`）：

| 路由 id | URI | Upstream | auth_mode | 插件 |
|---------|-----|----------|-----------|------|
| `gateway-docs-portal` | `/gateway/docs/` | 静态或 portal 容器 | `none` | — |
| `gateway-docs-{upstream}-ui` | `/gateway/docs/{upstream}/*` | 对应 upstream | `none` | `proxy-rewrite`：`^/gateway/docs/{upstream}/(.*)` → `/$1` |
| `gateway-openapi-{upstream}-schema` | `/gateway/openapi/{upstream}.json` | 对应 upstream | `none` | rewrite 到 `schemaPath` |

`auth_mode: none` 仅用于 **文档与 schema JSON**；业务 API 路由表不变。

### 7.2 Try it out 与 Server URL

各服务 OpenAPI 中 `servers.url` 应指向 **`TASK_GATEWAY_PUBLIC_BASE`**（非 upstream 端口）：

| 服务 | 配置手段 |
|------|----------|
| django | `SWAGGER_SETTINGS['DEFAULT_API_URL']` 或环境变量，从 `conf` sync |
| gitOauth | `SPECTACULAR_SETTINGS['SERVERS']` |
| taskAuth (后续) | `openapi.yaml` 内 `servers` 或 swag 生成参数 |

门户可额外提供说明：在 Swagger UI 顶部填写 **Authorize → Token &lt;key&gt;**，以走 forward-auth。

### 7.3 与现有 `auth_mode: token` 的关系

- `/swagger/`（django 根路径）**不再**依赖用户直接访问 `django-default` 的 `/*` + token。
- 开发态统一入口：`/gateway/docs/django/swagger/`（示例，以 rewrite 后 upstream 为准）。

---

## 8. 各服务 OpenAPI 补齐计划

| Upstream | V1 | 说明 |
|----------|-----|------|
| django | 已有 | 调整 `DEFAULT_API_URL` 指向 gateway；文档中标注已迁移至 taskAuth 的路径为 deprecated |
| gitOauth | 已有 | 仅加网关反代；继续遵守 gitoauth-swagger 规则 |
| taskAuth | **已实现（D4）** | Go：`src/openapi.yaml` + `GET /api/schema/`、`GET /api/swagger/`；网关 `docs` 已启用 |
| taskSse | V2 | `docs: false`，门户显示「未提供」 |
| taskContainerGateway | V2 | 同上 |

---

## 9. 安全与运维

| 项 | 策略 |
|----|------|
| 生产暴露 | `docs.enabled: false`；发布检查脚本断言无 `/gateway/docs` |
| 鉴权 | 文档路由 `none`；仅 dev 网络可达假设（与现有自签 TLS dev 一致） |
| 内部 API | `/api/internal/*` 仍 `deny`；不得出现在各服务 public schema |
| 限流 | docs 路由可选宽松 `limit-req`（防误扫）；V1 可省略 |

---

## 10. 测试与验收

| 验收项 | 方法 |
|--------|------|
| 门户可打开 | `curl -k https://<host>:8443/gateway/docs/` 返回 HTML 含 django、gitOauth 链接 |
| django UI | 浏览器打开 `/gateway/docs/django/...`，Try it out `GET /api/health/` 经 gateway 200 |
| gitOauth UI | `/gateway/docs/gitOauth/...` 可加载 schema，OAuth 路径与 `routes.yaml` 一致 |
| 生产关断 | `docs.enabled=false` 时 `apisix.yaml` 无 gateway-docs 路由；`check_taskgateway_routes.sh` 通过 |
| 契约 CI | 新测试：每个 upstream 声明 `docs`；enabled 时生成路由 id 快照 |

---

## 11. 实施增量（供 `/3-value-stream` / `/6-plans`）

| Inc | 名称 | 交付 |
|-----|------|------|
| **D1** | gateway-docs-config-codegen | `docs` 配置 + `routes-to-apisix` 生成反代路由 + 门户 HTML |
| **D2** | gateway-docs-django-gitoauth | django/gitOauth `SERVERS`/`DEFAULT_API_URL` + 手测清单 |
| **D3** | gateway-docs-upstream-contract-ci | R1–R4 CI 测试 |
| **D4** | taskauth-openapi-stub | taskAuth `openapi.yaml` + schema 端点 + 门户启用 |

---

## 12. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Swagger UI 静态资源路径错误 | 集成测试 + 手测；rewrite 规则按 upstream 单测 |
| django schema 含已迁出路径 | 逐步 `@swagger_auto_schema` 标记 deprecated；不阻塞 D1 |
| APISIX 静态门户挂载复杂 | D1 可用 compose 侧 car nginx 仅 dev |
| taskAuth 无 OpenAPI 拖延 | `docs: false` 显式占位，门户标注「待补齐」 |

---

## 13. 未决项（实现前可默认）

| 项 | 默认 |
|----|------|
| 门户静态文件服务方式 | 优先 dev compose `nginx` 挂载 `generated/docs-portal`；不与 APISIX 核心耦合 |
| schema 是否运行时 patch `servers` | 优先各服务 settings 配置；网关仅反代 |

---

## 14. 参考

- `taskGateway/routes/routes.yaml`
- `taskGateway/scripts/routes-to-apisix.py`
- `task2app/Saas_project/saas_project/urls.py`（drf_yasg）
- `gitOauth/config/urls.py`（drf-spectacular）
- `value-stream.yaml` → `task-gateway`
