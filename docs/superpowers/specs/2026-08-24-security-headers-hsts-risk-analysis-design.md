# Design: 安全响应头补齐（HSTS/CSP/XFO/nosniff/Referrer-Policy）+ HTTP→HTTPS 强制跳转 — 风险分析

- **状态**: 🎯 target（待审批）
- **作者**: claude
- **日期**: 2026-08-24
- **类型**: 安全加固 — 边缘配置与响应头策略
- **背景**: 实测响应无 HSTS、CSP、X-Frame-Options、X-Content-Type-Options、Referrer-Policy；无 HTTP→HTTPS 强制跳转配置记录。本设计分析**解决该问题的风险**并给出分阶段修复方案。

---

## 1. 现状盘点（事实基线）

### 1.1 访问面拓扑

```
Internet (HTTPS)
   │
   ▼
宿主机 边缘 nginx :443  ── letsencrypt wildcard *.daydaymoney.com
│  （配置在 repo 外: /etc/nginx/conf.d/；repo 内仅有 gitlab 子域 example:
│    gitService/docs/edge-nginx-gitlab-tencent-sh-1.conf.example）
   ▼
APISIX (taskGateway :18081, Docker)   ← 已落地 v55「gateway 统管所有路径」
   ├── /api/*       → Go 服务（taskAuth/taskTask/…）+ Django legacy /api/user/*
   ├── /gateway/ops → docsPortal
   └── /* GET/HEAD  → taskFE nginx 容器（SPA, /srv/taskfe/html, 常驻 :4000→容器内 80）
```

内网访问面（全部纯 HTTP，无 TLS）：
| 面 | 地址 | 服务 |
|----|------|------|
| runAll UI | `http://10.2.150.68:9999/` | runAll Go 二进制 |
| FE 内网直连 | `http://10.2.150.68:3000` / `:4000` / `localhost:4000` | taskFE |
| 网关内网 | `http://10.2.150.68:18081` | APISIX |

### 1.2 现状安全事实（证据）

| 项 | 现状 | 证据 |
|----|------|------|
| TLS | 仅公网边缘 :443 有（letsencrypt）；内网零 TLS | taskFE nginx.conf 仅 `listen 80`；无 ssl 配置 |
| 安全头 | 全链无 HSTS/CSP/XFO/nosniff/Referrer-Policy | nginx.conf / apisix.yaml 均无 add_header/response-rewrite 安全头 |
| HTTP→HTTPS | 无 :80 跳转记录（repo 内无主站边缘配置） | 仅有 gitlab 子域 example |
| 内联脚本 | **dist/index.html 含内联 `<script>`（API base bootstrap）** | `taskFE/app/dist/index.html` |
| Referer 依赖 | taskGitOauth `febFromRequest` 用 Referer/Origin 解析 OAuth 回跳源（白名单 `PublicEntryOrigins`） | `taskGitOauth/src/app.go:364` |
| 其它 Referer 校验 | taskAuth 无 referer 校验 | grep 无命中 |
| OAuth 发起 | FE 全部 `window.location.href = authorize_url` 顶层导航 | `startRepoOauthAuthorize.js` 等 6 处 |
| SSE | EventSource `/api/sse/*`（同源经网关） | `establishSSEConnection.js` |
| 子框架 | iframe `srcdoc` 富文本执行日志（自身子框架） | `stickyIframeSrcdoc.js` |
| 外部资源 | STUN (WebRTC, 不受 CSP 管)、blob: 下载（EnvVarTableEditor）、外部头像等未全量盘点 | — |
| 构建 | Vite 生产产物；`/static/assets/*` hashed + immutable 缓存；index.html `Cache-Control: no-cache` | nginx.conf / vite.config.js |

---

## 2. 逐项风险分析（修复本身的风险）

### 2.1 HSTS — 风险 🟡 中

