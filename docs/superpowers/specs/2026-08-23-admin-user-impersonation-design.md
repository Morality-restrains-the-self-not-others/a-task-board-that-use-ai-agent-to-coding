# 设计：系统管理员以目标用户角色登录（Impersonation）

- **Date:** 2026-08-23
- **Iteration:** admin-user-impersonation
- **Status:** accepted（goal-mode 自动采用）
- **ADR:** [ADR-0037](../../adr/0037-admin-user-impersonation.md)
- **页面:** `https://www.daydaymoney.com/system-admin/users/` 编辑用户模态框

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 成功。当前图 `Languages: javascript, typescript, python, bash`，**108 nodes / 17 files，不含 Go `taskAuth/` 与 Vue `SystemAdminUsers.vue`**。

`CRG unavailable for Go/Vue impersonation path: graph has no taskAuth or SystemAdminUsers nodes.`

设计基于源码阅读：

- 权限码 `user:impersonate` 已在 `shareLib/authz/permissions.go`、`roles.go`、`dataMigrate/taskAuth/027_rbac_roles.sql`、前端 `usePermissions.js` 定义；**无 API / 无 UI**。
- 用户编辑页：`taskFE/app/src/views/SystemAdminUsers.vue` 编辑模态框 `form > div.space-y-4`（用户给出的选择器）。
- 用户管理 API：`taskAuth/src/handlers_system_admin.go`，整段 `requireSuperuser`（`platform:manage` 或 `is_superuser`）。
- 会话：`auth_customtoken` 一用户一登录令牌；`activate-session` 种 HttpOnly `token`/`userId`；SPA 另有 `savedAccounts` 槽。
- APISIX：`/api/system-admin/users/*` 已指向 taskAuth，新子路径无需新路由条目。

## 当前架构理解

- 企业景观 / 应用集成基线：**v101 current**（runAll 编排器独立）。
- 认证真源：taskAuth；平台权限由 PDP 经 APISIX forward-auth 注入 `X-User-Roles`。
- 系统管理用户页仅超管可进；`employee` 角色虽有 `user:impersonate`，当前进不了该页。

本次需求将在此基础上：在编辑用户表单增加「以该用户身份登录」按钮，经 `user:impersonate` 权限控制，签发**独立模拟会话**进入目标用户身份，并允许退出恢复管理员会话。

## 问题

运维/超管排查租户侧权限、工作台、计费展示时，无法在不索要用户密码的前提下以该用户角色查看真实 UI。需求是模拟登录，不是改密码、不是共用用户持久 token。

## 决策（采用方案）

**独立模拟会话 + 平台权限码 `user:impersonate` + 可退出恢复。**

| 项 | 决策 |
|----|------|
| 落点 | 扩展现有 Go `taskAuth`（会话/用户 owner） |
| 开始 | `POST /api/system-admin/users/{id}/impersonate/` |
| 退出 | `POST /api/auth/impersonation/stop/` |
| 状态 | `GET /api/auth/impersonation/status/`（横幅） |
| 权限 | `authz.RequirePlatformPerm(PermUserImpersonate)`；**不**复用整页 `requireSuperuser` |
| 会话 | 新表 `auth_impersonation_session`，独立 `token_key`；**禁止**复用目标用户 `auth_customtoken` |
| 恢复 | HttpOnly cookie `impersonatorRestore` 保存管理员原 token；退出时还原 |
| TTL | 默认 1 小时；过期视同未登录模拟态 |
| 前端 | 编辑模态框按钮（`hasPlatformPerm('user:impersonate')`）；Navbar 模拟横幅 + 退出 |

### 安全闸门

