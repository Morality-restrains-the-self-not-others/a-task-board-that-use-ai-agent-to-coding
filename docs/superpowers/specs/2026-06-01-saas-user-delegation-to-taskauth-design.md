# saas 用户身份委托 taskAuth 架构设计

> 日期：2026-06-01  
> 状态：已批准 — Inc-1 ✓～Inc-5 ✓（2026-06-01）；value-stream / Playwright 租户 ID 收口待验证  
> 触发：双库 `accounts_user` 同步漂移（`sync-user` / `resolve_business_user_id` 缺陷）；用户确认 saas 不应保留 `accounts_user`，需 userId 时引用 taskAuth；`accounts_super_admin` 迁入 auth.db。  
> 业务字段策略：**C — 拆散到各域**（头像 → profile 表；当前工作区 → session / CompanyMember；凭证/激活态 → taskAuth API 按需拉取）

---

## 1. 问题陈述

### 1.1 现状与痛点

| 现象 | 根因 |
|------|------|
| 登录后 enrich-login / 业务 API 偶发 404 | auth.db 与 saas `accounts_user` 行不一致；`resolve_business_user_id` 仍查已删除的 default `accounts_login_method` |
| DB reset 后 URL 中 tenant/user 失效 | 业务 ID 仅在 saas 重生；浏览器/测试硬编码旧 Snowflake ID |
| 双写维护成本高 | 注册路径：taskAuth 写 auth.db → `sync-user` 复制到 saas；两库各一份 `accounts_user` |

### 1.2 目标架构（用户确认）

- **taskAuth auth.db**：用户身份与凭证的**唯一写真源**（含 `accounts_user`、`login_method`、`customtoken`、`super_admin`）
- **saas default**：**不再存在** `accounts_user` 物理表；业务表以 `user_id: string` 引用用户，必要时调用 taskAuth HTTP API
- **业务画像拆域**（策略 C）：
  - 头像 → saas `user_profile`
  - 租户内当前工作区 → `CompanyMember.workspace_id` + Django session
  - 邮箱/手机/激活态/超管标记 → taskAuth API 或 token 校验附带 claims

### 1.3 与 2026-05-31 已批准设计的差异

| 项 | 2026-05-31 方案 A | 本设计 |
|----|-------------------|--------|
| saas `accounts_user` | 保留（FK 锚点） | **删除** |
| `accounts_super_admin` | 留 saas（MTI） | **迁入 auth.db** |
| 用户同步 | `sync-user` 双库复制 | **取消**；saas 只存 `user_id` |
| 用户读 | Django ORM + router | **taskAuth API** + 本地 profile |

本设计为**有意识的架构演进**，Increment 7 的「default 保留 User FK」约束将被废弃。

---

## 2. 领域概念清单（供 `/5-ddd` 输入）

| 限界上下文 | 关键实体 | 聚合根 | 跨上下文事件 |
|------------|----------|--------|--------------|
| **Auth（taskAuth）** | User, LoginMethod, CustomToken, SuperAdmin | User（凭证 + 角色） | USER_CREATED, USER_ACTIVATED（已有） |
| **Tenant（saas）** | Company, CompanyMember, UserProfile | Company | COMPANY_CREATED, MEMBER_JOINED |
| **Workspace（saas）** | Workspace | Workspace | — |
| **Legal（saas）** | PrivacyConsent, LicenseConsent | —（按 user_id 关联） | — |
| **Integration（taskEvents）** | DomainEventConsumer, UserCreatedHandler | UserCreated → Company | USER_CREATED, COMPANY_CREATED |
| **Billing（taskBill）** | BillingTransaction, BillingUsage | Account | —（引用 user_id string） |

**Identity 规则**：全链路 `user_id` 为 string Snowflake ID；仅在 taskAuth 写库 boundary 转 DB 类型（见 `11_id_field_string_transit.md`）。

---

## 3. 价值流影响（摘要）

完整切片由 `/3-value-stream-价值流` 负责；本设计识别以下影响面：