| 风险 | 说明 | 缓解 |
|------|------|------|
| 只能加在 TLS 终结层 | HSTS 仅在 HTTPS 响应中被浏览器接受 → 必须加在宿主机边缘 nginx :443；加在 APISIX/taskFE 层无效且误导 | 实施位置定为边缘 nginx |
| `includeSubDomains` 波及子域 | 一旦加 includeSubDomains，`*.daydaymoney.com` 全部子域被 pin 到 HTTPS；若存在 HTTP-only 子域（如 gitlab-tencent-sh-1 之外的裸服务）则访问断裂。需先盘点全部子域 TLS 合规 | 初始**不加** includeSubDomains；盘点子域后再开 |
| `preload` 长期承诺 | 预载列表要求全子域合规 + 大 max-age + 不可逆撤销；误提交影响全体浏览器 | **禁用 preload** |
| 回滚慢 | 浏览器缓存 HSTS 至 max-age 到期；误设大值（如 31536000）后无法快速收回 | 阶梯灰度：`max-age=300` → `86400` → `31536000` |
| 内网 IP 面无法修复 | 浏览器对 IP 地址**忽略 HSTS**；内网无 TLS，HSTS 无从生效 | 内网面声明 out of scope（见 §5） |
| 证书脆弱性放大 | HSTS 后浏览器不再回退 HTTP → 公网证书到期/续签失败即全站不可达（失去 HTTP 兜底） | 增加 letsencrypt 续签监控/告警 |

### 2.2 CSP — 风险 🔴 高（需前置工程，不可直接上）

| 风险 | 说明 | 缓解 |
|------|------|------|
| **内联脚本被拦 → 全站不可用** | `dist/index.html` 的 `__TASK2APP_API_BASE_URL__` 内联 script 被 `script-src 'self'` 拦截 → API base 解析失败 → 登录/接口全挂 | **前置工作**：① 内联脚本外置为 `/static/assets/api-base.js`（改 Vite 入口 html），或 ② 用 `'sha256-…'` 固定 hash（构建变更需同步 hash，CI 易碎，不推荐） |
| `'unsafe-inline'` 妥协 | 若图省事在 script-src 加 'unsafe-inline'，CSP 对 XSS 的防护价值基本归零（扫描器可接受但安全目标未达成） | 坚持外置脚本；script-src 保持 `'self'` |
| Vue `:style` 绑定 | `style-src` 若不含 'unsafe-inline'，大量组件 style 属性被拦 → UI 样式大面积丢失 | `style-src 'self' 'unsafe-inline'`（style 的 unsafe-inline 风险远低于 script，业界标准做法） |
| SSE 连接 | `connect-src` 需放行 `/api/sse/*`（同源 `'self'` 即可）；若 FE 从跨源加载则需加域名 | prod 同源 → `connect-src 'self'` 足够；dev 模式不加 CSP（Vite HMR eval） |
| iframe srcdoc 日志 | 富文本执行日志 iframe 需 `frame-src 'self'`（srcdoc 继承父源） | `frame-src 'self'` |
| 外部资源未盘点 | 头像/COS/微信等外部 img 域未全量确认 → `img-src` 可能漏白导致图片失效 | 上线前全站 audit img/script/style 加载域，列出完整白名单 |
| blob:/worker | EnvVarTableEditor blob 下载（导航不拦）；需确认无 Web Worker | audit 无 worker 即 `worker-src` 不特殊处理 |
| 只对 HTML 有意义 | 对 API 响应加 CSP 无意义且可能误伤（如 JSON 被当文档解析受限场景） | 仅 index.html 文档响应加 |
| **上线即全站白屏** | CSP enforce 一旦漏白，回归影响面 = 全站（所有用户） | **必须 CSP-Report-Only 灰度 1–2 周**，收集违规上报后再 enforce；上报端点需落库或可查 |

### 2.3 X-Frame-Options — 风险 🟢 低

| 风险 | 说明 | 缓解 |
|------|------|------|
| 平台被第三方 iframe 嵌入场景 | 若存在嵌入需求（未发现），DENY 会打断 | 当前未发现嵌入方 → `SAMEORIGIN` 即可；有嵌入需求时改用 CSP `frame-ancestors` 白名单 |
| 与 CSP frame-ancestors 互斥 | 浏览器对 XFO 与 frame-ancestors 不一致时**强制按 DENY 处理** | 二选一：用 XFO SAMEORIGIN 则不再配 frame-ancestors；两者值必须一致 |
| 自身子框架不受影响 | iframe srcdoc 是自身子框架，XFO 管的是「谁嵌入我们」 | 无影响 |

