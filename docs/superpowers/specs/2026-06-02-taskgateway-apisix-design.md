# taskGateway（APISIX）统一 API 网关设计

> 日期：2026-06-02  
> 状态：**Redesign 已实现（2026-06-02）— auth_mode + profile + Docker + 证书**  
> 关联：`conf/` 端口体系、`taskContainerGateway/`（容器出站专用，非本网关）、`docs/superpowers/specs/2026-06-02-port-config-split-monorepo-conf-design.md`

---

## 1. 背景与目标

### 1.1 现状（As-Is）

| 层级 | 行为 |
|------|------|
| 浏览器 | 开发态访问 `vue :4000`；`apiFetch` 经 `VITE_API_BASE_URL` 或同源 `/api` |
| Vite dev proxy | 大部分 `/api` → `django :8001`；SSE 特例 → `taskSSE :8798`；`container-layer-git-commit` → `taskContainerGateway :8014` |
| 生产/配置 | `conf/vue/config.yaml` 的 `apiBaseUrl` 直连 `http://172.20.10.3:8001` |
| 后端 | `task-auth :8003`、`git-oauth :8002` 等由 **Django 内部 HTTP 桥接** 调用，浏览器不直连 |
| 鉴权 | `Authorization: Token …` 在各服务（主要为 Django DRF）各自校验 |
| 可观测 | 前端 `X-Trace-Id`；各服务分散日志；Vite 对部分路由有 proxy access log |

**痛点：**

- 入口分散（Vite 代理规则、直连 8001、`apiBaseUrl`），难以统一 CORS、限流、鉴权策略。
- 浏览器无法按路径直达 `task-auth` / `git-oauth`，与「身份在 taskAuth」架构不一致。
- 新增微服务需改 Vite proxy + 前端 base URL，易漏改。
- `taskContainerGateway` 职责清晰（容器 outbound），**不能**扩成通用 API 网关。

### 1.2 用户确认决策（2026-06-02 头脑风暴）

| 项 | 选择 |
|----|------|
| 范围 | **完整 API 平台**：统一入口、路径转发、网关鉴权、限流、CORS、可观测、配置化路由 |
| 部署 | **独立目录 `taskGateway/`**（自有 `run.sh` + `conf/task-gateway/`，不强制进 dockerInfra） |
| 前端接入 | **直连网关**：`apiBaseUrl` / `VITE_API_BASE_URL` 指向 taskGateway；Vite **不再**代理 `/api` |
| git-oauth 路由 | **尽可能** upstream 指向 `git-oauth:8002`；浏览器对外 URL 仍为网关 TLS 入口 |
| TLS | **本地终止 HTTPS**（对外 `https://{host}:{tlsPort}`） |
| Django 鉴权 | **G3 同步**：`CustomTokenAuthentication` 信任网关头 `X-User-Id`（见 §6.3） |

### 1.3 SMART 目标

1. 浏览器与 E2E **仅** 对 taskGateway 发起 API 请求（静态资源仍由 Vue 提供）。
2. 按路径前缀将请求转发到 `saas-backend`、`task-auth`、`git-oauth`、`task-sse`、`task-container-gateway`、`ai-provider` 等 upstream。
3. 网关层完成 **Token 校验**（委托 taskAuth `token/resolve`），向下游注入 `X-User-Id` 等可信头；公开路由白名单。
4. 统一 **CORS**、**限流**、**请求 ID / trace 透传**、**访问日志**。
5. 路由与插件由 **版本化配置**（YAML → APISIX Admin API / ADC）生成，纳入 runAll 与健康检查。
6. 与现有 `conf/<app>/` 端口体系对齐，新增 `conf/task-gateway/`。

---

## 2. 价值流影响（Value Stream Impact）