| 价值流 | 受影响 step | 变更要点 |
|--------|-------------|----------|
| **user-auth** | email-register, phone-register, login, activate, frontend-auth-guard, auth-table-cleanup, dev-bootstrap-admin | 删除 `saas-backend.accounts_user.*` 字段；新增 `task-auth.accounts_super_admin.*`；profile 改读 API + `user_profile` |
| **company-management** | member-crud, user-company-assoc | `CompanyMember.user` FK → `user_id`；`workspace_id` 承担租户内当前工作区 |
| **task-management / cloud-integration / billing** | 凡引用 `AUTH_USER_MODEL` 的 step | FK → `user_id` string；创建者/操作者展示名按需 batch-resolve taskAuth |
| **dev-platform-reset** | dev-bootstrap-admin | init 仅 bootstrap auth.db；saas 不再 sync SuperAdmin 行到 default User |
| **domain-events（taskEvents）** | user_created/0_create_company | handler 写 saas `company` + `company_member.user_id`；**不** upsert `accounts_user`；`user_id` 全程 string |
| **billing（taskBill）** | charge / usage 相关 step | 已用 `user_id varchar`；Inc-5 校验与 saas 删表后契约一致 |

**新增 step（建议）**：

- `user-auth.taskauth-user-resolve-api` — saas 调用 taskAuth 读用户与超管校验
- `user-auth.django-admin-taskauth-token` — `/admin/` 登录与会话走 taskAuth token

**废弃 step 字段**：所有 `saas-backend.accounts_user.*`（迁移完成后从 value-stream.yaml 移除）。

**明确废弃**：saas 侧 Kafka/Python 消费者（`manage.py start_kafka_consumer` 仅保留兼容壳或删除）；所有 handler 归 taskEvents intent 二进制管理。

---

## 4. 方案对比

### 方案 A — API 委托 + 域内 profile（**推荐**）

```
taskAuth auth.db          saas default
─────────────────         ─────────────────────────────
accounts_user (真源)      user_profile(user_id, avatar)
login_method              company_member(user_id, workspace_id, …)
customtoken               projects_*(creator_user_id, …)
accounts_super_admin      (无 accounts_user)
        ▲
        │ GET /api/accounts/users/{id}/
        │ POST /api/internal/users/batch-resolve/
        └──────── saas-backend
```

| 维度 | 评价 |
|------|------|
| 与用户需求 | ✅ 完全一致 |
| 一致性 | 单写真源，无 sync-user 漂移 |
| 性能 | 热路径可缓存 + batch API；比双库 sync 更简单 |
| Django 改造量 | 大（AUTH_USER_MODEL 重构） |
| 风险 | 需分 increment 迁移，不能 Big Bang |

### 方案 B — saas 只读 router 直连 auth.db（无 HTTP）

saas 通过 `DATABASE_ROUTERS` 读 auth.db 的 `accounts_user`，不写、不调 API。

| 维度 | 评价 |
|------|------|
| 与用户需求 | ❌ 用户明确要求 API |
| 耦合 | saas 与 auth SQLite 路径/ schema 强耦合 |
| 性能 | 最低延迟 |
| 多实例 | auth.db 文件锁/副本问题 |

### 方案 C — 保留 saas 最小 User 桩表（仅 id 列）

过渡态：`accounts_user(id)` 无业务字段，仍满足 Django FK。

| 维度 | 评价 |
|------|------|
| 与用户需求 | ⚠️ 部分满足（表仍存在） |
| 迁移难度 | 低于方案 A |
| 终点 | 仍需第二轮删表 |

**推荐方案 A**，按 increment 交付；方案 C 仅作可选过渡 increment（见 §6.2）。

---

## 5. 目标架构详设

### 5.1 auth.db 表归属（To-Be）

| 表 | 位置 | 说明 |
|----|------|------|
| `accounts_user` | auth.db | 真源：id, password, is_active, is_superuser, is_staff, date_joined, last_login |
| `accounts_login_method` | auth.db | 已有 |
| `accounts_customtoken` | auth.db | 已有 |
| `accounts_super_admin` | auth.db | **新增**；`user_id` PK/FK → accounts_user.id（Go 侧表，非 Django MTI） |
| `django_content_type` | auth.db | 已有 |

**SuperAdmin 语义**：`accounts_super_admin.user_id` 存在即超管；`accounts_user.is_superuser=1` 与之同步（bootstrap 时双写，日常以 super_admin 表为准）。

### 5.2 saas 域内用户相关表（To-Be）

