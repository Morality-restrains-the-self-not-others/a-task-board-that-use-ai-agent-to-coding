# taskFE 公网 SPA 改为 nginx 静态常驻

- **日期**: 2026-08-20
- **作者**: cursor
- **状态**: 🎯 accepted（Docker nginx 方案已批准并实施；待 /10-ship 升 current）
- **关联**: ADR-0022（accepted）；意图 `taskfe_nginx_static_resident`
- **python_api_approval**: n/a（零新增 Python 接口）
- **触发**: `/user/:id/profile/` 等 SPA 路径 APISIX 502；根因是 `vite preview` 进程被 stop-all / kill，不是 `dist/` 非原子替换

## 问题

公网 HTML 壳路径（`/`、`/login/`、`/user/:id/profile/`）经 APISIX `spa-catch-all` 转到 `host.docker.internal:4000`。当前 runAll `start` 跑的是 **Node `vite preview`**。该进程一旦被 `stop-all` / `pkill` / `kill -9 :4000`，网关立刻 **connection refused → 502**，磁盘上的 `dist/` 再完整也无人 serve。

原子构建（`dist.next` → `dist`）只保护「preview 还活着时不要掏空目录」。它不能让「没有监听进程」变成 200。

`taskGateway/routes/routes.yaml` 注释已写「生产：Nginx 静态文件容器」，与实现（vite preview :4000）不一致。

## 当前架构理解（基线，不改拓扑入口）

根据 current 架构设计稿：

- 共有 2 个 current 视图：`enterprise-landscape` v85、`application-integration` v85
- 业务层：Developer / Identity / Task / Billing 等
- 应用层：APISIX :18081、taskFE Vue SPA、taskAuth 等 Go 服务
- 技术层：应用主机、SH 边缘 nginx（公网 TLS，**不是** SPA 静态根）、GitLab、Kafka/Redis
- 上次 shipped 版本 **v85**；**v86** 仍为正交 target（PIPL 账号注销），本次不合并、不覆盖

本次需求：只替换 taskFE **怎么把 dist 变成 HTTP**，不改 APISIX 路由、不改 Vue 路由。

📋 架构版本历史：v85 ✅ current — 多区域 gitService；v86 🎯 target — 个人账号注销（积压）。本次将在 **v85 current** 上开 **v87**（批准后落盘）。

## 决策锁定

We will：公网 / runAll 托管的 taskFE **用 Docker `nginx` 容器常驻**，把宿主机 `0.0.0.0:4000` 映射到容器 `:80`；**Vite 只负责 build**。APISIX `spa-catch-all` → `up-taskFE` `host.docker.internal:4000` **不变**。

| 项 | 锁定 |
|----|------|
| 进程 | `taskFE/docker-compose.yml` 服务 `taskfe-nginx`；`restart: unless-stopped`；官方 `nginx` 镜像（tag 在 conf） |
| 端口 | conf `port: 4000` + `host: 0.0.0.0` → compose `ports: "${host}:${port}:80"`（fallback 与 conf 同值并注释 SSOT） |
| 卷 | **bind-mount 父目录** `taskFE/app/public`（内含 `releases/` + symlink `html`）。禁止只 mount 已解析的 `html` 目标，否则切 symlink 容器看不见 |
| 发布 | blue-green：`public/releases/<id>/` + 原子 symlink `public/html` → 当前 release |
| 构建 | 宿主机切 symlink；**禁止** `docker stop` / recreate |
| 精准编译重启 | taskFE：**build + 切 symlink**（内容切换不必 reload）；仅 conf 变更时 `nginx -s reload` |
| 缺失 hashed 资源 | **真 404**，禁止 SPA fallback 成 `index.html` |
| 本机 HMR | `npm run dev` 仍用 Vite；与容器占 4000 互斥 |
| 宿主机 apt nginx | 否（本机无 nginx 命令；不装系统包） |
| SH 边缘 nginx | 否（只反代到本机网关） |

### 为何必须 mount 父目录