### 2.4 X-Content-Type-Options: nosniff — 风险 🟢 低

| 风险 | 说明 | 缓解 |
|------|------|------|
| 资源 MIME 错配 | SPA hashed 资源由 nginx mime.types 正确标注 → 无影响；但 nginx `default_type application/octet-stream` 意味着**任何未映射 MIME 的 .js/.css 会在浏览器被拒** | 上线前核对 `/static/assets/` 与下载接口的 Content-Type；下载类（COS 签名、日志）不受 nosniff 影响 |
| 全局生效面 | 对 API 响应加 nosniff 无害（JSON 本身就该是 application/json） | 可全链加，风险低 |

### 2.5 Referrer-Policy — 风险 🟡 中（有代码依赖）

| 风险 | 说明 | 缓解 |
|------|------|------|
| **OAuth 回跳源解析断裂** | taskGitOauth `febFromRequest`（app.go:364）用 **Referer/Origin** 在 `PublicEntryOrigins` 白名单中解析回跳 FE 源。若设 `no-referrer`/`same-origin`：跨源发起登录场景（如内网 `10.2.150.68:3000` 发起 → OAuth 回调）Referer 丢失 → 回退默认 fallback 源 → **跳错域名 / 登录断裂** | 选 **`strict-origin-when-cross-origin`**（浏览器默认值，保留 origin 供白名单匹配，安全性达标）；**禁止 no-referrer** |
| 回归覆盖缺失 | febFromRequest 多入口场景（公网 www / apex / 内网 IP）无 Referrer-Policy 回归测试 | 补 taskGitOauth 单测：模拟无 Referer/Origin、异源 Referer 场景 |
| 其它 Referer 依赖 | 已审计 taskAuth 无 referer 校验；网关 forward-auth 不用 referer | 无 |

### 2.6 HTTP→HTTPS 强制跳转 — 风险 🟡 中

| 风险 | 说明 | 缓解 |
|------|------|------|
| 跳转只适用于公网域 | 内网 10.2.150.68 各端口无 TLS → 无法跳转（目标必须存在 HTTPS）；内网面强行跳转=自断 | 跳转仅配置于公网边缘 nginx :80；内网面 out of scope |
| 301 永久缓存不可回滚 | 浏览器/客户端缓存 301；若后续要回退（如临时降级），旧 URL 被永久改写 | 用 **308**（保方法、保 body，适合 API 类）或 302 过渡；确认无 API 客户端直连 http 公网 |
| http:// 硬编码回调未盘净 | OIDC redirect_uri、GitHub OAuth callback（已 https）、微信支付回调（微信强制 https）— 若仍有集成以 http:// 注册，跳转后断裂 | 上线前 grep 全部 `http://daydaymoney.com` 配置；公网跳转后用复扫验证 |
| :80 现有业务确认 | 需确认 Host 边缘 :80 当前无直接 serve 内容（或仅 302 语义） | 改前 `nginx -T` 快照核对 |

### 2.7 交付/运维风险 🟡 中（最大现实风险）

| 风险 | 说明 | 缓解 |
|------|------|------|
| **主站边缘 nginx 配置在 repo 外** | `/etc/nginx/conf.d/` 属宿主机手管；repo 内只有 gitlab 子域 example。若只改 repo（APISIX/taskFE）而 Host 未 apply → **复扫仍不过**；反之改 Host 不落文档 → 配置漂移 | 双交付：① repo 内新增主站边缘 nginx example + runbook；② Host apply + `nginx -t` 验证 + 复扫 |
| 头加在哪一层 | HSTS 必须在边缘 nginx（TLS 终结层）；CSP/XFO/nosniff/Referrer-Policy 可在边缘统一加（覆盖 SPA+API），或 APISIX response-rewrite 全局插件（仅覆盖经网关流量） | 统一在边缘 nginx 加（单一事实源）；APISIX 层不加，避免双头不一致 |
| 内网直连面 | :9999 runAll UI / :18081 / :3000 加不加？内网无 PKI，加自签证书会造成浏览器告警反而更糟 | **不加**，内网信任域内声明 out of scope（文档写明理由） |
| 服务重启登记 | taskGateway（APISIX）/ taskFE（nginx.conf）变更须登记 `precise_restart_services.txt` 并走「精准编译重启」 | 实施时按硬约束 42 登记 |
| index.html 缓存 | 外置脚本改 `dist/index.html` 后，旧 index.html 为 no-cache 会快速更新；hashed assets 不变 → 无缓存问题 | 无额外动作 |

