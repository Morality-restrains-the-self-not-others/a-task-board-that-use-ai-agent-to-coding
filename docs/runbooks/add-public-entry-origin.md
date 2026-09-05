# 新增公网入口（域名 / IP）操作手册

多入口采用**同源反代**：浏览器只访问入口 Host；`/` 前端，`/api/` → task-gateway。

## 成功标准

- 新入口 HTTPS（或 HTTP IP）打开 `/auth/login/`，公共 API 返回 JSON
- OAuth 从该入口发起时，`feb` 与 `redirect_uri` 落在该入口 Origin
- CSRF / CORS / Vite allowedHosts 已含该 Origin

## 步骤清单

### 1. DNS

- 域名 A/AAAA 指向边缘机（如 HK）；裸域与 `www` 按需都配

### 2. TLS（域名）

- 证书放到边缘，例如 `/etc/nginx/ssl/<domain>/fullchain.pem` + `certkey.pem`

### 3. 同构 nginx

**前端入口**（`www` / 裸域）复制同源模式：

```nginx
# /api/ → http://<APP_HOST>:18081
# /     → 生产：http://<APP_HOST>:4000（taskFE Vite preview，纯 Vite 无 Django）
#         开发：http://<APP_HOST>:4000（Vite）
# 转发 Host、X-Forwarded-Proto、X-Forwarded-For、X-Real-IP
```

**daydaymoney 全量子域**（与 `conf/base.yaml` 对齐）以仓库示例为准：

- 示例：`task2app/scripts/daydaymoney.hk.nginx.example`
- 映射摘要：`api`→`:18081`，`gitlab`→`:8012`，`auth_api`→`:8003`，`gitoauth_api`→`:8002`，`provider`→`:8010`，`agentsupport_api`→`:8011`，`cloud_api`→`:8018`，…（与 `conf/base.yaml` `subdomains.*` 一致；勿用遗留 `*.api.*`）
- 注意：`*.example.com` 通配符**不覆盖** `*.api.daydaymoney.com`；多级子域需单独 LE 证书（示例文件头注释有 `certbot` 命令）

`nginx -t && systemctl reload nginx`

生产前端：在 `taskFE/app` 执行 `npm run build`（Vite build，写出 `taskFE/app/dist/`），runAll 的 taskFE 服务以 `npm run preview`（:4000）直接服务 dist 产物；`/static/assets/*` 由 Vite preview 中间件映射到 `dist/`。Django 已退役，`collectstatic` 为历史操作，不再需要，也不得在 build 链上挂回。

> **构建后确认**：公网 `/static/assets/main-<hash>.js` 返回 **200** / `javascript`（见 `.ai/09_failure_experience/02_runtime_errors/04_public_spa_static_js_404_after_vite_build.md` 的定位方法）；taskFE 健康路径 `/health` 返回 200。

### 4. conf 一行白名单

编辑 `conf/frontend/vue/config.yaml` 的 `publicEntryOrigins`，追加：

```yaml
- https://www.daydaymoney.com
- https://example.com
```

同步 gateway CORS（`conf/gateway/task-gateway/config.yaml`）与 taskAuth `oidc.postLogoutRedirectOrigins`（若使用 SLO）。

然后：

```bash
# 再生 APISIX
cd taskGateway && TASK_GATEWAY_APISIX_IN_DOCKER=1 python3 scripts/routes-to-apisix.py
docker compose restart apisix   # 或等价重载

# 重载 gitOauth / taskAuth 等以加载 conf（Django 已退役）
```

### 5. OAuth / OIDC 提供商控制台

在 **GitLab / GitHub / 其它** 应用中，为每个入口追加回调（与运行时选择一致）：

```text
https://www.daydaymoney.com/api/accounts/<service_provider>/oauth/callback/
```

本地 GitLab 可用脚本或 Admin → Applications 维护多回调。

OIDC RP（GitLab SSO、ai-provider）的 `redirectUri` 一般仍指向专用子域；主站入口主要影响 `post_logout_redirect_uri` 白名单。

### 6. 验收

```bash
curl -sI https://www.daydaymoney.com/auth/login/
curl -s https://www.daydaymoney.com/api/public/system-feature-policy/   # application/json
```

浏览器打开登录页：无 Mixed Content；OAuth 回跳落在同一入口域名。

## 参考

- 设计：`docs/superpowers/specs/2026-07-11-multi-entry-same-origin-api-design.md`
- 架构：`docs/architecture/v16-application-integration-20260711-1726-claude.*`