Docker 若 bind-mount **symlink 本身**，daemon 在 `start` 时解析目标，之后宿主机 `ln -sfn` **不会**进入容器。必须 mount 含 symlink 的父目录，让容器内 `html` 仍是 symlink，内核按请求跟随新 target。

## 请求路径（变更后）

```
浏览器
  → SH 边缘 nginx :443
  → APISIX spa-catch-all GET/HEAD /*
  → host.docker.internal:4000   # 宿主机 docker-proxy，APISIX 不变
  → taskfe-nginx 容器 :80（unless-stopped）
       /health                     → 200/503 JSON（index.html 是否存在）
       /static/assets/<file>       → 文件或真 404；Cache-Control immutable
       其它（含 /user/:id/profile/）→ try_files 文件，否则 index.html（no-cache）
```

## 发布与原子性

当前 `mv dist dist.prev && mv dist.next dist` 在两步之间 **没有名为 `dist` 的目录**。nginx 若 `root` 直接指 `dist/`，该窗口会 404。改为：

1. `vite build --outDir ../public/releases/<id>`（或 staging 再 `mv` 进该目录）
2. 校验 `public/releases/<id>/index.html`
3. 在 `public/` 内 `ln -sfn releases/<id> html.tmp && mv -T html.tmp html`
4. 容器 `root` = 挂载点下的 `html`（跟随 symlink；**不必** recreate）
5. 保留上一 release ≥ 1 个（懒加载 chunk 宽限）

构建失败：不切 symlink，nginx 继续旧 release。

## runAll 生命周期

| 命令 | 行为 |
|------|------|
| `start` | 读 conf → 渲染 nginx.conf → `docker compose up -d`。已有 healthy 容器则跳过。无 `public/html/index.html` → 失败 |
| `stop` | `docker compose stop`（graceful）。禁止宿主机对 :4000 `kill -9` |
| `build` | 只出新 release + 切 symlink；**不 stop / 不 recreate 容器** |
| 精准重启登记 `taskFE` | build + 切 symlink；upstream :4000 不掉 |
| `restart-all` / `stop-all` | 可停容器，但 `up -d` 是秒级（镜像已在本地时） |

`runall-lifecycle.sh stop` 今日的 `pkill vite` / `lsof -ti:4000 | kill -9` **删除**。

## 配置 SSOT

只改 `conf/frontend/vue/config.yaml`（人工可调）。拟增：

```yaml
# SSOT: conf/frontend/vue/config.yaml
serve: nginx          # nginx | vite-preview（仅应急）
releaseKeep: 2
nginxImage: nginx:1.27-alpine   # compose 只消费
memLimit: 128m
```

compose 的 `image` / `mem_limit` / `ports` **只消费** conf（`${VAR:-fallback}` 须与 conf 同值并注释 `SSOT: conf/frontend/vue/config.yaml`）。生成的 `nginx.conf` bind-mount 进容器。

Access log 须带 `X-Trace-Id`（APISIX 已注入），对齐 value-stream `increment2-taskFE-proxy-access-log`。

## 缓存与 SPA fallback（强制）

| location | 行为 |
|----------|------|
| `= /health` | JSON；无 index → 503 |
| `^~ /static/assets/` | alias 到 release；**不** fallback HTML；hashed 文件 `immutable` |
| `/` | `try_files $uri $uri/ /index.html`；HTML `Cache-Control: no-cache` |

`base: '/static/assets/'`（生产 Vite）保持不变。

## 本机开发

- 公网验收 / runAll：**Docker nginx :4000**
- HMR：`npm run dev`；须先 stop 容器释放 4000
- `npm run preview` 可留作对照，**不再**作为 runAll start

## 替代方案（拒绝）