| 问题 | 评估 |
|------|------|
| 受影响 stream | **`user-auth`**（login/profile/guard 入口改为经网关）、**`frontend-auth-guard-redirect`**、**`task2app-outbound-governance`**（出站仍由 Django 发起，不变）、**`task-container-gateway`**（upstream 经网关暴露）、**`runall-lifecycle` / vue runtime** |
| 新 stream | 建议新增 **`task-gateway`** domain：`edge-routing`、`gateway-auth-forward`、`gateway-rate-limit`、`gateway-observability` |
| 字段变更 | `taskFE.config.apiBaseUrl` → 指向 `task-gateway`；新增 `task-gateway.routes.*`、`task-gateway.plugins.cors_origins` 等（三步命名见 `value-stream.yaml.ai.md`） |
| 测试影响 | Playwright / Vitest 的 API base URL；`accounts/view_test` 若直连 Django 可保留（pytest 不经浏览器）；新增网关路由集成测 |
| 交叉依赖 | runAll：`task-gateway` 需在 `taskFE` 之前 healthy；`saas-backend` / `task-auth` 为 upstream |

---

## 3. 领域概念清单（供 `/5-ddd`）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | **Edge Gateway（taskGateway）**、Identity（taskAuth）、SaaS API（django）、Git OAuth、Task SSE、Container Outbound |
| **实体** | `Route`（uri + methods + upstream）、`Upstream`（host:port + health）、`PluginPolicy`（auth/cors/limit/trace）、`PublicRoute`（匿名白名单） |
| **聚合** | `GatewayRouteTable`（路由表根，一致性：全表版本一起发布） |
| **领域事件** | `GatewayConfigPublished`（可选，用于审计；V1 可仅 Git 版本化） |

---

## 4. 方案对比

| 方案 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **A. APISIX standalone（推荐）** | `taskGateway/` 包装 APISIX + 路由生成器 | 插件生态（auth、cors、limit、otel）、Admin API、社区成熟 | 多一个进程/端口；团队需熟悉 APISIX 配置 |
| **B. 扩展 Go taskContainerGateway** | 在现有 Go 服务上加通用 reverse proxy | 单语言、已有 tracelog | 与容器出站耦合；自研鉴权/限流成本高；违背「APISIX」诉求 |
| **C. Nginx + lua** | OpenResty 自写 | 轻量 | 完整平台能力需大量 lua；可维护性不如 A |

**推荐 A**：与用户指定 APISIX + standalone 目录一致；`taskContainerGateway` 保持专用，经 taskGateway 暴露同一路径前缀即可。

---

## 5. 目标架构（To-Be）

```text
Browser / Playwright
    │  https://<host>:8443/api/...   （本地 TLS 终止）
    ▼
┌─────────────────────────────────────┐
│  taskGateway (APISIX data plane)     │
│  :8443 HTTPS（对外）                  │
│  :8080 HTTP（可选，仅本机调试）       │
│  Plugins: cors, limit-req,           │
│           forward-auth (→ taskAuth),  │
│           prometheus, request-id      │
└──────────┬────────────────────────────┘
           │ upstream (uri 路由)
     ┌─────┼─────┬─────────┬──────────┬────────────┐
     ▼     ▼     ▼         ▼          ▼            ▼
  django task-auth git-oauth task-sse  t-c-gateway  ai-provider
  :8001   :8003    :8002     :8798      :8014        :8010
```

**与 Vue 的关系：**

- `:4000` 仅静态资源 + HMR + `/health`（前端自身）。
- `conf/vue/config.yaml`：`apiBaseUrl: https://172.20.10.3:8443`（示例；与网关 TLS 端口一致）。
- 删除 `vite.config.js` 中 `/api`、`/logout`、`/accounts` 到 Django 的 proxy（保留 `/health` 等前端自有路径）。

---

## 6. 组件设计

### 6.1 仓库布局

```text
taskGateway/
  run.sh              # start|stop|reload|routes-apply
  build.sh            # 可选：拉 APISIX docker 镜像或校验二进制
  README.md
  routes/
    routes.yaml       # 声明式路由源（人工维护）
    generated/        # scripts 生成的 apisix 配置（可提交或 CI 生成）
  scripts/
    routes-to-apisix.py   # YAML → Admin API / ADC
    health-check.sh
conf/task-gateway/
  config.yaml         # host, port, adminPort, cors, rateLimit, auth
  django.yaml         # GENERATED upstream 摘要
  task-auth.yaml
  vue.yaml            # GENERATED apiBaseUrl 指向网关
  sync.manifest.yaml
  sync.sh
```

**运行方式（V1 建议）：**

- **Docker Compose** `apache/apisix`（`taskGateway/docker-compose.yml`）：
  - 数据面 **HTTPS `:8443`**（本地 dev 证书，见 §6.7）
  - 可选 HTTP `:8080`（仅本机 curl / 健康探测）
  - Admin `9180` bind `127.0.0.1`
