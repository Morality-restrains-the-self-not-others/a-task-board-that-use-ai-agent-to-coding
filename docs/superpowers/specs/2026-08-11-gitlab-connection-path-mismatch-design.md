# 租户 GitLab 连接页「无法加载」— 设计文档

- **日期**: 2026-08-11
- **状态**: accepted（用户批准方案 B/C：后端对齐 key/value + 旧路径兼容，2026-08-11）
- **迭代**: gitlab-connection-path-mismatch
- **作者**: claude
- **页面证据**: `https://www.daydaymoney.com/tenant/874599492341493760/settings/gitlab-connection/`
- **DOM**: `p[data-testid="gitlab-connection-error"][data-traceid="49b2be86-dd59-4269-b9ed-dd8cf6c59dec"]` →「无法加载 GitLab 连接」

---

## 1. 问题陈述

租户设置页「GitLab 连接」加载失败，前端展示通用文案「无法加载 GitLab 连接」，并带 `data-traceId`。

## 2. 🔍 Trace 日志分析 (traceId: `49b2be86-dd59-4269-b9ed-dd8cf6c59dec`)

- **Grafana Trace Dashboard**: [打开](http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=49b2be86-dd59-4269-b9ed-dd8cf6c59dec&var-tempo_trace_id=49b2be86-dd59-4269-b9ed-dd8cf6c59dec)
- **Grafana 日志搜索**: [打开](http://10.2.150.68:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"49b2be86-dd59-4269-b9ed-dd8cf6c59dec\"","queryType":"range"}]})
- **时间**: 2026-08-10 16:44:55 UTC（网关 access log）
- **涉及服务**: APISIX `git-oauth-token` → `taskGitOauth :8002`

### Loki 空结果诊断

| 步骤 | 条件 | 结果 |
|------|------|------|
| 主查询 / 回退 1–2 | 1h–24h，跨 job | 0 条 |
| D1 7d 全文 | `|= "<id>"` | 0 条 |
| D2 任意日志 1h | `{job=~".+"}` limit 1 | **0 条**（Loki ready 但无 ingest） |
| D3/D7 变体/前缀 | 7d | 0 条 |
| **替代源 D5** | `taskGateway/logs/taskgateway-access.log` | **命中** |

**根因（日志管道）**: Loki 数据管道当前无写入（本机与 `10.2.150.68:3100` 均 ready 但空）。排障改用网关 access JSON。

### 网关 access 关键发现（同一 trace）

| 字段 | 值 |
|------|-----|
| `request.uri` | `/api/git-oauth/tenant-connection/tenant_id/874599492341493760/` |
| `x-gateway-auth-verified` | `1`（鉴权已通过） |
| `x-user-id` | `874694567671132160` |
| `upstream` | `172.17.0.1:8002` |
| **`response.status`** | **404** |
| `response` body（上游） | `404 page not found`（`text/plain`） |

### 复现对照（本机 :8002）

| 路径 | HTTP | 说明 |
|------|------|------|
| `/api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/` | 401 `未认证` | **后端契约命中**（无 cookie 时鉴权拒绝） |
| `/api/git-oauth/tenant-connection/tenant_id/{tid}/` | **404** | **前端当前路径 → NotFound** |
| `/api/git-oauth/tenant-connection/tenant_id/{tid}/gitlab-oauth-connection/` | 400 `missing tenant id` | 键值段被误解析 |

### 根因假设（已用日志+直连证实）

1. **主因**：前后端路径契约不一致。  
   - FE（`e105bad` 路径约定迁移）：`/api/git-oauth/tenant-connection/tenant_id/{tid}/`  
   - BE（OpenAPI + `extractTenantIDFromPath`）：`/api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/`  
   - `handleTenantRoutes` 仅当 path 含 `/gitlab-oauth-connection` 才分发，否则 `http.NotFound`。
2. **文案表象**：404 正文为 plain text，`response.json()` 得不到 `detail`，UI 回落「无法加载 GitLab 连接」。
3. **非主因**：鉴权/清库（该 trace 已 `x-gateway-auth-verified=1`）；与「清库后假登录」是独立问题。

---

## 3. 🕸️ Code Review Graph 分析

- CRG 图存在但仅覆盖 js/ts/python/bash（16 files），**无 Go 符号覆盖**；本次关键契约在 `taskGitOauth` Go + `taskFE` Vue。
- 爆炸半径（手工）：`WorkspaceSettingsGitlabConnection.vue`（唯一 FE 调用点）↔ `handleTenantRoutes` / OpenAPI / `tenant_connection_*_test.go` / `db/api_route_ownership.yaml`。
- 记录：`CRG unavailable for Go path contract: graph languages exclude Go`

---

## 4. 方案决策

项目强制路径规范（`.cursor/rules/api-url-path-design.mdc`）：

```text
/api/${serviceName}/${funcName}/${key}/${value}/...
```

前端 `e105bad` 已按规范改写；**后端未跟进**。因此推荐 **后端对齐 key/value + 保留旧路径兼容**，前端路径保持不变。

### 目标契约

| 角色 | 路径 | 方法 |
|------|------|------|
| **Canonical（新）** | `/api/git-oauth/tenant-connection/tenant_id/{tid}/` | GET / PUT / DELETE |
| Compat（旧） | `/api/git-oauth/tenant-connection/{tid}/gitlab-oauth-connection/` | 同上 |
| Internal（不变） | `/api/internal/git-oauth/gitlab-tenant-connection/?company_id=` | GET |

### 实现要点（taskGitOauth）

1. 扩展 `extractTenantIDFromPath` / `handleTenantRoutes`：
   - 识别 `.../tenant-connection/tenant_id/{tid}/`（允许可选尾段）
   - 保留 `.../tenant-connection/{tid}/gitlab-oauth-connection/`
2. OpenAPI paths 增加 canonical；旧 path 标 deprecated 或并列文档化
3. 单测：`tenant_connection_test.go` + path 解析表测（新/旧/非法）
4. 更新 `db/api_route_ownership.yaml`：新增 `/api/git-oauth/tenant-connection/` 前缀所有权（替换或补充过时的 `/api/tenant/*/gitlab-oauth-connection/`）

### 前端（taskFE）

1. **保持**现有 `apiPath`（已符合规范）
2. **测试**：`WorkspaceSettingsGitlabConnection` 增加/恢复对 GET URL 断言（禁止再写成旧 Django `/api/tenant/...` 或漏 `tenant_id` 段）
3. （可选加固）非 JSON 错误体时展示 `HTTP {status}` + traceId，避免纯「无法加载」——非本次阻断项

### 网关

- 已有 `git-oauth-token`：`/api/git-oauth/*` → :8002，**无需改路由**

---

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 加载连接配置 | — | — | — | 纯查询，无状态变更 |
| 保存连接 | `TENANT_GITLAB_OAUTH_CONNECTION_UPSERTED` | `handleTenantGitlabConnectionPut` | 既有 | 已有，路径修复不改语义 |
| 删除连接 | `TENANT_GITLAB_OAUTH_CONNECTION_DELETED` | `handleTenantGitlabConnectionDelete` | 既有 | 同上 |

---

## 6. Domain Concept Inventory（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | Git OAuth（taskGitOauth） |
| Entity | TenantGitLabOAuthConnection（按 `company_id`） |
| Aggregate | 租户级 OAuth App 配置 |
| Domain Events | Upserted / Deleted（已有） |

---

## 7. Value Stream Impact

- 触及 `conf/value-stream.yaml` 中 git-oauth 相关步骤（租户 GitLab OAuth 绑定前置：租户须先配置 connection）
- **无新 stream**；不改字段 schema；预期补 FE/Go 契约单测
- 全量价值流切片留给 `/4-value-stream`（若审批后需要）

---

## 8. 🐍 Python 新增接口

**not_applicable** — 无新增 Python endpoint；修复落在 Go `taskGitOauth` + 前端路径保持。

---

## 9. 🏛️ 架构变更影响

- **不需要**新架构版本：无组件/数据流/基础设施增删改，属契约对齐 Bugfix。
- **No-ADR**: trivial tech choice / bugfix, no architectural impact（路径解析与 ownership 文档更新）

---

## 10. 验收标准

1. 已登录租户管理员打开 `/tenant/{tid}/settings/gitlab-connection/`，自建连接区不再出现「无法加载 GitLab 连接」（未配置时展示空表单 `configured:false`）
2. 直连 `:8002`：`GET /api/git-oauth/tenant-connection/tenant_id/{tid}/` → 401（无凭据）或 200（有凭据），**不再 404**
3. 旧路径仍可用（兼容）
4. OpenAPI + ownership YAML + 单测绿灯
5. 公网 SPA：`taskFE` 改动后按既有规范 `runall-lifecycle.sh build`（若仅 Go 改动且 FE 不变，可只精准重启 `taskGitOauth`）

## 11. 举一反三（Search）

| 范围 | 模式 | 结果 |
|------|------|------|
| taskFE | `tenant-connection` | **仅** `WorkspaceSettingsGitlabConnection.vue` 一处 |
| taskGitOauth | `gitlab-oauth-connection` / OpenAPI | 后端仍旧契约 |
| ownership | `/api/tenant/*/gitlab-oauth-connection/` | **过时**，须随修更新 |
| 网关 | `/api/git-oauth/*` | 已覆盖，无需改 |

---

## 12. 推荐落地顺序

1. Go：路径解析 + 测试（红→绿）
2. OpenAPI + `api_route_ownership.yaml`
3. FE：URL 契约单测（源 path 已正确则只加固测试）
4. 精准编译重启 `taskGitOauth`；若测改 FE 构建产物则 build FE
)
