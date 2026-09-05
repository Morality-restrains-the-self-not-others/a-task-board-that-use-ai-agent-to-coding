# 设计文档：容器执行全链路绕过 Django

- **版本**: v1.0
- **作者**: claude
- **日期**: 2026-07-10 15:05
- **迭代**: Container Exec Hot-Path Zero-Django（v13）
- **状态**: ✅ Phase A–F shipped（含 Django tcg internal 全清、auth-context GitLab/readiness、env AI agent、djangoInternalApiBase 删除）
- **前置**:
  - L0 outbound / relay 公网已切 `taskGateway → taskContainerGateway`（Django 公网 410）
  - `taskCloudService` 已具备 `GET /api/internal/cloud-server-config/container-target/`
  - `taskAuth` 已具备 `GET|POST /api/internal/gateway/forward-auth/`
  - 运行时证据（切流前）：旧二进制每次 job 轮询 `forward_stage=django_validate`；Django 不可达 → 502。切流后（2026-07-10 15:23 重启）为 `auth_validate` / `cloud_member` / `cloud_resolve`。

---

## 1. 背景与动机

### 1.1 现象（runAll :9999 / saas-backend）

在 [runAll Status](http://183.250.1.132:9999/) 查看 **saas-backend** 日志时，可见与容器执行相关的请求痕迹。根因不是「公网仍走 Django 转发」，而是：

> **taskContainerGateway 每个容器 outbound 请求仍同步回调 Django internal**（`validate-session` + `resolve-container-target`），因此 saas-backend 必然出现容器执行相关日志；且 Django 成为热路径单点——不可达则整条执行链 502。

### 1.2 现状调用链（问题路径）

```text
Browser
  → taskGateway (:18081)          # priority 890 → tcg
  → taskContainerGateway (:8014)
      ├─ POST Django /api/internal/.../validate-session/     ← 热路径依赖
      ├─ POST Django /api/internal/.../resolve-container-target/ ← 热路径依赖
      └─ POST onlineServiceJS .../api/jobs                   ← 真正执行
```

公网 `CloudComputeViewSet` 对已迁 action 返回 410，**但 internal 双跳仍经 Django**，故「整个链路不经过 Django」尚未达成。

### 1.3 成功标准（SMART）

| # | 标准 | 可核验方式 |
|---|------|-----------|
| S1 | 容器执行热路径（`container-layer-command`、`container-job-*`、层/文件读、job-stream 轮询）**零 Django HTTP** | tcg 日志无 `django_validate` / `django_resolve`；改为 `auth_validate` / `cloud_resolve` |
| S2 | saas-backend 在上述操作期间 **无** `/api/internal/task-container-gateway/validate-session|resolve-container-target` 访问 | runAll saas-backend 日志 / Loki `{job="saas-backend"}` |
| S3 | Django 进程停止时，已注册容器的 job 提交与轮询仍可用（鉴权与 resolve 不依赖 :8001） | 停 saas-backend 后发一条 layer-command，期望 2xx |
| S4 | 鉴权语义不弱化：未登录 401；非租户成员 403；无 server_url/token 409 | 单测 + 契约测 |
| S5 | 冷路径（git-push L3 prepare/complete、mock-run、open-runtime-session）可分阶段迁出；本设计明确边界与验收 | 见 §3 / §6 |

---

## 2. 架构基线确认

根据 `docs/architecture/` 与 `VERSION_HISTORY.md`：

> **系统现状摘要**
>
> - **交付态基线**：v11 shipped（Vendor Cloud Credentials）；v12 target（relay clear-logs path scope）
> - **应用层（容器执行相关）**：
>   - `taskContainerGateway` :8014 — L0/L1 出站编排（已绕 Django 公网）
>   - `taskCloudService` :8018 — CloudServerConfig SSOT + `container-target` internal（**已存在，tcg 未接线**）
>   - `taskCredentialService` :8015 — container access_token SSOT
>   - `taskAuth` :8003 — forward-auth / token resolve（**可复用，缺租户 scope**）
>   - `saas-backend` :8001 — 仍承载 tcg internal：`validate-session`、`resolve-container-target`、git-push prepare、job-stream publish、open-runtime-session
> - **数据流（当前）**：Vue → APISIX → tcg → **Django×2** → onlineServiceJS

📋 **架构版本历史（相关）**

| 版本 | 状态 | 要点 |
|------|------|------|
| v5–v6 | target/部分 | 引入 taskContainerGateway / taskCloudService |
| v9 | target | CloudServerConfig / container-target 迁 Go |
| v12 | target | relay clear-logs path scope |
| **v13** | 🎯 本设计 | 容器执行热路径零 Django |

---

## 3. 范围边界

### 3.1 纳入（本迭代必须达成「执行热路径零 Django」）

| 能力 | 现状 | 目标 |
|------|------|------|
| Session/Token 鉴权 | Django `validate-session` | **taskAuth** 扩展 internal API（见 §5.1） |
| 租户成员 + 任务 scope | Django `company_memberships` + Django ORM `CloudServerConfig.exists` | 成员：taskAuth/公司域；scope：`taskCloudService` lookup（relay 路径豁免同现逻辑） |
| Resolve base_url + token | Django `resolve-container-target`（读 Django ORM + Credential） | **直接** `taskCloudService` `container-target`（内部再调 Credential） |
| tcg 日志 stage 名 | `django_validate` / `django_resolve` | `auth_validate` / `cloud_resolve`（可观测验收） |

### 3.2 明确分阶段（不阻塞 S1–S3，但写入路线图）

| 能力 | 现状 | 目标阶段 | 说明 |
|------|------|----------|------|
| `publish-container-job-stream` | Django internal | **Phase B** | 改为 tcg 直写 Kafka / taskSSE，与既有 job-stream 设计对齐 |
| `prepare/complete-layer-git-push*` | Django L3 | **Phase C** | 冷路径；可暂留 Django，但须在文档标注「非执行热路径」 |
| `open-runtime-session` | Django/Cloud 混用 | **Phase B/C** | relay 启动副作用；优先走已有 Cloud upsert |
| `mock-run-container/*` | Django 实现；经 TCS 可能 501 | **Phase D** | 经 tcg 或直达 `go_run_container`，对齐 relay 旁路 |
| 删除 Django `forward_container_*` / internal 死代码 | 残留 | **Phase D** | 网关常开后强制单路径 |

### 3.3 不纳入

| 项 | 理由 |
|----|------|
| accounts / company 公网 API 迁出 | 与容器执行无关 |
| APISIX 直连 onlineServiceJS | 跳过网关鉴权与 resolve，风险过高 |
| 改变 `machine_container.md` 容器侧协议 | 本设计只改平台侧编排 |

---

## 4. 方案比选（已自动选定）

| 方案 | 描述 | 优点 | 缺点 | 结论 |
|------|------|------|------|------|
| **A. tcg → taskAuth + taskCloudService** | 热路径两跳改为 Go 服务 | 复用已有 SSOT；与 v9 设计一致；Django 可停 | 需补 taskAuth 租户 scope API | **采用** |
| B. tcg 内嵌鉴权 + 直读 Credential/Cloud DB | 减少一跳 | 耦合数据所有权，违反单服务数据所有权 | 否 |
| C. APISIX forward-auth + 直转 OSJS | 最短路径 | 无 resolve/token 注入、无 L0 映射 | 否 |
| D. 仅把 Django internal 日志降噪 | 治标 | 不解决 502 单点 | 否 |

**选定 A**：与既有 Go 拆分方向一致；`container-target` 与 `forward-auth` 已部分存在，改动面可控。

---

## 5. 目标架构

### 5.1 目标热路径

```text
Browser
  → taskGateway (:18081)
  → taskContainerGateway (:8014)
      ├─ POST/GET taskAuth  /api/internal/container-gateway/validate-session/
      │     （或扩展 forward-auth：返回 user_id + tenant_membership_ok）
      ├─ GET  taskCloudService /api/internal/cloud-server-config/container-target/
      │     ?tenant_id&workspace_id&task_id[&container_page_url]
      │     （内部：CloudServerConfig + taskCredentialService token）
      └─ forward → onlineServiceJS /api/jobs|...
```

**Django 不在图中。**

### 5.2 validate-session 迁移细节

现 Django 逻辑（`internal_views.validate_session`）：

1. 校验 internal secret  
2. 用 Cookie / Authorization 做 Session 或 Token 认证  
3. 校验 `user.company_memberships` 含 `tenant_id`  
4. 非 `relay-to-trae` 路径时要求 Django `CloudServerConfig` 行存在  

目标（taskAuth，推荐新路径以免污染 APISIX forward-auth 语义）：

```http
POST /api/internal/container-gateway/validate-session/
X-Internal-Secret: ...
{
  "cookie": "...",
  "authorization": "...",
  "tenant_id": "...",
  "workspace_id": "...",
  "task_id": "...",
  "path": "...",
  "method": "POST"
}
→ 200 { "user_id", "auth_method", "scope_ok": true }
```

实现要点：

| 步骤 | 归属 |
|------|------|
| 解析 Cookie session / Bearer / AccessToken | taskAuth（已有 resolve 能力） |
| 租户成员 | taskAuth 读本地用户-公司关系，或 internal 调公司域；**禁止**再回 Django |
| CloudServerConfig.exists（非 relay） | tcg 在 auth 通过后调 `taskCloudService` lookup；或 validate API 内调 Cloud（推荐 **tcg 编排**：auth 只做身份+成员，scope 由 cloud lookup） |

**推荐编排（降低 taskAuth 对 Cloud 的耦合）**：

```text
tcg:
  1) authValidate(session) → user_id + tenant_member_ok
  2) if not relay path: cloudLookup(scope) → 404 则 403/409
  3) cloudResolveTarget(scope) → base_url + access_token
  4) upstream forward
```

### 5.3 resolve-container-target 迁移细节

- **删除** tcg → Django `resolve-container-target`  
- **新增** `cloudResolveTarget`：`GET {CloudServiceURL}/api/internal/cloud-server-config/container-target/?...`  
- 支持 `container_page_url` override：若现 Go handler 未支持，扩展 query/body 与 Django 行为对齐（`override_container_page_url`）  
- Token 仅来自 Credential SSOT（与现 Go `resolveContainerTarget` 一致）

### 5.4 可观测性与日志归属

| 事件 | 落点 | 不再落点 |
|------|------|----------|
| 鉴权 / resolve / upstream | `logs/task-container-gateway/` + Loki job=`task-container-gateway` | saas-backend |
| 容器 inbound（换票等） | taskAgentSupport / taskCloudService / Credential | Django middleware（已迁） |
| 真正 job 执行 | onlineServiceJS / go-relay | Django |

验收：runAll 打开 saas-backend 日志，执行 zTree「发送指令」与 job 轮询，**不应再刷** `task-container-gateway` internal 行。

### 5.5 失败语义对齐

| 条件 | HTTP | body 要点 |
|------|------|-----------|
| 未认证 | 401 | authentication required |
| 非成员 | 403 | forbidden scope |
| 无 CloudServerConfig | 404/403 | 与现网一致（建议保持 403 forbidden scope 或 404 配置不存在，选一并写契约测） |
| 无 server_url | 409 | 容器尚未注册… |
| 无 token | 409 | 缺少容器 access_token |
| Cloud/Auth 不可达 | 502 | 明确 `auth unreachable` / `cloud unreachable`（勿再写 django） |

---

## 6. 分阶段实施计划

### Phase A — 热路径零 Django（✅ shipped）

1. tcg：`djangoResolveTarget` → `cloudResolveTarget`（接 :8018）  
2. tcg：`djangoValidateSession` → `authValidateSession`（接 :8003 新 API）  
3. taskAuth：实现 validate-session（身份 + 租户成员）  
4. 单测：handlers 用 httptest mock Auth/Cloud，**禁止**再起 Django mock 为热路径依赖  
5. 契约/手工：停 Django，job 提交仍成功  
6. 文档：更新 `machine_container` 旁路说明（若有平台侧章节）

### Phase B — job-stream / relay session（✅ shipped）

1. ✅ `publish-container-job-stream` 迁出 Django（tcg → Kafka `SSE_MESSAGE`）  
2. ✅ `open-runtime-session` → Cloud `POST /api/internal/runtime-session/open/`（tcg + taskEvents）

### Phase C — git-push L3（✅ shipped）

1. ✅ prepare / complete → taskCloudService  
2. ✅ auth-context → Cloud `GET /api/internal/layer-git-push/auth-context`  
3. ✅ `identity_id` 路径：校验 `accounts_user_company_git_identity` + gitOauth 换票（无 501）  
4. ✅ 删除 tcg `djangoGet` / `djangoPost` / `djangoInternalApiBase` 配置

### Phase D — mock-run + 清理（✅ shipped）

1. ✅ `mock-run-container/*` 经 taskGateway → tcg → `go_run_container`  
2. ✅ `installed_image_id` → Cloud resolve-image；env-defaults → Cloud（AI agent + heuristic 回退）  
3. ✅ 删除 Django tcg internal **全部**路由（`urlpatterns = []`）  
4. ✅ taskEvents `WorkflowUpdateHandler` NO-OP（Redis 由 tcg→Cloud upsert）  
5. ✅ auth-context：Task/Project 收集 GitHub+GitLab + summary-for-user readiness 文案对齐 Django

---

## 7. 风险与缓解

| 风险 | 缓解 |
|------|------|
| CloudServerConfig 双真源（Django ORM 仍有行） | Phase A resolve **只读 Go SSOT**；确认 register-reachability 已写 Go；必要时做一次一致性校验脚本 |
| Session Cookie 解析与 Django 不一致 | 复用 taskAuth 已验证的 session/token 解析；加 Cookie+Token 双路径契约测 |
| 租户成员数据仍在 Django | 若 taskAuth 无成员表，短期可：tcg 调公司域 Go；或从 login enrich 缓存；**禁止**为省事回 Django |
| 停 Django 后冷路径（git-push auth-context）失败 | prepare/complete 已迁 Cloud；auth-context 仍 Django，UI 可降级 |
| 日志误判「仍经 Django」 | 统一改 stage 名；Loki 仪表盘过滤 `django_*` 应为 0 |

---

## 8. 测试意图（摘要）

| 用例 | 期望 |
|------|------|
| T1 登录用户提交 container-layer-command | 200；tcg 无 django_* stage |
| T2 停 saas-backend 后重复 T1 | 仍 200（已注册容器） |
| T3 无 Cookie/Token | 401 |
| T4 跨租户 | 403 |
| T5 无 server_url | 409 |
| T6 job 轮询 10 次 | saas-backend 无 validate/resolve 访问；tcg 仅 auth/cloud/upstream |
| T7 relay-to-trae/start（无 CloudServerConfig） | 仍允许（path 豁免） |

完整测试意图见：`docs/intents/backend/container_exec_bypass_django.test-intent.md`

---

## 9. 架构交付物

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v13-application-integration-20260710-1505-claude.puml` |
| ArchiMate | `docs/architecture/v13-application-integration-20260710-1505-claude.archimate` |
| Mermaid | `docs/architecture/v13-application-integration-20260710-1505-claude.mermaid.md` |
| 版本史 | `docs/architecture/VERSION_HISTORY.md`（v13 条目） |

---

## 10. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-10 | 初版：基于 runAll/saas-backend 现象与 tcg `django_validate` 运行时证据，选定 Auth+Cloud 热路径旁路方案 |
