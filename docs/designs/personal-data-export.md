# 个人信息导出（Personal Data Export）设计

- 日期：2026-08-24
- 状态：已实现
- 目标：补齐 PIPL/GDPR 合规能力 —— 「删除权」已有 15 天冷却注销流程（taskAuth account-deletion），「导出权/可携带权」此前缺失 export 接口，本次新增。

## 1. 背景与问题

- 已有：`taskAuth` 用户自助注销（`/api/accounts/users/me/account-deletion/*`），precheck/status/request/cancel + 内部 execute-due 扫描，15 天冷却后归档 + PII 脱敏。
- 缺失：无任何「导出我的数据」能力，用户无法在注销前取得自己的数据副本。
- 调研发现（一并修复）：网关 `routes.yaml` 从未登记 account-deletion 的 **POST** 路由（`request`/`cancel`），生产中 POST `/api/accounts/users/me/account-deletion/request/` 落入 `api-orphaned`（null-upstream）→ **502**，用户无法从 Web 端提交注销。本次新增导出路由时同族修复（追加 POST 方法路由）。

## 2. 设计决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 导出格式 | 单一 JSON 文件（分 section），`Content-Disposition: attachment` | 结构化、可机器读取；数据量小（KB 级），同步聚合可行 |
| 生成方式 | 请求时同步聚合（本地表 + 3 个内部服务） | 与注销同族、无异步任务栈新增 |
| 有效期 | 生成后 7 天，过期 410 + 新请求重建 | 控制 PII 留存窗口；惰性清理（新请求时删旧行） |
| 鉴权 | 仅本人（requireAuthenticatedUser），无密码二次验证 | 只读操作非破坏性，与注销（破坏性、需密码）区分 |
| 失败降级 | 单个 section 不可达 → 标记 `unavailable`，不阻断整体 | 导出不应因单服务故障整体失败 |
| 数据边界 | 仅个人可归因数据；**不含** password_hash / token 原文 / 激活令牌 / 验证码 | 安全三层边界（见 §6） |
| 存储 | taskAuth MySQL 新表 `auth_personal_data_export`（content MEDIUMTEXT） | 与服务内既有表同库，审计简单 |

## 3. API 契约（taskAuth，与注销同族路由）

`/api/accounts/users/me/personal-data-export/`（GET+POST 均注册，ServeMux 子树）：

### POST `request/`
同步聚合生成导出快照。
- 200/201：`{status, export_id, generated_at, expires_at, sections_ok: [..], sections_unavailable: [..]}`
- 401 未认证

### GET `status/`
最近一次导出状态。
- `{status: "ready"|"expired"|"none", export_id?, generated_at?, expires_at?}`

### GET `download/`
- 200：`application/json; charset=utf-8` + `Content-Disposition: attachment; filename="personal-data-<export_id>.json"` + `Cache-Control: private, no-store`
- 404 无导出；410 `{error:"export_expired"}` 已过期

## 4. 数据源

### 4.1 taskAuth 本地（task_auth 库）
| section | 数据 | 表 |
|---|---|---|
| `account` | user_id, date_joined, last_login, is_active, is_archived, deletion_completed_at | auth_user |
| `profile` | username, avatar | auth_user_profile |
| `login_methods` | method_type, identifier（邮箱/手机号）, is_verified, created_at, binding_voided_at | auth_login_method |
| `access_tokens` | name, last_4, created_at, last_used_at, expires_at, is_revoked（**无原文**） | auth_user_access_token |
| `wechat_identity` | app_key, app_id, openid, unionid, nickname, avatar_url, created_at | wechat_identity |
| `kyc` | tier, status, effective_at, expires_at, risk_flags | auth_kyc_profile |
| `deletion_history` | 最新删除请求 status/requested_at/effective_at/cancelled_at/executed_at + 历史条数 | auth_account_deletion_request |

### 4.2 内部服务契约（taskAuth → 各服务，与 account-deletion-blockers 同模式）
`GET /api/internal/<svc>/users/{user_id}/personal-data/` → `200 {"data": {...}}`
内部密钥头：taskBill 用 `X-TaskBill-Internal-Secret`，tenant/cloud 用 `X-Internal-Secret`（与既有 blockers 调用一致）。