### 2.8 联动风险 🟡

- **证书续签单点**：HSTS + 跳转后，letsencrypt 到期未续 = 全站不可达（见 2.1）。
- **复扫范围预期**：若扫描器复扫内网 IP 面要求 HTTPS/HSTS —— 无公网 IP 证书无法满足，须在报告中明确「修复范围 = 公网域」。
- **微信内置浏览器**：对 CSP/XFO 兼容完整；无已知阻断。
- **APISIX allow_origins 含 `http://10.2.150.68:3000` 等内网源**：公网跳转不影响内网 CORS 源，无回归。

---

## 3. 修复方案草案（分阶段、按风险从低到高灰度）

### Phase 0 — 前置工程（先做，不对外发布）
1. `dist/index.html` 内联脚本外置为 `/static/assets/api-base.js`（改 `taskFE/app/index.html` + Vite 构建；验证 `__TASK2APP_API_BASE_URL__` 行为不变）
2. 全站 audit 外部加载域（img/script/style/connect/frame）→ 形成 CSP 白名单清单
3. 盘点 `*.daydaymoney.com` 全部子域 TLS 合规性（HSTS includeSubDomains 前提）
4. taskGitOauth 补 febFromRequest 回归单测（无 Referer / 异源 Referer / 无 Origin 场景）

### Phase 1 — 低风险头立即生效（边缘 nginx）
- `X-Content-Type-Options: nosniff`（全响应）
- `X-Frame-Options: SAMEORIGIN`（HTML 文档）
- `Referrer-Policy: strict-origin-when-cross-origin`（全响应）
- `HSTS max-age=300`（仅 :443；不含 includeSubDomains、不含 preload）

### Phase 2 — CSP 灰度（1–2 周）
- 边缘 nginx 对 HTML 文档下发 `Content-Security-Policy-Report-Only`（白名单按 Phase 0 audit）
- 收集违规上报，修正白名单；确认零生产违规后转 `Content-Security-Policy` enforce
- 预期最终策略骨架（以 audit 结果为准）：
  - `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-src 'self'; img-src 'self' <外部域>; object-src 'none'; base-uri 'self'; form-action 'self'`

### Phase 3 — HSTS 阶梯升级
- `max-age=300` 观察 ≥1 周 → `86400` → `31536000`（仍不加 includeSubDomains，除非子域盘点全绿）

### Phase 4 — HTTP→HTTPS 强制跳转
- 边缘 nginx `:80` → `308` → `:443`（先 302 过渡 1 周，确认无 http 客户端断裂后转 308）
- 复扫验证 + 报告

### 每阶段验收
- `curl -sI https://www.daydaymoney.com/` 核对头齐全
- `curl -sI http://www.daydaymoney.com/` 核对 308
- 前端冒烟：登录、OAuth（GitHub/GitLab/微信）、SSE 执行日志、富文本日志 iframe、容器页跳转、blob 下载
- 复扫：无「响应缺安全头」高危项；内网面缺 HSTS 项声明 out of scope

---

## 4. 影响面分析

### 4.1 🕸️ Code Review Graph 分析
`CRG unavailable: 图存在但过期（built 2026-08-24T01:56，head 不匹配），且 taskGitOauth 子仓符号（febFromRequest）未入图；本任务主体为 nginx/APISIX 配置（非 tree-sitter 可解析代码），图价值低`。以静态 grep/Read 审计替代（§1.2 证据表）。

### 4.2 Value Stream 影响
| 流 | 影响 |
|----|------|
| `user-auth`（login / git-site-oauth 等步骤） | Referrer-Policy 收紧影响 OAuth 回跳源解析 → 需回归测试；无字段/表变更 |
| 其余流（task/billing/…） | 无字段变更；仅受全局头影响（SSE 连通、下载）— 由 Phase 验收冒烟覆盖 |