- `run.sh` 封装 compose + `routes-apply` + 证书检查。

### 6.2 路由表（核心规则）

优先级：**更具体的路径优先**。`routes.yaml` 使用 **`auth_mode`**（Redesign 2026-06-02，替代 `public`/`protected` 二元）。

| auth_mode | APISIX 行为 | 典型路由 |
|-----------|-------------|----------|
| `deny` | 403，不转发 | `/api/internal/*` |
| `none` | 无 forward-auth | login、OAuth **callback**、webhook、health |
| `app-jwt` | 无 forward-auth；**upstream 应用**校验 `?token=` | OAuth **start** |
| `token` | forward-auth（`Authorization`）+ 剥离客户端 `X-User-Id` | 业务 API、django profile、app/start |

**已确认：浏览器可见 OAuth 尽可能 upstream → `git-oauth:8002`**；**OAuth callback 保持 `none`**（IdP 回调无 Bearer，由 gitOauth state/JWT 校验）。

| 优先级 | URI 前缀 / 正则 | Upstream | auth_mode | 备注 |
|--------|-----------------|----------|-----------|------|
| 1 | `…/server-startup-status-sse` | task-sse | `token` | 长连接超时加大 |
| 2 | `…/container-layer-git-commit` | task-container-gateway | `token` | |
| 3 | `/api/accounts/{github,gitlab}/oauth/start/` | git-oauth | **`app-jwt`** | 仅 `?token=`，禁止 forward-auth |
| 4–5 | `…/oauth/callback/` | git-oauth | `none` | |
| 6–7 | task-auth 注册/登录/重置 | task-auth | `none` | |
| 8 | `/api/internal/*` | — | `deny` | |
| 9 | `GET /api/accounts/users/*`（单段） | task-auth | `token` | 不含 `users/profile/` |
| 10–13 | django 业务（含 profile、app/start） | django | `token` | profile 见 §6.8 |
| 14 | `/*` 默认 | django | `token` | |
| 15–16 | webhook 等 | django | `none` | |

**git-oauth 对外 URL 约定（重要）：**

- `GithubAppAuthorizeStartView` 等生成的 `authorize_url` 必须使用 **`TASK_GATEWAY_PUBLIC_BASE`**（如 `https://172.20.10.3:8443`），路径为 `/api/accounts/github/oauth/start/`，保证浏览器只打网关。
- OAuth 应用在 IdP 注册的 `redirect_uri` 同步改为网关 HTTPS 地址（如 `https://…:8443/api/accounts/gitlab/oauth/callback/`），与 `conf/git-oauth/providers/*.yaml` 一致。
- git-oauth 服务间 internal 路径（`/api/internal/github/oauth/*`）**不**经网关暴露。

**原则：**

- 凡 taskAuth / gitOauth 已实现的**浏览器路径**，网关 upstream 直达对应服务；Django 仅保留 catalog、JWT 签发、connection 聚合等尚未下沉的端点。
- 未迁移路径仍走 Django，避免双实现。

### 6.3 网关鉴权（仅 `auth_mode: token`）

**`app-jwt` / `none` 不得挂载 forward-auth**（审查发现：对 OAuth start 使用 `protected` 会导致 401）。

```text
Client Request + Authorization: Token <key>
    → APISIX forward-auth（仅 auth_mode=token）
    → POST task-auth /api/internal/token/resolve/  (X-TaskAuth-Internal-Secret)
    ← 200 { user_id, is_active, is_superuser }
    → APISIX 注入 X-User-Id, X-Auth-Superuser, X-Gateway-Auth-Verified 等到 upstream
    → upstream 处理（Django **G3 同步**信任网关头，见下）
```

| 项 | 说明 |
|----|------|
| 无网关 Token 校验 | `auth_mode: none` 与 `app-jwt`；维护于 `routes.yaml` |
| 客户端头剥离 | `token` 路由在 forward-auth 前 `request-transformer` 移除客户端 `X-User-Id`、`X-Gateway-Auth-Verified` |
| 失败语义 | resolve 401/403 → 网关直接 401，**不**转发到 upstream（fail closed） |
| taskAuth 不可用 | 503 + `Retry-After`；与 `identity-service-unavailable-503` 价值流一致 |
| **Django G3（已确认）** | **`CustomTokenAuthentication` 同步改造**：在受信请求上优先读 `X-User-Id`，避免二次 `resolve_token` |

