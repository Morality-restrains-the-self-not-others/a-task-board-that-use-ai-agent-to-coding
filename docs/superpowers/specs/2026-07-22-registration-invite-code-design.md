# 平台注册邀请码（每日限量）— 设计文档

- **日期**: 2026-07-22
- **迭代**: registration-invite-code-daily-quota
- **入口**: `/goal` → 0-auto-flow（跳过 USER GATE）
- **架构版本**: v43（基于 v41 current；v42 仍为其他迭代 target）

## 1. 问题与目标

平台需要可控的新用户增长闸门：管理员开启后，新注册必须持有限量发放的邀请码；每日全局放量可配置；用户可申请与追踪自己发出的码；管理员可审计邀请关系。

### 成功标准（SMART）

1. `enable_registration_invite=true` 时，`email_register` / `phone_register` 无有效码 → 4xx，有码 → 注册成功且码一次性核销。
2. Admin `/system-admin/` 可改开关与 `daily_quota`，并分页查看邀请关系。
3. Profile 可申请码（受当日剩余配额约束）并查看自己的码状态。
4. Login（含 `add_account=1`）与 Register 在开启时展示邀请码字段；Login 写入 `sessionStorage.registration_invite_code`。
5. 日界线：`Asia/Shanghai` 日历日；配额计「当日新发放数」。
6. Go 单测覆盖 T1–T9；OpenAPI + 网关路由同步；意图事件有 publish 点。

## 2. 方案选型（自主决策）

| 方案 | 说明 | 结论 |
|---|---|---|
| A. 扩展 Django SystemFeaturePolicy + Django 表 | 与手机登录开关同页，但违反 Go-first / 单表所有权 | ❌ |
| B. 新建独立 Go 服务 | 边界清晰但过重 | ❌ |
| C. **扩展 taskAuth**（策略+码表+注册校验+用户/Admin API） | 与注册同进程，原子核销简单 | ✅ 采用 |

与现有机制边界：

| 机制 | 用途 | 本设计 |
|---|---|---|
| 公司 Invitation | 入租户 | 不改动 |
| referral `accessCode=u{id}` | 分销绑定 | 不改动；字段名用 `invite_code` 区分 |

## 3. 数据模型（owner: taskAuth / task-auth.db）

### `accounts_registration_invite_policy`（单例）

| 列 | 类型 | 说明 |
|---|---|---|
| singleton_key | TEXT PK | 固定 `global` |
| enabled | INTEGER | 0/1 |
| daily_quota | INTEGER | ≥0，当日最多新发码数 |
| updated_at | TEXT | UTC |
| updated_by | TEXT | user id |

### `accounts_registration_invite_code`

| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | snowflake string |
| code | TEXT UNIQUE | 8 位 `[A-HJ-NP-Z2-9]` |
| issuer_user_id | TEXT | 申请人 |
| status | TEXT | `unused` / `used` / `revoked` |
| issued_day | TEXT | `YYYY-MM-DD`（上海） |
| created_at | TEXT | UTC |
| redeemed_by_user_id | TEXT NULL | |
| redeemed_at | TEXT NULL | |

索引：`(issued_day)`、`(issuer_user_id, created_at)`、`(status)`。

## 4. API（全部 taskAuth + OpenAPI）

| 方法 | 路径 | Auth | 说明 |
|---|---|---|---|
| GET | `/api/public/registration-invite-policy/` | none | `{enabled, daily_quota, remaining_today}` |
| GET/PUT | `/api/system-admin/registration-invite-policy/` | token + superuser | 读/改策略 |
| GET | `/api/system-admin/registration-invite-relations/` | token + superuser | 分页关系列表 |
| POST | `/api/accounts/users/registration-invite-codes/apply/` | token | 申请一码 |
| GET | `/api/accounts/users/registration-invite-codes/` | token | 我的码列表 |
| POST | `/api/accounts/users/email_register/` | none | body 增 `invite_code`（条件必填） |
| POST | `/api/accounts/users/phone_register/` | none | 同上 |

错误码（JSON `error` 字段）：

- `invite_code_required` / `invite_code_invalid` / `invite_code_used` / `invite_code_revoked`
- `invite_feature_disabled` / `daily_quota_exhausted`

鉴权：网关 `auth_mode: token` 注入 `X-User-Id`；handler 信任网关头（直连端口不公网），并 `resolveUserIDFromRequest` Bearer 回退。Admin 额外查 `accounts_super_admin` / `is_superuser`。

## 5. 前端

1. **SystemAdmin.vue**：新区块「注册邀请码」— 开关、每日放量、今日剩余、关系表（链接或内嵌简易表）。
2. **UserProfile**：面板「我的邀请码」— 申请按钮 + 列表（码/状态/使用人/时间）。文件行数门禁：拆 `RegistrationInvitePanel.vue`。
3. **Register.vue**：开启时必填；提交带 `invite_code`。
4. **Login.vue**：开启时展示字段；变更写入 `sessionStorage`；链到注册时携带。

## 6. 领域事件

成功路径经现有 `publishDomainEventKafka`（无 Kafka 时记 warn 不阻断主事务，与 EMAIL_SENT 同类权衡：本设计**优先保证注册成功**，事件失败仅日志；单测可注入 mock）。

## 7. 架构交付物

- `docs/architecture/v43-application-integration-20260722-1148-claude.puml`
- 同名 `.archimate` + `.mermaid.md`
- `VERSION_HISTORY.md` 增 v43 target

## 8. Python 例外

无新增 Django 公网 API。

## 🕸️ Code Review Graph 分析

- **graph_status**: stale / sparse（Nodes≈70，未索引 taskAuth/task2app 主代码）
- **探测路径**: MCP unavailable → CLI `uvx code-review-graph search/status/communities`
- **触点社区 / 关键节点**: CLI 未命中 `email_register` / `SystemFeaturePolicy`；改以仓库探索为准：`taskAuth` 注册、`SystemAdmin.vue`、`Register.vue`、`Login.vue`、`taskGateway/routes.yaml`
- **爆炸半径（文件/函数）**: `handleEmailRegister` / `handlePhoneRegister`、`mountRoutes`、OpenAPI YAML、网关 auth 路由、SystemAdmin / Profile / Register / Login
- **与 Archimate 对照**: 将新增 taskAuth 数据对象与 Vue→GW→taskAuth 流；一致目标写入 v43
- **风险与约束**: 图未覆盖导致影响分析依赖人工；日配额并发需 DB 事务；勿与 referral `accessCode` 混字段
- **对本次设计的影响**: 落点锁定 taskAuth；实现期 Step 8 对改动文件再跑 `detect-changes` / `impact`