| 表 | 字段 | 说明 |
|----|------|------|
| `accounts_user_profile`（新） | `user_id` PK, `avatar`, `updated_at` | 全局头像；与租户无关 |
| `accounts_company_member` | `user_id`（替换 FK）, `company_id`, `workspace_id`, `member_avatar`, … | `workspace_id` = **该租户下**用户当前工作区；session 存 `(tenant_id → workspace_id)` 作 UI 默认 |
| 业务表（projects, cloud, …） | `*_user_id` CharField | 无 DB 级 FK 到 User；应用层校验 user 存在（taskAuth batch-resolve） |

**删除**：`accounts_user`、`accounts_super_admin`（saas 侧）。

### 5.3 taskAuth 新增 / 调整 API

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/accounts/users/{user_id}/` | 公开字段：id, is_active, is_superuser, email/phone（脱敏策略可配置） |
| POST | `/api/internal/users/batch-resolve/` | saas internal：批量解析 user_id → display label（email 前缀/手机尾号） |
| GET | `/api/internal/users/{user_id}/super-admin/` | saas 管理端：是否超管 |
| — | 删除 saas `sync-user` 调用方 | 注册/登录后不再复制 User 行 |

**鉴权**：用户-facing 读接口需 Token；internal 接口 `X-TaskAuth-Internal-Secret`（与现有 internal 一致）。

**响应示例**（ID 均为 string）：

```json
{
  "id": "849095291104686080",
  "is_active": true,
  "is_superuser": false,
  "email": "contact@daydaymoney.com",
  "login_methods": [{"method_type": "email", "identifier": "…", "is_verified": true}]
}
```

### 5.4 saas Django 认证模型改造

**现状**：`AUTH_USER_MODEL = accounts.User`；`request.user` 为 saas ORM 行。

**To-Be**：

1. 引入 **`AuthPrincipal`**（非 DB Model 或 unmanaged stub）：仅 `user_id: str`、`is_superuser: bool`、`is_active: bool`。
2. **`CustomTokenAuthentication`**：token 校验仍在 auth.db；加载 Principal，**不**查 saas User 表。
3. **`enrich-login`**：session 绑定 `user_id` + Principal；不再 `django_auth_login(User)`。
4. **Profile API**（`/api/accounts/users/profile/`）：组装响应：
   - 身份段：taskAuth GET user（或登录响应缓存）
   - 画像段：`UserProfile` + `CompanyMember` 列表
   - 工作区段：`CompanyMember.workspace_id` 或 session
5. **权限**：`requiresAdmin` 路由守卫 → taskAuth internal super-admin 或 token claims 中 `is_superuser`。

### 5.4.1 Django Admin（/admin/）— 已确认

**要求**：保留 `/admin/`；Admin 会话**与业务 API 相同**，走 taskAuth token，不维护 saas 本地 User 密码会话。

| 项 | To-Be |
|----|-------|
| 登录页 | Admin login form POST → taskAuth `/api/accounts/users/login/`（或专用 admin login delegate） |
| 会话 | `session['taskauth_token']` + `session['user_id']`；`ModelBackend` 替换为 **`TaskAuthAdminBackend`** |
| `request.user` | `AuthPrincipal`（同 API）；Admin 权限：`is_staff` / `is_superuser` 来自 token 校验 + auth.db `accounts_super_admin` |
| ORM Admin 注册 | `accounts/admin.py` 移除 `User` / `SuperAdmin` / `LoginMethod` 注册；保留 Company、Project 等业务模型 |
| 展示外键 user | `list_display` 中 user 列通过 `TaskAuthUserResolver` batch-resolve 显示 email 前缀 |

**Inc-4 交付物**：`TaskAuthAdminBackend`、Admin login 模板改造、移除对 saas `User` 模型的 Admin 依赖。

### 5.5 当前工作区（策略 C 细化）

| 层级 | 存储 | 行为 |
|------|------|------|
| 租户默认 | `CompanyMember.workspace_id` | 用户在某 company 下上次选择的工作区 |
| 会话 | `session['workspace_by_tenant'][tenant_id]` | 当前浏览器 tab 上下文；切换 tenant 时读取 member 默认 |
| 废弃 | `User.current_workspace_id` | 迁移后删除列 |

现有 `utility_views` 写 `request.user.current_workspace_id` 改为写 `CompanyMember.workspace_id`（需 tenant 上下文）。

### 5.6 头像（策略 C 细化）

| 项 | 说明 |
|----|------|
| 存储 | `accounts_user_profile.avatar`（ImageField） |
| 租户内头像 | 已有 `CompanyMember.member_avatar` — 保持不变 |
| API | `UserProfileViewSet` 或扩展现有 profile PATCH |
| 展示 | `UserSerializer.avatar_url` 读 profile 表，不再读 User.avatar |

### 5.7 取消 sync-user 与修复漂移

| 机制 | 动作 |
|------|------|
| `POST /api/internal/taskauth/sync-user/` | **废弃**（increment 6 删除） |
| `resolve_business_user_id` | **删除**或改为 no-op（无双库可对齐） |
| `mirror_user_to_auth_db` | **删除** |
| 注册/登录 | taskAuth 写 auth.db 即可；saas 仅在创建 CompanyMember 时写入 `user_id` |

### 5.8 数据迁移策略

1. **Backfill**：从 saas `accounts_user` 导出 `id → user_profile`；`CompanyMember.user_id = str(user.id)`。
2. **Dual-read 窗口**（可选过渡）：FK 与 `user_id` 并存一个 increment，验证一致后 drop FK。
3. **Cutover**：停写 saas User；删表；更新 init/migrate 脚本。
4. **验证**：`verify_no_saas_accounts_user.sh` — saas.sqlite3 不得含 `accounts_user` / `accounts_super_admin`。

### 5.9 领域事件与消费者（已确认）

**原则**：saas-backend **不运行**领域事件消费者；消费者统一由 **taskEvents** intent 进程管理（与现有 G5 / `DOMAIN_EVENTS_CONSUMER=go` 方向一致）。

```
taskAuth 注册/激活
    → Django internal post_register 发 USER_CREATED（producer only）
    → Kafka/Redis
    → taskEvents user_created/0_create_company
           → saas.Repository.CreateCompanyForUser(user_id, username)
           → 写 company + company_member.user_id（无 accounts_user）
    → taskEvents user_created/1_send_welcome_email（可选 intent）
