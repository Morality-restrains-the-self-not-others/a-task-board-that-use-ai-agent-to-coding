# 浏览器插件固定 ID 改为管理员后台可配置 — 设计文档

- **迭代**: browser-extension-oidc-extension-id-admin
- **作者**: claude
- **设计日期**: 2026-08-08
- **关联**: OPT-20260808-024（taskChromePlugin OAuth2+PKCE 登录迁移，阶段 0a 的固定 ID 一致性硬前提由此设计解决）
- **状态**: 已批准（2026-08-08，管理权属落 DB：adminManaged）

## 1. 背景与目标

### 1.1 背景

taskAuth 内置 OIDC Provider（`/api/oidc/authorize|token|userinfo`）为 Chrome 插件登录服务（OPT-20260808-024）。OIDC client `chrome-extension` 的授权回调白名单 `redirect_uris` 依赖 Chrome 扩展 ID：

```yaml
# conf/auth/task-auth/config.yaml
bootstrapClients:
  - clientId: "chrome-extension"
    clientSecret: "chrome-extension-dev-secret"
    redirectUri: "chrome-extension://cmkahnnaofomeaodefegkgljniiphbhj/oauth-callback.html"
```

该 ID 硬编码于 conf，且 `seedOidcBootstrapClients()`（oidc_bootstrap.go）启动时经 `ensureOidcClient`（oidc_db.go:53）**自愈覆盖** DB 中 `auth_oidc_client.redirect_uris`（不一致即 UPDATE）。扩展 ID 一旦变化（重建 key、不同构建/环境、浏览器重新加载无 key 的扩展），必须改 conf 并重启 taskAuth 才能放行，且管理员改动会被自愈逻辑覆盖——不可运维。

### 1.2 目标

1. 插件 ID 由「conf 硬编码固定值」改为「管理员后台可配置」：管理员在系统管理后台维护插件 ID 列表，系统自动生成 `chrome-extension://<id>/oauth-callback.html` 白名单并写入 `auth_oidc_client`，**立即生效，无需改 conf / 重启**。
2. 消除 bootstrap 自愈对管理员改动的覆盖（conf 托管的 client 保持原自愈语义，admin 托管的 client 以 DB 为准）。
3. 与 OPT-024 插件 OAuth2+PKCE 登录联动：插件 ID 未注册时给用户可操作的错误提示（而非笼统失败）。

### 1.3 非目标

- 不改变 Chrome 扩展 ID 的生成机制（见 §2.1，浏览器安全模型不可改）。
- 不做通用 OIDC client 管理后台（仅插件 client 专用，通用化留待未来）。
- 不新增插件侧公开 API（插件不查询服务端注册状态，仅错误提示联动）。

## 2. 需求澄清与关键决策

### 2.1 概念澄清：为什么「配置插件 ID」= 配置服务端白名单

Chrome 扩展 ID 由 manifest `key` 派生，由浏览器强制；**运行时无法改变**。因此「把插件固定 ID 改为可配置」的唯一合理落点是**服务端 OIDC client 的 `redirect_uris` 白名单**：任何 ID 的扩展只要其 ID 被管理员加入白名单，即可完成 OIDC 登录。插件侧 authorize URL 用 `chrome.runtime.id` 动态构造（本身已动态）。

### 2.2 核心矛盾与决策：bootstrap 自愈覆盖 + 管理权属落 DB

`ensureOidcClient` 的「self-healing on config change」UPDATE 会覆盖管理员写库的 redirect_uris。决策（用户审批：**adminManaged 落 DB**）：

- `auth_oidc_client` 新增列 **`managed_by VARCHAR(32) NOT NULL DEFAULT 'bootstrap'`**（值：`bootstrap` | `admin`），管理权属**持久化在 DB**，重启/换 conf 不丢失。
- seed 语义按行判定：`managed_by='admin'` 的行 seed 仅 INSERT 缺失、**不 UPDATE**（admin 托管，以 DB 为准）；`managed_by='bootstrap'` 维持原自愈 UPDATE（conf 托管，行为不变）。
- conf `chrome-extension` 条目**保留**作为 seed 值（首装/DB 重建时插入默认行，`managed_by` 由 migration 置为 `admin`）；conf 不新增字段。
- dataMigrate 新增 migration `031_oidc_client_managed_by.sql`：`ALTER TABLE ... ADD COLUMN managed_by` + `UPDATE ... SET managed_by='admin' WHERE client_id='chrome-extension'`（幂等；兼容已存在/不存在该行的库）。
- 管理端点写库时**幂等置位** `managed_by='admin'`（一次 PUT 即完成接管，防 bootstrap 托管态覆盖竞态）。