### 4.3 业务意图 → 事件对照
| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 补齐安全响应头 / 强制 HTTPS | — | **无对应事件**：纯配置/基础设施变更，无业务状态变更、无跨边界副作用；CSP Report-Only 上报为运营遥测非业务事件 |

### 4.4 🐍 Python 新增接口清单
不触发（纯边缘/网关配置变更，零新增 Python endpoint）。

### 4.5 领域概念清单
无新实体/聚合/域事件（基础设施安全加固，不改变业务边界）。

---

## 5. 范围声明（✅ 已确认 2026-08-24）

| 面 | 处理 | 理由 |
|----|------|------|
| 公网 `www.daydaymoney.com` / `daydaymoney.com`（+ 已 TLS 子域） | ✅ 完整修复（Phase 0–4） | **扫描目标已确认为公网面**；边缘 nginx 可加 HSTS/跳转 |
| 内网 `10.2.150.68:*`、`localhost:*` | ⛔ out of scope | 无 TLS/PKI；HSTS 对 IP 无效；强制 HTTPS 需自签证书 → 浏览器告警损害大于收益 |
| runAll UI :9999 | ⛔ out of scope | 同上 |

> 决策记录：`scan_target: public-daydaymoney.com`（用户确认）；复扫报告应注明内网面缺 HSTS 为不可满足项。

---

## 6. 🏛️ 架构变更影响（✅ 已落盘 v106 target，2026-08-24 08:54）

- **迭代版本**: v106 🎯 target（用户已确认按安全策略变更生成）
- **迭代名称**: security-headers-hsts-https-redirect
- **作者**: claude | **设计日期**: 2026-08-24 08:54
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v106-application-integration-20260824-0854-claude.puml`
  - 🆕 `docs/architecture/v106-enterprise-landscape-20260824-0854-claude.puml`
  - 🆕 `docs/architecture/v106-application-integration-20260824-0854-claude.diff.archimate`（增量变迁：v104→v106 变更元素 + Plateau/Gap/WP 链）
  - 🆕 `docs/architecture/v106-enterprise-landscape-20260824-0854-claude.diff.archimate`（增量变迁视图）
  - 🆕 `docs/architecture/v106-application-integration-20260824-0854-claude.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v106-enterprise-landscape-20260824-0854-claude.full.archimate`（全量拓扑）
  - 🆕 伴生 `.mermaid.md`（每个视图）
  - ✅ 全部通过 Archi `--loadModel` 验证；PlantUML 语法检查通过
- **已有文件（未修改）**: v104 (current) 两个视图 puml + 伴生格式
- **变更明细**:
  - 🟡 [MODIFIED v106] 边缘 nginx（application-integration + enterprise-landscape）：安全响应头 + :80→:443 308 跳转 + 证书续签监控
  - 🟢 [NEW v106] Motivation 约束 × 3：HSTS 阶梯灰度 / CSP Report-Only 灰度 / Referrer-Policy 保留 Origin

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量模型 — Plateau v104 → Gap → WorkPackage → Plateau v106 链 + 边缘 nginx 🟡 + 约束 🟢；视图「架构变迁 v104→v106」+「v106 Target — 边缘安全头 + HTTPS 跳转」（含 sourceConnection 连线） |
| **`.full.archimate`** | 全量模型 — 变迁后拓扑（边缘/网关/SPA/云服务/Django legacy/事件/Kafka + 变更标注）；视图「v106 全量拓扑」+「架构变迁」（含 sourceConnection 连线） |

---

## 7. 风险结论（一句话）

> 本修复**主要风险不在「加头」本身（nosniff/XFO/Referrer-Policy 低风险），而在 CSP 与 HSTS**：CSP 需先外置内联脚本 + Report-Only 灰度（否则全站白屏），HSTS 需避免 includeSubDomains/preload 误伤子域且须防证书单点；**最大现实风险是主站边缘 nginx 配置在 repo 外**，repo 内改动 ≠ 线上生效，须 example + runbook 双交付并 Host apply 后复扫验证。