**Django `CustomTokenAuthentication` To-Be（G3 交付）：**

```text
若 request 带齐：
  X-Gateway-Auth-Verified: 1
  X-TaskGateway-Internal-Secret: <与 conf 一致>
  X-User-Id: <snowflake>
则：
  直接构造 AuthPrincipal（可选再 GET taskAuth 校验 is_active，或信任网关头 X-Auth-Active）
否则：
  沿用 load_principal_from_token(Authorization: Token …)
```

| 防伪造 | 仅当 `X-TaskGateway-Internal-Secret` 匹配且 `X-Gateway-Auth-Verified=1` 时信任 `X-User-Id`；直连 Django `:8001` 绕过网关时无此头，仍走 Token resolve |
| 设置项 | `TASK_GATEWAY_TRUST_HEADERS=True`（runAll 默认 true）；pytest 直连 Django 保持 false |
| git-oauth | 同为 Django 栈时可复用相同 Authentication 类；纯 gitOauth 视图在 G2 评估是否引入等价中间件 |

### 6.4 横切插件

| 能力 | APISIX 插件 | 配置来源 |
|------|-------------|----------|
| CORS | `cors` | `conf/task-gateway/config.yaml`：`allowedOrigins` 含 `vue` origin |
| 限流 | `limit-req` / `limit-count` | 全局默认 + webhook 宽松 + login 防暴力 |
| Trace | `request-id` 或 `opentelemetry` | 透传/生成 `X-Trace-Id`，与前端 `apiFetch` 对齐 |
| 访问日志 | `http-logger` 或 `file-logger` | JSON 行：`route_id`, `upstream`, `status`, `latency`, `trace_id` |
| 健康 | 路由 `GET /api/health/` 聚合或单独 `gateway-health` | runAll 探活 `http://host:8080/api/health/`（可代理 django 汇总） |

### 6.5 配置与 conf 集成

- 新增 `conf/task-gateway/config.yaml`：

```yaml
host: 172.20.10.3
tlsPort: 8443       # 对外 HTTPS（浏览器 apiBaseUrl）
httpPort: 8080      # 可选，本机调试
publicBase: https://172.20.10.3:8443
adminPort: 9180
enabled: true
tls:
  certFile: taskGateway/certs/dev-gateway.pem
  keyFile: taskGateway/certs/dev-gateway-key.pem
  # 开发用自签；run.sh 首次启动可 mkcert 生成
cors:
  allowedOrigins:
    - http://172.20.10.3:4000
    - https://172.20.10.3:4000
  allowCredentials: true
rateLimit:
  globalPerMinute: 3000
  loginPerMinute: 30
auth:
  forwardAuthUrl: http://172.20.10.3:8003/api/internal/token/resolve/
  internalSecret: <from conf/task-auth sync>
  gatewayInternalSecret: <task-gateway 注入下游>
upstreams:
  django: { host: 172.20.10.3, port: 8001 }
  taskAuth: { host: 172.20.10.3, port: 8003 }
  gitOauth: { host: 172.20.10.3, port: 8002 }
  # ...
```

- `scripts/conf-sync.py` 扩展：从各 app `config.yaml` 生成 `conf/task-gateway/*.yaml` upstream 碎片（**GENERATED，禁止手改**）。
- `conf/vue/sync.manifest.yaml`：`apiBaseUrl` ← `task-gateway.publicBase`（**https**）。
- `conf/git-oauth/providers/*.yaml`：`redirect_uri` / `service.allowedHost` 碎片 sync 自 `task-gateway.publicBase`（G2 路由审计时批量更新）。

### 6.8 Profile API 与 login_method（Redesign 纳入）

**问题：** `GET /api/accounts/users/profile/` 经网关 → django upstream，`_build_profile_payload` 仍 `LoginMethod.objects` → saas 库无 `accounts_login_method` 表 → 500。

**To-Be：**