### 2.3 API 形态决策：专用端点，非通用 CRUD

极简优先：API 层做**插件 OIDC 配置专用端点**（非通用 client 管理），校验规则定制（插件 ID 格式），避免通用 redirect_uris 编辑的越权面与校验复杂化。落点 taskAuth `handlers_system_admin.go`（Go，复用 `requireSuperuser` 鉴权，符合 Go-first 元规则）。

### 2.4 插件 ID 输入形态

管理员输入**插件 ID 列表**（每行一个），服务端拼装 `chrome-extension://<id>/oauth-callback.html`。不开放任意 redirect_uri 编辑（防止 `http://evil` 之类非法回调注入）。

## 3. 现状分析（关键事实）

| 事实 | 位置 | 影响 |
|------|------|------|
| OIDC bootstrap seed：INSERT 缺失 + UPDATE 差异（自愈） | taskAuth `oidc_bootstrap.go` / `oidc_db.go:53-79` | 自愈会覆盖管理员改动 → 需 `adminManaged` 语义 |
| OIDC client 表 | `dataMigrate/taskAuth/003_oidc_tables.sql` `auth_oidc_client(id, client_id, client_secret_hash, name, redirect_uris, created_at, updated_at)` | 无需新表/新列 |
| authorize 校验 redirect_uri ∈ client.RedirectURIs | `oidc_handlers.go`（authorize L40-58 区间） | 白名单即生效点 |
| 系统管理后台鉴权模式 | taskAuth `handlers_system_admin.go` `requireSuperuser` | 新端点复用同一模式 |
| 管理后台前端 | taskFE `/system-admin/*`（SystemAdmin.vue 快捷卡片 + SystemAdminSidebar + 子路由） | 新页 + 入口 + 路由 |
| 网关路由 | taskGateway `apisix/apisix.yaml` `/api/system-admin/*` 组 | 新增一条 URI 前缀 |
| 插件当前回调页 | taskChromePlugin 无 oauth-callback.html（OPT-024 阶段 2 新增） | 本设计不涉及插件实现，仅错误提示联动 |

## 4. 方案设计

### 4.1 dataMigrate 变更（db 子仓）— 管理权属落 DB

新增 `dataMigrate/taskAuth/031_oidc_client_managed_by.sql`：

```sql
ALTER TABLE auth_oidc_client ADD COLUMN managed_by VARCHAR(32) NOT NULL DEFAULT 'bootstrap';
UPDATE auth_oidc_client SET managed_by = 'admin' WHERE client_id = 'chrome-extension';
```

- 幂等：`ADD COLUMN` 加 IF NOT EXISTS 防护（MySQL 8 支持 `ADD COLUMN IF NOT EXISTS`，按 dataMigrate 既有写法适配）；UPDATE 无匹配行零影响。
- 部署门禁（OPT-013 教训）：推送含 dataMigrate 新文件的子仓后，**必须先执行 9999 init-databases 再重启 taskAuth**，否则 seed 对 `managed_by='bootstrap'` 的行仍自愈覆盖。

### 4.2 conf 变更（conf 子仓）

`conf/auth/task-auth/config.yaml`：`chrome-extension` 条目**保持不变**（作为 seed 默认值，首装/DB 重建时 INSERT 默认行，随后 migration 置 `managed_by='admin'`）。conf 不新增字段。

### 4.3 taskAuth 变更（Go）

**4.3.1 seed 语义**（oidc_bootstrap.go / oidc_db.go）：

- `oidcClientRow` 增加 `ManagedBy` 字段（SELECT 读取 `managed_by`）。
- `ensureOidcClient`：命中已有行时，若 `row.ManagedBy == "admin"` → **跳过 UPDATE**（INSERT-only 语义）；否则维持原自愈 UPDATE。INSERT 新行时 `managed_by` 用默认值 `'bootstrap'`（显式写列，兼容 default）。

**4.3.2 管理端点**（handlers_system_admin.go，均 `requireSuperuser`）：

| 方法 | 路径 | 请求 | 响应 |
|------|------|------|------|
| GET | `/api/system-admin/oidc-extension/` | — | `{client_id, name, managed_by, extension_ids: [...], raw_redirect_uris: [...]}` |
| PUT | `/api/system-admin/oidc-extension/` | `{extension_ids: ["32位小写a-p×32"]}` | 更新后的 `{client_id, managed_by, extension_ids, raw_redirect_uris}` |