1. 目标必须 `is_active` 且未归档。
2. 禁止模拟自己。
3. 禁止嵌套模拟（当前 token 已是模拟会话则 409）。
4. 目标持有 `platform:manage` / `is_superuser` 时，操作者必须也有 `platform:manage`（防员工提权）。
5. 审计：结构化日志 `event=impersonation_started|stopped`（只打 user id，不打 token）+ Kafka 事件。
6. 幂等：`Idempotency-Key` 与 `(actor_id, target_id, open session)` 同粒度；重放返回同一会话。

### 前端行为

1. 编辑表单底部主操作旁增加「以该用户身份登录」；无 `user:impersonate` 不渲染。
2. `createClickGuard` + 同一次意图固定 `Idempotency-Key`。
3. 成功：`sessionStorage` 备份当前账号槽 → `persistLoginSuccessCredentials`（走 activate-session 种 cookie）→ `location.href = redirect_url`（沿用目标用户登录落地，一般为工作台/onboarding，**不是**系统管理页）。
4. 全站 Navbar：若 status 为模拟中，展示「正在以 {username} 的身份查看」+「退出模拟登录」。
5. 退出：stop API → 恢复槽与 cookie → 跳回 `/system-admin/users/`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员开始模拟登录 | UserImpersonationStarted | ImpersonationAppService | 审计/告警（本期无自动消费者，DLT 仍按规范预留 topic） | — |
| 管理员结束模拟登录 | UserImpersonationStopped | ImpersonationAppService | 同上 | — |
| 浏览用户编辑表单 | — | — | — | 纯只读 UI |
| 查询模拟状态 | — | — | — | 纯查询 |

Topic：`user-impersonation-started` / `user-impersonation-stopped`（及 `-dlt`）。

## 领域概念清单

- **Bounded Context:** Auth（taskAuth）
- **Aggregate:** ImpersonationSession（根：session id；不变式见上）
- **Entities:** ActorUser, TargetUser（既有 User）
- **Domain events:** UserImpersonationStarted, UserImpersonationStopped
- **VO:** ImpersonationToken, SessionTTL

## 价值流影响

影响既有 `user-auth` 流，新增步骤 `admin-impersonate` / `admin-stop-impersonation`。不新增独立域。字段：`task-auth.auth_impersonation_session.*`。

## API 契约

### POST `/api/system-admin/users/{id}/impersonate/`

Auth：已登录 + `user:impersonate`。Header：`Idempotency-Key` 必填。

成功 200：与登录同形 `{ token, user, redirect_url }` + `impersonation: { actor_user_id, target_user_id, expires_at }`。Set-Cookie：`impersonatorRestore`（管理员原 token）。

错误：401 / 403 无权限 / 404 用户不存在 / 409 嵌套或自己 / 422 目标不可登录。

### POST `/api/auth/impersonation/stop/`

Auth：当前凭据必须是未过期模拟会话。200：管理员 `{ token, user, redirect_url }`（`/system-admin/users/`）。清除模拟 cookie，恢复管理员 cookie。

### GET `/api/auth/impersonation/status/`

Auth：已登录。`{ impersonating: bool, actor_user_id?, target_user_id?, target_username?, expires_at? }`。

## Python 新增接口

不新增 Python 接口。`python_api_approval: scoped-down`。

## 🏛️ 架构变更影响

- **迭代版本:** v102 🎯 target
- **迭代名称:** admin-user-impersonation
- **作者:** cursor
- **设计日期:** 2026-08-23 06:20
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v102-enterprise-landscape-20260823-0620-cursor.puml`
  - 🆕 `docs/architecture/v102-application-integration-20260823-0620-cursor.puml`
  - 🆕 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细:** 🟢 模拟会话表与事件；🟡 taskAuth / taskFE / forward-auth 注入 `X-Impersonator-Id`

## 冷热分离（新表）

`auth_impersonation_session` 为时间累积型审计会话。预估年增量 **< 10 万**。策略 B：热表 + 90 天外可归档；主键 Snowflake `id`（不分片，无 AUTO_INCREMENT）。本期只建热表 + `expires_at`/`ended_at` 索引；归档任务记 OPT。
