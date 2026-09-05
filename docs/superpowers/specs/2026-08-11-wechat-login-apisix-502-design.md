# 微信登录 APISIX 502（上游 Connection refused）— 设计文档

- **日期**: 2026-08-11
- **状态**: implemented（/goal 已落地，2026-08-11；待精准编译重启 taskFE + runAll 热加载验证）
- **迭代**: wechat-login-apisix-502-hardening
- **作者**: claude
- **范围选择**: 用户确认 **1+2+3**（根因归档 + 韧性加固 + 可观测性补齐）
- **关联**: `docs/superpowers/specs/2026-08-11-db-reset-stale-login-ui-design.md`（清库后登录态 UI；同窗口常伴随服务停启）

---

## 1. 问题陈述

用户在登录页点击「微信扫码登录」后，浏览器跳转到：

`https://www.daydaymoney.com/api/auth/wechat/login/?app=web&next=/tenant/.../task-detail/.../`

页面显示 **502 Bad Gateway**（APISIX/openresty），无法进入微信扫码。

---

## 2. 运行时证据（2026-08-11）

| 检查项 | 结果 |
|--------|------|
| 公网首次探测 | HTTP **502**，`x-apisix-upstream-status: 502`，`x-trace-id: 6370502f2f293bffbced1737698e0fd4` |
| Loki（job=~".+"，1h/24h/7d） | **0 条**（该 traceId 从未进入 taskAuth） |
| APISIX access（`taskGateway/logs/taskgateway-access.log`） | **命中**同 URI，`route_id=taskauth-login`，`upstream=172.17.0.1:8003`，`status=502` |
| APISIX error.log | `connect() failed (111: Connection refused)` → `http://172.17.0.1:8003/api/auth/wechat/login/...` |
| 同窗口副作用 | `vue-frontend :4000` favicon 亦 Connection refused（多服务停机窗口） |
| taskAuth 进程 | **02:10:01** 重新 `listening on 0.0.0.0:8003` |
| 复测同一长 URL | **302** → `open.weixin.qq.com/connect/qrconnect`（已恢复） |

### 根因（已证实）

**不是**微信 OAuth 配置、`next` 过长、或路由缺失。

链路：`taskFE` 整页跳转 → 边缘 nginx → **APISIX** `taskauth-login` → `up-taskAuth`=`host.docker.internal:8003` → **当时无进程监听** → APISIX 返回 502。

触发窗口与 **清库/停服务/精准重启** 高度吻合：`ClearAllDatabases` 会停业务进程；`InitAllDatabases` 仅预拉起 `task-bill` / `tenant` / `project`，**不保证** `task-auth` / `vue-frontend` / 网关侧上游就绪。用户在该窗口点微信登录即见裸 502。

### 同类事故（2026-08-29 callback，trace_id `cf7b5ea2918884602b81fd80eae1efb1`）

登录 302 已成功；微信回调 `GET /api/auth/wechat/callback/?code=…&state=…53bf0456…` 打到 APISIX 时 taskAuth `:8003` 已无监听，JSON 体为网关 `error_page 502`（`服务暂时不可用，请稍后重试`）。

- Loki 仍空（aimonitor-promtail 未跑）；证据在 `taskGateway/logs/error.log` `connect() failed (111)` + `runall-console.log`。
- **不是** OAuth `code`/`state`/`redirect_uri` 配错。`code` 一次性，禁止重放。
- 根因：runAll `rt_sigtimedwait` 把 `EINTR` 当退出 → ADR-0035 skip SIGTERM 但 **StdoutPipe 读端关闭 → 子进程 SIGPIPE**。修复见 ADR-0035 mitigations（文件 FD stdio + EINTR 重试）。

---

## 🔍 Trace 日志分析 (traceId: `6370502f2f293bffbced1737698e0fd4`)