校验规则（服务端权威，前端同规则预校验）：
- 插件 ID：正则 `^[a-p]{32}$`（Chrome 扩展 ID 字符集/长度）；
- 去重（大小写敏感，ID 全小写天然无大小写问题）；上限 20 个；至少 1 个；不允许空字符串；
- client 不存在（`loadOidcClient("chrome-extension") == nil`）→ 404 带 trace_id（writeError 惯例，注入 trace_id，遵循 OPT-053/059 契约）；
- 写库（幂等接管）：`UPDATE auth_oidc_client SET redirect_uris = <JSON数组>, managed_by = 'admin', updated_at = ? WHERE client_id='chrome-extension'`，返回新状态；
- 错误统一 `writeError`（含 `trace_id`，与 OPT-059 推广方向一致）。

**4.3.3 路由注册**：taskAuth RegisterRoutes 增加 `/api/system-admin/oidc-extension/`（GET/PUT）handler。

### 4.4 网关路由（taskGateway 子仓）

`apisix/apisix.yaml` 系统管理组新增：

```yaml
- id: api-system-admin-oidc-extension
  uri: /api/system-admin/oidc-extension/*
  # 与 api-system-admin-users 同 upstream（taskAuth）、同鉴权插件（superuser 由 taskAuth requireSuperuser 兜底）
```

### 4.5 taskFE 管理页（前端）

- 新路由 `/system-admin/oidc-extension/` → 新页 `SystemAdminBrowserExtension.vue`（命名对齐现有 SystemAdmin* 系列）。
- SystemAdmin.vue 快捷卡片区新增「浏览器插件」入口（与「用户管理/云平台授权」同卡片模式）。
- 页面功能：加载 GET 现状 → 插件 ID 列表展示（chips + textarea 编辑）→ 校验（前端正则 `^[a-p]{32}$`、去重、上限）→ 保存 PUT → 成功/失败提示（失败含 data-traceId，复用 resolveRequestTraceId 契约）。
- URL 契约：`/api/system-admin/oidc-extension/`（尾斜杠，对齐 system-admin 系列）。

### 4.6 taskChromePlugin 联动（随 OPT-20260808-024 阶段 2 实施）

插件 OAuth 登录失败时错误分类提示：
- authorize 响应 `redirect_uri not allowed` / token 端点 `401 invalid_client` → 提示：「插件 ID 未注册：`<chrome.runtime.id>`，请联系管理员在系统后台 → 系统管理 → 浏览器插件中添加」。
- 实现位置：oauth-callback.html 失败渲染 + SW oauthCallback 错误分支（OPT-024 阶段 2 新增代码内）。

## 5. 测试计划

| 层 | 用例 |
|----|------|
| taskAuth 单测 | PUT 校验：非法格式/超限/去重/空列表/客户端不存在 404；GET 现状；写库后 loadOidcClient 可见（含 managed_by='admin'）；错误体含 trace_id |
| taskAuth 单测 | seed 语义：`managed_by='admin'` 行 DB 已有不同 redirect_uris → seed 不覆盖；`managed_by='bootstrap'` 行仍自愈覆盖（回归） |
| dataMigrate | 031 migration 幂等性：列存在/缺列/行存在/行缺失四场景重跑不报错 |
| taskFE 单测 | 页面加载渲染现状、编辑保存触发 PUT URL 断言、非法 ID 前端拦截、失败 data-traceId 展示 |
| e2e（可选） | Playwright：管理后台保存新 ID → taskAuth authorize 用新 ID 返回 code（浏览器级回归） |

## 6. 🕸️ Code Review Graph 分析

`CRG unavailable: .code-review-graph/graph.db 过期（2026-08-06，head_sha 不匹配），仅 12 文件/93 节点且无 Go 覆盖（taskAuth/taskGateway/taskFE 均不在图内）`。设计基于直接源码阅读（oidc_bootstrap.go / oidc_db.go / oidc_handlers.go / handlers_system_admin.go / apisix.yaml / SystemAdmin.vue / router.js），非 CRG 图。

## 7. Domain Concept Inventory（轻量）

- **Bounded Context**: 认证（Authentication）— OIDC Provider 与 client 配置；系统管理（System Admin）。
- **Key Entities**: `OidcClient`（auth_oidc_client，含 redirect_uris 白名单）— 本设计唯一事实变更对象。
- **Candidate Aggregates**: OidcClient 自身（单行一致性，无跨实体事务）。
- **Domain Events**: 无新增事件。

