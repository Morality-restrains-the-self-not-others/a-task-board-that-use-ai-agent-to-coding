# daydaymoney「代码仓库」应打开 gitlab.daydaymoney.com — 设计

- **状态**: shipped（2026-07-11）
  - conf `publicUrl`/`allowedHost` → `https://gitlab.daydaymoney.com`
  - HK nginx `gitlab.daydaymoney.com` → `183.250.1.132:8012`（通配符证书）
  - DNS 由运营指向 HK；前端 rebuild + collectstatic
  - 验收：Navbar `nav-git-service` href = `https://gitlab.daydaymoney.com`；`/users/sign_in` HTTPS 200

- **日期**: 2026-07-11
- **迭代**: gitlab-public-url-daydaymoney

## 问题

在 `https://www.daydaymoney.com/tenant/.../projects/` 点击导航「代码仓库」，打开的是 `http://183.250.1.132:8012/`，期望为 `https://gitlab.daydaymoney.com/`。

## 根因

1. Navbar：`gitServiceUrl = import.meta.env.VITE_GIT_SERVICE_PUBLIC_URL`（构建期注入）。
2. 来源：`conf/infra/git-service/config.yaml` → `publicUrl: http://${subdomains.gitlab}`，经 vue sync 进构建。
3. 当前 `DEPLOY_MODE` 默认 **local**：`${subdomains.gitlab}` → `183.250.1.132:8012`。
4. 生产静态包已烘焙 `http://183.250.1.132:8012`（见 `Navbar.logic-*.js`）。

与 v16「多入口域名」不一致：主站已用 daydaymoney，Git 服务公网 URL 仍停在 IP:8012。

## 架构理解（基线）

- **Current**：v13 application-integration（容器热路径零 Django）。
- **积压 target**：v14–v16（含多入口同源 API）。
- 应用层已有 GitLab / git-service；本次主要是 **边缘公网 URL + conf 公网基址**，不新增业务服务。

## 方案对比

| 方案 | 做法 | 优点 | 缺点 |
|------|------|------|------|
| **A（推荐）** | ① 边缘 DNS+nginx：`gitlab.daydaymoney.com` → GitLab；② conf 将 `git-service.publicUrl`（及 `allowedHost`）改为 `https://gitlab.daydaymoney.com`；③ 重同步 + 前端 rebuild/collectstatic | 与主站域名一致；一处配置驱动 Navbar/文档 | 须确认 TLS/反代；local IP 开发需保留别名或 `website_aliases` |
| B | 仅前端：daydaymoney Origin 时硬编码 `https://gitlab.daydaymoney.com` | 改动面小 | 与 conf SSOT 分叉；OAuth/clone 仍指向 IP |
| C | 全局 `DEPLOY_MODE=domain` + `BASE_DOMAIN=daydaymoney.com` | 子域统一 | 影响面大（api/auth/gateway 等），不适合本迭代单点修复 |

**推荐 A**：conf SSOT + 边缘同构（对齐 `docs/runbooks/add-public-entry-origin.md` 精神，专用于 gitlab 子域）。

### 配置落点（A）

- 源：`conf/infra/git-service/config.yaml`  
  - `publicUrl: https://gitlab.daydaymoney.com`  
  - `allowedHost: https://gitlab.daydaymoney.com`（或保留 http IP 为 alias，视 GitLab external_url 而定）
- 或 local 覆盖：`GITLAB_PUBLIC_URL` / 专用 overlay，避免打断纯 IP 开发机。
- 同步：`conf/frontend/vue/sync.sh` → `git-service.yaml`；`taskAuth.gitServicePublicBase` 等同引用 `${subdomains.gitlab}` 的链需一并评估是否改 HTTPS 域名。
- 前端：`npm run build` + `collectstatic` + 重载 gunicorn。
- 边缘：HK nginx server_name `gitlab.daydaymoney.com`，TLS，反代至 `183.250.1.132:8012`（或容器 GitLab）。

### 非本迭代（可列后续）

- OAuth `authorize_origin` / provider `website` 若仍有 IP 残留，按入口逐步清掉。
- Navbar 入口感知逻辑（方案 B）仅作 conf 未切时的临时兜底。

### 变更记录（2026-07-11 续）

- **问题**: 公网 `https://gitlab.daydaymoney.com` 点 taskAuth SSO 时，OmniAuth 仍发
  `redirect_uri=http://gitlab.daydaymoney.com:8012/...`，taskAuth 白名单为
  `https://gitlab.daydaymoney.com/users/auth/openid_connect/callback` → `redirect_uri not allowed`。
- **修复**: `gitService/docker-compose.yml` 改为 `GITLAB_OIDC_REDIRECT_URI`（由 `run.sh` 从
  `task-auth` `bootstrapClients` 解析，与 SSOT 一致）；`sync_omniauth_oidc.sh` 检测漂移并 reconfigure。

### 变更记录（2026-07-11 external_url HTTPS）

- **问题**: GitLab `external_url` 仍为 `http://gitlab.daydaymoney.com:8012`，页面 clone URL 带端口。
- **修复**: `GITLAB_EXTERNAL_URL` ← conf `publicUrl`/`allowedHost`（`https://gitlab.daydaymoney.com`）；
  Omnibus `external_url` 用该值；`nginx['listen_https']=false` + `listen_port=8012`（边缘终止 TLS）。
- **验收**: `Gitlab.config.gitlab.url` 与 `Project#http_url_to_repo` 不含 `:8012`，scheme 为 https。

## 🐍 Python 新增接口

无。`python_api_approval: n/a`

## 价值流影响

- 触及「代码仓库入口 / Git 服务可达性」；不改登录 Token 模型。
- 测试：Playwright 断言 `nav-git-service` 的 `href` 为 `https://gitlab.daydaymoney.com`（或以其为前缀）。

## 🏛️ 架构变更影响（若批准 A 且含边缘 GitLab 子域）

建议 **v17 application-integration**（仅此视图即可）：

| 变更 | 说明 |
|------|------|
| 🟡 Edge nginx | 新增 `gitlab.daydaymoney.com` → GitLab |
| 🟡 git-service / conf | `publicUrl` 公网基址改为 HTTPS 域名 |
| 伴生 | `.puml` + `.archimate`（Plateau v16→v17 + sourceConnection）+ `.mermaid.md` |

若仅改 conf/前端、边缘已存在且架构图已有「用户→GitLab」边，可降级为 **不建架构版本**（配置微调）。实施前按边缘现状二选一。

## 验收标准

1. daydaymoney 已登录页，「代码仓库」`href` = `https://gitlab.daydaymoney.com`（可带尾斜杠规范化）。
2. 点击后浏览器打开该 HTTPS 源（非 `183.250.1.132:8012`）。
3. `https://gitlab.daydaymoney.com/` 返回 GitLab 登录/首页（TLS 有效）。
4. 本地 IP 入口（若保留）仍可通过 `website_aliases` / 环境覆盖访问 GitLab。