```

| 组件 | 角色 | 变更 |
|------|------|------|
| saas `post_register` / `post_activate` | **Producer** | 继续 `send_event`；**删除**任何「upsert local User」逻辑 |
| saas `start_kafka_consumer` | **废弃** | 文档标记 legacy；runAll 不启动 |
| taskEvents `usercreated.Handler` | **Consumer** | `CreateCompanyForUser` 改为纯 `user_id` string；不假定 saas 存在 User 行 |
| taskEvents saas Repository | 写库 | `creator_id` / `member.user_id` 为 string；SQL 不 JOIN `accounts_user` |

**USER_CREATED payload**（保持）：

```json
{
  "user_id": "849095291104686080",
  "username": "example-user",
  "email": "contact@daydaymoney.com",
  "is_active": false
}
```

**注意**：`usercreated/handler.go` 当前 `int64Field(user_id)` 与 ID string transit 规范冲突 — 纳入 Inc-5 改为 string 解析。

### 5.10 taskBill / taskEvents 纳入 Inc-5（已确认）

Inc-5 除 saas 删表外，同步完成：

| 服务 | 库/代码 | 动作 |
|------|---------|------|
| **taskBill** | `billing.sqlite3` | 已 `user_id varchar(36)` — 审计无 FK；补充契约测试：charge API 接受 string user_id，不依赖 saas User 表 |
| **taskEvents** | `internal/repository/saas` | 所有 SQL 去除对 `accounts_user` 的 JOIN/INSERT；`CreateCompanyForUser(userID string, …)` |
| **taskEvents** | handlers（usercreated、welcome 等） | event payload / repository 边界统一 string user_id |
| **registry** | `db/registry.yaml` migrate/init | 与 saas 删表同一 dev reset 批次验证 |

---

## 6. 分阶段交付（Increments）

### Inc-1 — taskAuth 读 API + SuperAdmin 表（auth.db）

- Go：`accounts_super_admin` 表 + bootstrap 写入
- Go：GET user + batch-resolve + super-admin internal
- 测试：`bootstrap_admin_test.go`、HTTP 契约测试
- **不删** saas User（并行）

### Inc-2 — saas `user_profile` + 工作区迁到 CompanyMember

- 新建 `UserProfile`；迁移 avatar
- `current_workspace_id` → `CompanyMember.workspace_id` + session 结构
- Profile API 双读（User + profile）→ 单读 profile

### Inc-3 — 业务表 `user_id` 字符串化（第一批）

- `CompanyMember`, `Company.creator`, comments, workspace_access 等高频表
- 引入 `TaskAuthUserResolver` 服务（HTTP client，`trust_env=False`）

### Inc-4 — Django 认证切 Principal + Admin taskAuth 会话

- Token/session 不再依赖 saas User 行
- **`TaskAuthAdminBackend`**：`/admin/` 登录走 taskAuth token
- 删除 `sync-user` / `resolve_business_user_id` / `mirror_user_to_auth_db`
- System-admin 前端路由 + Django Admin 权限均读 auth.db super_admin / token claims
- saas Admin 注销 `User` / `SuperAdmin` / `LoginMethod` ModelAdmin

### Inc-5 — 全 monorepo user 引用收口 + 删 saas User 表

- saas：projects/cloud/legal 等剩余 FK → `user_id` string
- **taskEvents**：repository + handlers string user_id；USER_CREATED 链不 touch accounts_user
- **taskBill**：契约测试与文档对齐（已无 User FK）
- Drop saas `accounts_user`、`accounts_super_admin`
- 更新 value-stream.yaml、init 流水线、Playwright 租户 ID 策略（env 或 init 输出）
- `verify_no_saas_accounts_user.sh` + taskEvents integration 全绿

### Inc-6（可选过渡）— 方案 C 桩表

若 Inc-4 风险过高：临时保留 `accounts_user(id)` 空表满足 Django 未改完的 FK，Inc-5 再删。

---

## 7. 错误处理与 NFR

| 场景 | 行为 |
|------|------|
| taskAuth 不可达 | 读用户：503 + `{detail: "authentication service unavailable"}`；写业务：依赖 user_id 的创建可继续（仅校验 id 格式），展示名降级为 user_id 前缀 |
| 无效 user_id | batch-resolve 返回 `found: false`；创建 Member 前必须 resolve 成功 |
| 缓存 | saas 进程内 TTL 缓存（60s）batch-resolve 结果；登录时刷新 |
| 性能 | 列表页 N+1：强制 batch-resolve，禁止逐条 GET |

---

## 8. 测试策略

| 层级 | 内容 |
|------|------|
| taskAuth | user read API、super_admin bootstrap、auth.db schema 迁移 |
| taskEvents | USER_CREATED 链 string user_id；registration_chain integration |
| saas 单元 | `TaskAuthUserResolver` mock、Principal auth、UserProfile、`TaskAuthAdminBackend` |
| saas 集成 | 注册→创建公司→member.user_id；profile 无 saas User 表 |
| 回归 | 原 `user-auth` 流测试改断言字段；新增 `test_no_saas_accounts_user_table.py` |
| E2E | Playwright 租户 ID 从 init 输出 / env 读取，去除硬编码 827923… |

---

## 9. 已确认决策（2026-06-01）

| # | 问题 | 决策 |
|---|------|------|
| 1 | Django Admin | **保留** `/admin/`；Admin 会话**走 taskAuth token**（见 §5.4.1） |
| 2 | Kafka USER_CREATED | saas **不做消费者**；消费者统一在 **taskEvents**；saas 仅 producer + 可选 `user_profile` 占位（见 §5.9） |
| 3 | taskBill / taskEvents | **纳入 Inc-5**，与 saas 删表同一批次（见 §5.10） |

---

## 10. 成功标准

- [x] saas.sqlite3 **无** `accounts_user`、`accounts_super_admin` 表（测试库 + verify 脚本）
- [x] auth.sqlite3 **有** `accounts_super_admin`
- [x] 注册/登录 **无** `sync-user` 调用
- [x] `/api/accounts/users/profile/` 与 **`/admin/`** 在无 saas User 表时正常工作（token 会话）
- [x] saas **无** Kafka 消费者进程（taskEvents intent 独占）
- [x] USER_CREATED → taskEvents 创建 company/member，**不** upsert accounts_user
- [x] taskBill / taskEvents 全链路 **string user_id**
- [x] `CompanyMember` 使用 `user_id` string，工作区来自 member/session
- [x] 头像读写走 `user_profile`
- [x] value-stream `user-auth` 无 `saas-backend.accounts_user.*` 字段引用
- [x] 现有 user-auth pytest + taskEvents registration_chain 绿

---

## 11. 变更日志

- 2026-06-01：§9 三项决策写入 — Admin taskAuth token；消费者归 taskEvents；Inc-5 扩至 taskBill/taskEvents
- 2026-06-01：初稿 — 用户确认策略 C；推荐方案 A（API 委托 + 域内 profile）；六 increment 路线图