- `email` / `has_phone` / `phone_masked` 一律经 `login_methods_resolver.get_identifier_for_method(user_id, 'email'|'phone')`（taskAuth HTTP）。
- **不**在 profile 路径调用 `LoginMethod.get_*_for_user()`。
- pytest：`test_profile_payload_resolves_login_methods_via_taskauth`（mock resolver 或 `TASKAUTH_ENABLED` 假服务）。

与网关正交：即使未启 taskGateway，直连 django 的 profile 亦须修复。

### 6.9 Docker 容器访问 upstream（Redesign 纳入）

| 项 | 决策 |
|----|------|
| 问题 | APISIX 在容器内访问 `172.20.10.3:800x` 在 Mac bridge 网络常不可达 |
| 方案 | `routes-to-apisix.py` 生成 upstream 时，若 `TASK_GATEWAY_APISIX_IN_DOCKER=1`（`run.sh routes-apply` 默认设置），将 host 换为 `conf.task-gateway.docker.upstreamHost`（默认 `host.docker.internal`） |
| 备选 | Linux runAll 可设 `network_mode: host`（文档说明，非默认） |
| 验证 | `run.sh start` 后 `curl -k https://…:8443/api/health/` 来自容器路径 |

### 6.10 开发 TLS 证书（Redesign 纳入）

| 项 | 决策 |
|----|------|
| 生成 | `run.sh setup-tls` → `taskGateway/certs/dev-gateway.pem` + `dev-gateway-key.pem` |
| Git | `taskGateway/certs/.gitignore` 忽略 `*.pem`；提交 `certs/README.md` + 可选 `*.pem.example` 占位说明 |
| 禁止 | 私钥不得进入版本库 |

### 6.7 本地 TLS 终止（已确认）

| 项 | 说明 |
|----|------|
| 终止点 | APISIX `ssl` listener `:8443` |
| 证书 | `taskGateway/certs/` 开发自签或 **mkcert**（`run.sh setup-tls`）；不入库私钥，提供 `.example` + README |
| 浏览器 | Vue `:4000`（HTTP）→ `fetch('https://…:8443/api/…')`；需在 dev 信任 CA 或点击继续 |
| 上游 | gateway → upstream 仍用 **HTTP**（`http://172.20.10.3:800x`），TLS 仅在边缘 |
| runAll health | `https://172.20.10.3:8443/api/health/`（`-k` 仅脚本内） |
| E2E | Playwright `ignoreHTTPSErrors: true` 或导入 mkcert CA |

### 6.6 runAll 编排

```yaml
# runAll.yaml 片段（示意）
- name: task-gateway
  working_dir: ../taskGateway
  start_command: "bash run.sh start"
  stop_command: "bash run.sh stop"
  depends_on: [task-auth, saas-backend]
  health_check:
    url: "https://172.20.10.3:8443/api/health/"
    timeout: 60

- name: taskFE
  depends_on: [task-gateway, saas-backend, git-oauth]  # OAuth 路由依赖 git-oauth upstream
```

顺序：`task-auth` → `git-oauth` → `saas-backend` → **`task-gateway`**（routes-apply + TLS）→ `taskFE`。

---

## 7. 前端与客户端变更

| 变更 | 说明 |
|------|------|
| `conf/vue/config.yaml` | `apiBaseUrl` → taskGateway |
| `vite.config.js` | 移除 `/api`、`/logout`、`/accounts` proxy；SSE/container 特例删除（改走网关） |
| `apiFetch` / `sessionUserIdUtils` | 无需改路径，仅 base URL 变化 |
| Cookie / CORS | 网关统一 `Access-Control-Allow-Origin`；`credentials: include` 与 allowlist 一致 |
| E2E Playwright | `baseURL` 仍为 vue；API 请求用 `apiBaseUrl` → `https://…:8443`（`ignoreHTTPSErrors`） |
| `GITHUB_APP` / OAuth redirect | `redirect_uri`、authorize_url 统一为网关 `publicBase` |

---

## 8. 安全与运维

| 风险 | 缓解 |
|------|------|
| `/api/internal/*` 误暴露 | 网关默认 deny；仅 mesh 内网 DNS 直达 upstream |
| Admin API 9180 暴露 | 仅 bind 127.0.0.1；`routes-apply` 经本地脚本 |
| 双重 CORS | 关闭 Django `CORS` 对浏览器源（或收窄），以网关为准 |
| Webhook 伪造 | 匿名路由 + 限流 + 后续 mTLS（V2） |
| 配置漂移 | `routes.yaml` PR 评审 + CI `routes-to-apisix.py --check` |