### 7.1 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 管理员更新插件 ID 白名单 | — | — | — | 纯配置管理：单行 UPDATE + 日志记录，无跨聚合/跨服务副作用、不触发下游行为变更（OIDC authorize 校验即时生效，无需事件）；「无对应事件」+ 理由按规范登记 |
| 插件 OIDC 登录（关联流程） | — | — | — | 既有 OIDC 流程（authorize/token/userinfo），非本次新增；OPT-024 范围内评估 |

## 8. Value Stream Impact

`conf/value-stream.yaml` user-auth 域（用户与认证）既有 login/activate/oidc 相关流（如 `taskauth-oidc-issuer-docker-reachability`、`ai-provider-oidc-login`、`oidc-slo-logout-sync`）。本设计影响面：

- **受影响流**：user-auth 域 —— OIDC 客户端配置来源从 conf 单源变为 conf+admin DB 双源（admin 优先级）。
- **字段影响**：`task-auth.auth_oidc_client.redirect_uris`（既有字段，写入方新增管理端点）。
- **新增流建议**：`browser-extension-oidc-config`（管理员维护插件白名单 → auth_oidc_client.redirect_uris → authorize 校验放行），由 /4-value-stream 决定是否落 yaml。
- **测试影响**：taskAuth handlers_system_admin_test（新增）、taskFE SystemAdminBrowserExtension 单测（新增）、既有 oidc_bootstrap/oidc_handlers 回归。

## 9. 🐍 Python 服务新增接口门禁

`not_applicable` — 全部变更落在 Go（taskAuth）与前端（taskFE）+ 配置（conf/taskGateway），无 Python 服务新增接口。

## 10. 🏛️ 架构变更影响（概要；正式文件批准后生成）

- **迭代版本**: v16 🎯 target（enterprise-landscape，基于 v15）/ v68 🎯 target（application-integration，基于 v67）
- **变更明细**:
  - 🟡 [MODIFIED] `taskAuth` — system-admin 新增插件 OIDC 配置端点（GET/PUT `/api/system-admin/oidc-extension/`）；bootstrap 按 `managed_by='admin'` 行执行 INSERT-only 语义
  - 🟡 [MODIFIED] `authDB.auth_oidc_client` — 新增 `managed_by` 列（bootstrap|admin）；`redirect_uris` 写入方扩展（conf 自愈 → DB 权属判定 + 管理端点写库）
  - 🟡 [MODIFIED] `taskFE` SystemAdmin — 新增「浏览器插件」管理页 + 路由 + 快捷入口
  - 🟡 [MODIFIED] `taskGateway` APISIX — 新增 `/api/system-admin/oidc-extension/*` 路由
- **新增文件**（2026-08-08 17:32 已生成）:
  - 🆕 `v16-enterprise-landscape-20260808-1732-claude.{puml,archimate,mermaid.md}`（landscape：4 组件 MODIFIED + Plateau v16/Gap/WP + 管理链 Rel_Flow）
  - 🆕 `v68-application-integration-20260808-1732-claude.{puml,archimate,mermaid.md}`（integration：4 组件 MODIFIED + 管理链 Rel_Flow taskFE→gateway→taskAuth→taskAuthDB）
  - 两视图 .archimate 均含「架构变迁 v(N-1)→vN + 插件 ID 白名单管理链拓扑」双视图 + sourceConnection 完整连线；Archi CLI 加载验证通过；PlantUML 渲染验证通过（既有 line 11 字体警告与 v15 基线一致，非致命）
- **⚠️ target 积压提示**：landscape v14/v15、integration v64-v67 均为未交付 target，本次基于最新 target 链（v15/v67）继续。

## 11. 部署与数据

1. **部署顺序（硬门禁，OPT-013 教训）**：db 子仓推送（含 031 migration）→ **先执行 9999 init-databases**（031 置 `managed_by='admin'`）→ 再重启 taskAuth（seed 对 admin 行 INSERT-only）。
2. conf 无变更（chrome-extension 条目原样保留作 seed）。
3. taskAuth / taskGateway / taskFE 部署无其他强依赖；管理员在后台保存即生效。
4. 生产现状：`auth_oidc_client` 中 chrome-extension 行 redirect_uris 含 `cmkahnnaofomeaodefegkgljniiphbhj`，保留；新 ID 由管理员追加。

## 12. 遗留/加固项

1. 插件启动自检「本 ID 是否已注册」（需公开只读端点或授权后查询）— 暂缓，登录失败提示兜底。
2. 通用 OIDC client 管理后台（其他 client 也可 admin 化）— 本次不做。
3. redirect_uris 变更审计（谁/何时改）— 本次仅结构化日志，不建审计表。
4. conf 其他 bootstrap client 的 adminManaged 扩展 — 默认 false，行为不变。