| 服务 | section | 数据 | 实现 |
|---|---|---|---|
| taskBill | `billing` | 该 user_id 的交易/用量明细（各限 200 条，倒序）、引荐关系（referrer/referred）、佣金应计 | 新 `personal_data.go` + 路由分支 |
| taskTenantService | `tenant_memberships` | 成员关系（company_id, company_name, is_admin, is_creator, is_active, workspace_id, created_at） | 复用 `listMembersByUser` 精简 |
| taskCloudService | `cloud_servers` | image_invoker_user_id = 用户的服务器（id, company_id, workspace_id, task_id, comment_id, server_url, last_runtime_status, created_at） | 新 `personal_data.go` + 路由分支 |

失败（网络/5xx/超时）→ `sections_unavailable` 记录 `{section, reason}`，导出仍完成。

## 5. 存储（dataMigrate/taskAuth/042_personal_data_export.sql）

```sql
CREATE TABLE IF NOT EXISTS auth_personal_data_export (
  id BIGINT NOT NULL PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,          -- ready | partial
  content MEDIUMTEXT NOT NULL,          -- JSON 快照
  generated_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_auth_pde_user (user_id),
  INDEX idx_auth_pde_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- 每次新请求前删除该用户旧行（惰性清理，天然限 1 行/用户）。
- 过期判断：`expires_at < NOW()`，下载时 410；status 接口返回 `expired`。
- 可观测：slog 结构化日志（correlation traceId），不含 PII；关键动作：request/generated/download/download_expired。

## 6. 安全边界（security-and-hardening 三层）

1. 输入：user_id 仅来自鉴权上下文（resolveUserIDFromRequest），不信任 URL/body。
2. 输出边界：导出**绝不包含** password_hash、access token 原文、activation_token、password_reset_token、短信验证码、customtoken key；token 类仅 last_4。
3. 传输：内部端点均校验内部密钥；下载响应 `Cache-Control: private, no-store`；鉴权同注销（本人）。
4. 网关：新路由 auth_mode=token（forward-auth），同 taskauth-access-tokens 插件块。

## 7. 网关路由（routes.yaml + 重新生成 apisix.yaml）

```yaml
# 个人信息导出（GET+POST 子树；POST 必须显式路由，GET 原 taskauth-get-user 兜底）
- id: taskauth-personal-data-export
  priority: 849
  uris:
    - /api/accounts/users/me/personal-data-export/
    - /api/accounts/users/me/personal-data-export/*
  upstream: taskAuth
  auth_mode: token

# 修复：注销 request/cancel POST 落 api-orphaned(15) → 502（GET 由 taskauth-get-user 承接）
- id: taskauth-account-deletion-post
  priority: 849
  uris:
    - /api/accounts/users/me/account-deletion/*
  methods: [POST]
  upstream: taskAuth
  auth_mode: token
```

## 8. FE 交互（taskFE）

- 新组件 `UserProfilePersonalDataExportPanel.vue`，置于 UserProfile.vue 注销面板之前。
- 状态机：`none`（显示「导出我的数据」按钮）→ 请求生成（busy）→ `ready`（显示有效期 + 「下载」按钮 + 过期重新生成提示）→ `expired`（提示重新生成）。
- 下载：`apiFetch` → `response.blob()` → `URL.createObjectURL` → `<a download>` 触发（带 Authorization 头路径，不依赖裸导航 cookie）。
- 错误提示遵循全站 `data-traceId` 惯例。

## 9. 验收标准

1. `POST request/` 生成完整 JSON（7 个本地 section + 3 个远端 section），远端失败降级不阻断。
2. `GET download/` 本人可下载、含 `Content-Disposition`；未生成 404；过期 410。
3. `GET status/` 正确反映 ready/expired/none。
4. 导出内容不含任何凭证原文（password hash/token/激活码/验证码）。
5. 网关 POST 注销请求不再 502（同族修复，回归验证）。
6. taskAuth Go 单测全绿；各服务 personal-data 端点单测全绿；FE 组件测试全绿。
7. dataMigrate 042 幂等可重复执行；迁移走既有 apply_datamigrate 通道。