---

## 9. 分期交付（在「完整平台」范围内的实施顺序）

| 阶段 | 交付物 | 验收 |
|------|--------|------|
| **G1** | taskGateway 骨架 + docker + TLS `:8443` + runAll + 默认 `/* → django` | `curl -k https://:8443/api/health/` 200 |
| **G2** | 路由拆分 task-auth / git-oauth OAuth / task-sse / t-c-gateway；更新 `authorize_url` / redirect_uri | OAuth start/callback 经网关到 git-oauth upstream |
| **G3** | forward-auth + 白名单 + **CustomTokenAuthentication 信任 X-User-Id** | 无 token profile → 401；带网关头无二次 resolve |
| **G4** | cors + limit + access log + trace | 前端 :4000 调 :8080 无 CORS 错误；日志含 trace_id |
| **G5** | vue/vite/conf-sync 切换直连；删除 Vite API proxy | runAll 全栈绿；Playwright auth 套件绿 |
| **G6** | 文档 + value-stream `task-gateway` 步骤 + CI 路由校验 | 门禁脚本纳入 pre-commit |

---

## 10. 测试策略

| 层级 | 内容 |
|------|------|
| 单元 | `routes-to-apisix.py` 快照测试（YAML → JSON） |
| 集成 | bash + curl：公开路由、受保护路由、internal 拒绝 |
| 回归 | 现有 `user-auth` view_test（直连 Django 不变）；新增 `tests/test_taskgateway_routes_integration.py` 经 :8080 |
| E2E | Playwright 改 `API_BASE`；`AuthResetPassword`、`GitSiteOAuth` 全链路 |

---

## 11. 非目标（V1 不做）

- 用 taskGateway **替代** Django 到 taskAuth 的 **服务端** `forward_to_taskauth`（服务端出站仍走 internal URL）。
- 在网关终止 **JWT 登录**（`2026-05-31-taskauth-jwt-migration-design` 可后续切换 `jwt-auth` 插件）。
- 替换 `taskContainerGateway` 实现（仅改变暴露方式）。
- 生产 K8s Ingress 与 APISIX 集群 HA（本设计聚焦 monorepo 本地/runAll）。

---

## 12. 已确认决策（2026-06-02 补充）

| # | 问题 | 决策 |
|---|------|------|
| 1 | git-oauth 路径 | **尽可能** upstream → `git-oauth:8002`（start/callback/动态 sp）；catalog、app/start、connection 暂留 django；对外 URL 统一网关 HTTPS |
| 2 | HTTPS | **本地 APISIX 终止 TLS**，对外 `:8443`；upstream 仍 HTTP |
| 3 | Django 网关头 | **G3 同步**改 `CustomTokenAuthentication`：校验 `X-TaskGateway-Internal-Secret` + `X-Gateway-Auth-Verified` 后读 `X-User-Id` |

---

## 13. 完成标准（Definition of Done）

- [ ] runAll 启动 `task-gateway`（HTTPS），vue `apiBaseUrl` 指向 `https://…:8443`，**无需** Vite `/api` proxy 即可登录、profile、Git OAuth（start/callback 经网关至 git-oauth）。
- [ ] G3：`CustomTokenAuthentication` 在网关头存在时不二次 `resolve_token`。
- [ ] `routes.yaml` 覆盖 value-stream `user-auth` 全部公开/受保护路径。
- [ ] 网关访问日志可关联 `X-Trace-Id`。
- [ ] `verify_auth_tables_dropped` / saas 无 `accounts_login_method` 场景下，profile 经 django upstream 仍正常（依赖 Inc-6 profile 修复项）。
- [ ] 文档与 `conf/task-gateway/`、`value-stream.yaml` 新步骤已更新。

---

## 14. 变更记录

- 2026-06-02：初稿 — 用户选择完整平台 + standalone `taskGateway/` + 浏览器直连网关
- 2026-06-02：确认 git-oauth 尽可能直连 upstream、本地 TLS :8443、G3 Django 信任 `X-User-Id`
- 2026-06-02：**Redesign 扩 scope** — `auth_mode` 三分；profile HTTP-only；Docker `host.docker.internal`；certs gitignore