| 方案 | 拒绝原因 |
|------|----------|
| 继续 vite preview + 缩短 stop 窗口 | 进程模型仍会 502；stop-all 必然杀 Node |
| 宿主机 apt `nginx -p` | 用户选定 Docker；本机当前无 nginx 命令，也不把 SPA 绑进系统默认站 |
| 只 bind-mount symlink 目标目录 | 切 `html` 后容器仍读旧 inode，发版不生效 |
| 让 SH 边缘 nginx 直接读 dist | dist 在应用主机，边缘机无文件 |
| rsync 覆盖同一 `dist/` | index.html 与 hashed chunk 更新顺序会短暂错配 |

## 业务意图 → 事件对照

**无对应事件**：静态托管切换，不改变业务事实、无跨聚合副作用。

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 公网 SPA 由 nginx 常驻提供 | — | — | — | 基础设施 serving，无领域状态变更 |

## 领域概念（轻量，供 /6-ddd 可跳过）

- Bounded context: 平台交付 / 前端托管（非租户业务）
- 无新聚合；无新 Kafka topic

## 价值流影响

- `user-auth.frontend-auth-guard-redirect`：SPA 壳可用性（502 时整条流不可达）
- `increment2-taskFE-proxy-access-log`：access log 从 Vite stdout 迁 nginx，须保留 `trace_id`
- 无新 `<service>.<table>.<column>`；无 Python 接口

不新增 value-stream 条目；step 4 若要切片，建议 I1 nginx 常驻 + 构建不杀进程，I2 精准重启改为 reload。

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `CRG unavailable: 仓库根无 .code-review-graph/graph.db` |
| 关键发现 | — |
| 决策影响 | 爆炸半径：`taskFE/docker-compose.yml`（新建）、`runall-lifecycle.sh`、`atomic-vite-build.sh`、`vite.config.js` preview 中间件、`conf/frontend/vue/config.yaml`、runAll precise-restart 对 taskFE 的 stop/start |
| skip 理由 | `unavailable` — 缺图 |

## 🏛️ 架构变更影响（批准后写入，本次尚未落盘）

- **迭代版本**: v87 🎯 target（**不**改 v86 PIPL 积压文件）
- **迭代名称**: taskFE nginx 静态常驻
- **将新增**（每视图四类，缺一不可）:
  - `docs/architecture/v87-enterprise-landscape-<ts>-cursor.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`
  - `docs/architecture/v87-application-integration-<ts>-cursor.puml` + 同上伴生
- **变更明细（拟）**:
  - 🟢 [NEW] taskfe-nginx Docker 容器（:4000→:80，bind-mount `public/`）
  - 🟡 [MODIFIED] taskFE — 公网 serving 从 vite preview 改为容器 nginx + `html` symlink
  - 🔴 [DEPRECATED] runAll 路径上的 `vite preview` 作为公网入口
  - 无标记 — APISIX spa-catch-all / SH 边缘 nginx / APISIX 容器

### .archimate 要点（批准后）

| 文件 | 内容 |
|------|------|
| `.diff.archimate` | Plateau v85→v87 + Gap「SPA 绑 Node 进程」+ WP Docker nginx 常驻；APISIX→host:4000→taskfe-nginx→html |
| `.full.archimate` | v85 全量拓扑叠加容器 nginx 与废弃 preview |

## 验收（实施阶段）

1. `docker ps` 中 `taskfe-nginx` Up；`ss` :4000 为 `0.0.0.0`/`*`（docker-proxy），不是 vite/node preview
2. `curl -sI https://www.daydaymoney.com/user/<id>/profile/` → 200 HTML；构建中途再 curl 仍 200
3. `curl -sI http://127.0.0.1:4000/static/assets/does-not-exist.js` → **404** 非 200 HTML
4. `GET /health` → 200 JSON `ok: true`
5. runAll 仅 build taskFE：容器 **Id 不变**，symlink 指向新 release
6. 精准重启 taskFE：上游 502 窗口 ≈ 0（允许 reload 数毫秒）

## 实施边界（批准前不写代码）

- 不改 Vue 路由、不改 APISIX uris
- 不把 SH 边缘 nginx 当 SPA 根
- 不在 Python/Django 新增接口