- **Grafana Trace Dashboard**: [打开](http://localhost:3000/d/distributed-trace-view?var-trace_id=6370502f2f293bffbced1737698e0fd4&var-tempo_trace_id=6370502f2f293bffbced1737698e0fd4)
- **Grafana 日志搜索**: [打开](http://localhost:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"6370502f2f293bffbced1737698e0fd4\"","queryType":"range"}]})
- **权威证据文件**: `taskGateway/logs/taskgateway-access.log` + `error.log`（未进 Loki）

### 🔍 TraceId 日志缺失诊断报告

| 步骤 | 结果 |
|------|------|
| 主查询 / 回退 / 7d | 0 条 |
| Loki ready + 有其它日志 | 是（仅见 `task-auth` job） |
| D5 原始日志 | **命中** APISIX access/error |

- **根因**: Promtail 只刮 `/var/log/runall/task-gateway.log`（runAll 包装日志），**未刮** `taskGateway/logs/taskgateway-access.log` / `error.log`；502 由网关本地生成，taskAuth 无请求日志。
- **短期**: 设计纳入 APISIX access/error → Loki。
- **对本次设计影响**: 无 APISIX job 时，有 `X-Trace-Id` 仍无法在 Grafana 还原网关 502。

---

## 🕸️ Code Review Graph 分析

- `code-review-graph status` OK，但图以 JS/TS/Python 为主，**无 Go `handleWeChatLogin` 覆盖**。
- 记：`CRG unavailable for Go wechat path`；静态确认 `taskAuth` 路由 `GET /api/auth/wechat/login/` 存在且健康时 302。

---

## 3. 目标

1. **归档根因**：文档 + 验收用例，确认恢复路径。
2. **韧性**：清库/初始化后关键登录路径服务自动就绪；登录页遇上游不可达时**不整页掉进裸 502**，可重试并展示 `data-traceId`。
3. **可观测**：APISIX access/error（含 `x-trace-id`、`status`、`route_id`、`upstream`）进入 Loki，可按 traceId 检索。

**非目标**：改造微信开放平台回调协议；新增 Python 接口；拆分微信登录到其它服务。

---

## 4. 推荐方案

### 4.1 runAll：清库/初始化后恢复登录关键路径（韧性主修复）

在 `InitAllDatabases` **成功结束后**（或新增显式「清库后恢复」钩子），按依赖序确保下列服务 **running + health**：

| 顺序 | 服务 | 理由 |
|------|------|------|
| 1 | `task-auth` | 微信登录 / forward-auth / token |
| 2 | `task-gateway` | APISIX；`depends_on: [task-auth, …]` |
| 3 | `taskFE` | SPA；避免同窗前端上游亦 502 |

规则：

- 已在跑且 health 通过 → 跳过。
- 未跑 → `StartService` + 既有 health_check。
- 失败 → init 结果标记 `partial` + UI 日志明确「登录关键路径未就绪」，**禁止**静默成功。

可选（同批或 follow-up）：`ClearAllDatabases` 完成提示「业务已停止，请执行初始化或全部启动」——避免用户以为清库后仍可登录。

### 4.2 taskFE：微信入口预检（体验加固）

现状：`Login.vue` → `window.location.href = buildWechatOAuthUrl(...)`，浏览器直接吃 APISIX HTML 502。

改为：

1. 构造 URL（沿用 `wechatLoginFlow.js`）。
2. `fetch(url, { method: 'GET', redirect: 'manual', credentials: 'same-origin' })` 预检。
3. 若 `302` / `opaqueredirect` → `window.location.assign(url)`（或读 `Location`）。
4. 若 `502`/`503`/`0` 网络失败 → **留在登录页**，toast/alert：「登录服务暂时不可用，请稍后重试」，展示 **`data-traceId`**（响应头 `X-Trace-Id` 或生成前端 id）。
5. 单测覆盖：预检 302 放行；502 不跳转且带 traceId。

> 说明：预检无法消除上游宕机，但避免「整页白屏 502」；与 4.1 互补。

### 4.3 Promtail：刮取 APISIX access/error（可观测主修复）

新增 job（建议）：

| job | path | 解析 |
|-----|------|------|
| `apisix-access` | `<repo>/taskGateway/logs/taskgateway-access.log`（或部署机等价路径） | JSON：`response.status`、`request.uri`、`route_id`、`upstream`、headers.`x-trace-id` → `trace_id` structured_metadata |
| `apisix-error` | `<repo>/taskGateway/logs/error.log` | 文本：`Connection refused` + 可选从邻近 access 关联；至少全文可 `|= "<traceId>"`（若 error 行无 id，靠时间窗 + URI） |

部署注意：

- Promtail 容器/本机须能读到 `taskGateway/logs`（volume 或 symlink 到现有 scrape 根）。
- 改完重启 Promtail；用已知 502 traceId 或新制造 Connection refused 做验收。

### 4.4 APISIX（可选、降优先级）

对 `up-taskAuth` 增加 **active health check**（`GET /api/health/`）。

- **收益**：多节点时摘除坏节点；单节点开发机上坏节点=整上游，仍 502，但 error 更干净、可配合监控。
- **不做**：自定义 HTML 错误页大改造（FE 预检已覆盖主体验）；本次不强制。

若做：在 `taskGateway/apisix/apisix.yaml` 的 `up-taskAuth` 增加 `checks.active`，改后 reload/restart gateway；单测/脚本验证 yaml 仍 `#END` 可加载。

---

## 5. 架构变更判断

| 维度 | 变更 |
|------|------|
| 服务拓扑 | 无新增服务 |
| 调用关系 | 🟡 FE 微信入口增加同源预检；runAll init 后强制拉起 auth/gateway/fe |
| 可观测 | 🟡 Promtail 新增 APISIX access/error 采集 |
| 数据所有权 | 无 |

**需要**新建 architecture target（建议仅 `application-integration` v71）：标注 Promtail/APISIX/taskFE/runAll 的 🟡 MODIFIED。  
`enterprise-landscape` 可不动（无新组件类型）。

---

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 用户点击微信登录（发起 OAuth） | — | — | — | **纯重定向/预检**，无服务端状态变更；既有登录成功路径仍发既有用户/身份事件，本次不改 |
| 清库后恢复登录关键路径 | — | runAll Init hook | — | **运维编排**，无领域事实变更 |

---

## 7. 价值流影响

- 影响流：`user-auth`（微信登录入口可用性；步骤可新增 `wechat-login-gateway-ready` planned→active 于 step 4）。
- 字段：无 DB 字段变更。
- 测试：taskFE 单测 + runAll init 后服务就绪单测 + Promtail 配置校验（yaml load）。

---

## 🐍 Python 新增接口清单与 Go 替代评估

**not_applicable** — 无新增 Python HTTP 接口；改动在 runAll（Go）、taskFE、Promtail、可选 APISIX yaml。

`python_api_approval: not_applicable`

---

## 8. 验收标准

1. 停掉 `task-auth` 后点微信登录：**留在登录页**，可见错误文案 + `data-traceId`（非裸 APISIX HTML）。
2. 启动 `task-auth` 后点微信登录：302 进入微信扫码（与现网一致）。
3. `InitAllDatabases` 结束后：`task-auth` / `task-gateway` / `vue-frontend` health 通过（或 UI 明确 partial）。
4. Loki 可查：`{job="apisix-access"} |= "<X-Trace-Id>"` 命中含 `wechat/login` 与 `status`。
5. 制造一次 Connection refused 后，error/access 任一 job 可在 1 分钟内检索到。

---

## 9. 实现切片（供后续 /7-plans）

1. Promtail jobs + 路径可达性（最快闭合排障缺口）
2. runAll Init 后恢复登录关键路径 + 测试
3. taskFE 微信预检 + 单测
4. （可选）APISIX `up-taskAuth` active health check
5. 架构 v71 四件套 + VERSION_HISTORY

---

## 🏛️ 架构变更影响

- **迭代版本**: v71 🎯 target
- **迭代名称**: wechat-login-apisix-502-hardening
- **作者**: claude
- **设计日期**: 2026-08-11 02:22
- **新增文件**:
  - 🆕 `docs/architecture/v71-application-integration-20260811-0222-claude.puml`
  - 🆕 `docs/architecture/v71-application-integration-20260811-0222-claude.diff.archimate`
  - 🆕 `docs/architecture/v71-application-integration-20260811-0222-claude.full.archimate`
  - 🆕 `docs/architecture/v71-application-integration-20260811-0222-claude.mermaid.md`

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v70→v71 + Gap（auth 未拉起 / 无 Loki / 裸 502）+ WP；目标拓扑含 FE→GW→Auth + Promtail→Loki |
| **`.full.archimate`** | 全量拓扑 + 变迁视图；含 runAll→auth/gateway/taskFE |

---

## 落地记录（/goal 2026-08-11）

| 切片 | 状态 | 证据 |
|------|------|------|
| Promtail apisix-access/error | ✅ | Loki job 标签出现；`{job="apisix-access"} \|= "6500e4d5…"` 命中 |
| EnsureLoginCriticalPath | ✅ | `go test ./src/domain -run LoginCritical\|EnsureLogin` 全绿 |
| navigateWechatOAuth | ✅ | vitest 12/12 绿 |
| 架构 v71 | ✅ | Archi `Loaded model` diff+full |
