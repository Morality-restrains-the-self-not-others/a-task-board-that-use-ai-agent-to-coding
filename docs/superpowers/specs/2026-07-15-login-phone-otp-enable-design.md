# 设计：登录页电话号码验证码登录可用（含 add_account / next）

**日期**: 2026-07-15  
**类型**: 认证 / 前端 UX / 系统策略  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `login-phone-otp-enable`  
**作者**: claude  
**目标页**: `https://www.daydaymoney.com/auth/login/?add_account=1&next=%2Ftenant%2F…%2Fbilling%2Frecharge%2F`

## 问题分析

线上实测（浏览器）：

- 登录方式仅「邮箱/密码」「访问令牌」，**无**「手机号/验证码」
- `GET /api/public/system-feature-policy/` → `enable_phone_login: false`

代码现状：

- `Login.vue` / `LoginMethodSelector` / taskAuth→Django OTP 链路**已实现**
- `allowPhoneLogin` 由策略门禁；关则隐藏手机登录入口且后端拒绝
- `route.query.next` 仅经 `sanitizeOidcResumeNext`（只接受 OIDC authorize URL），Navbar 传入的业务 path（如 `/tenant/.../billing/recharge/`）被忽略
- 登录成功未调用 `persistLoginAccountSlot`；`?add_account=1` 未按多账号设计 upsert

## 成功标准（SMART）

| # | 标准 | 可验证 |
|---|------|--------|
| S1 | 策略开启后登录页展示「手机号/验证码」并可发码登录 | Vitest + 手工 / Playwright |
| S2 | 部署后全局策略 `enable_phone_login=true`（管理员仍可关） | Django data migration + API |
| S3 | `next` 为同站相对 path 时登录成功后跳转该 path（防开放重定向） | 单元测试 `PostLoginReturnUrl` + Login 解析 |
| S4 | `add_account=1` 登录成功 upsert `savedAccounts`，不清其它槽 | Vitest |
| S5 | **不新增** Python/Go 公网接口；复用现有 OTP / policy API | 设计门禁 |
| S6 | 意图文档与 value-stream 步骤同步 | `docs/intents/` + `conf/value-stream.yaml` |

## 方案对比与选型

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 仅运维手工开 SystemAdmin 开关 | 零代码 | 无法保证目标页可用；next/add_account 仍坏 | 否 |
| B. Data migration 开启 + 修复 Login next/add_account | 小改动、复用现网 OTP | 依赖 SMS 供应商配置 | **采用** |
| C. 去掉策略门禁、默认永远展示手机登录 | UX 简单 | 违背 2026-05-22 策略设计 | 否 |
| D. 本期把 OTP 迁出 Django 到 taskAuth | 符合 Go-first 长期目标 | 范围过大（表+SMS SDK） | 否（后续） |

**选定 B**：保持策略门禁与管理端可关；用迁移把现网全局策略打开；补齐前端 `next` / `add_account`。

## 🐍 Python 新增接口清单与 Go 替代评估

**拟新增接口**：无。

`python_api_approval: n/a`（scoped-down：零新 Python endpoint）。

复用：

- `POST /api/auth/`（phone+code）— taskAuth → Django forward-login
- `POST /api/accounts/users/send_verification_code/`
- `GET /api/public/system-feature-policy/`
- `POST /api/system-admin/system-feature-policy/`（管理员可再关）

## 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | 用户与认证（taskAuth + Django accounts OTP + 前端会话） |
| SystemFeaturePolicy | `enable_phone_login` 全局门禁 |
| PhoneOtpLogin | 手机号 + 6 位验证码登录（可自动注册） |
| PostLoginReturnUrl | 同站相对 path 回跳（已有 VO） |
| AccountSlot | 多账号槽 upsert（已有 `persistLoginAccountSlot`） |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员开启手机登录策略 | SystemFeaturePolicyUpdated（存量） | SystemFeaturePolicyService | — | 复用存量事件 |
| 用户手机验证码登录成功 | UserLoggedInViaPhoneOtp | Django forward-login / LoginSerializer 成功路径 | 审计/下游（若已有 handler） | 存量 OTP 路径；本期不改事件契约 |
| 登录页展示/回跳/多账号槽 | — | — | — | 纯前端 / 策略读；无新服务端状态意图 |
| Data migration 打开开关 | — | — | — | 运维配置迁移；非运行时业务意图 |

## 前端设计

### `resolvePostLoginRedirectUrl` 优先级

1. OIDC resume `next`（`sanitizeOidcResumeNext`）— 保持不变  
2. **同站业务 `next`**（`PostLoginReturnUrl.normalize(route.query.next)`）— **新增**  
3. `localStorage.postLoginRedirect`  
4. `data.redirect_url` / 默认 `/system-admin/`

### 登录成功

- 调用 `persistLoginAccountSlot({ userId, token, username, avatarUrl })`
- `add_account=1`：仅 upsert，不 `localStorage.clear` / 不清其它槽（当前也未 clear，补齐 upsert 即可）

### 策略

- 前端仍读 public policy；migration 打开后入口可见
- SMS 未配置时发码失败由现有错误提示处理（不静默）

## 后端设计

- Django data migration：`SystemFeaturePolicyModel` `singleton_key=global` → `enable_phone_login=True`（行不存在则 `update_or_create`）
- **不改**模型字段 `default=False`（新空库仍安全默认；migration 负责现网打开）
- 无新 API

## 🏛️ 架构变更影响

- **无需**新建 `docs/architecture/` target 文件：无新服务组件、无新 Rel_Flow、无数据所有权变更（仅策略布尔与前端回跳）
- 基线仍为 v27 current；并行 target（v28/v29）不受影响

## 价值流影响（预览）

- `user-auth.phone-otp-login`：已 active；补前端回归测例
- `system-admin-phone-login-recharge-policy`：策略仍可关；迁移将现网打开
- 新增/激活步骤：`login-phone-otp-next-redirect`、`login-add-account-slot-upsert`（前端）

## 安全

- `PostLoginReturnUrl`：仅允许以 `/` 开头的相对 path，拒绝 `//`、绝对 URL、换行（防开放重定向）
- Token 不打日志；SMS 验证码不入前端持久化

## 测试策略

1. Vitest：`resolvePostLoginRedirect` / Login 辅助函数对 tenant recharge `next`
2. Vitest：登录成功调用 `persistLoginAccountSlot`
3. Django：migration 后 policy `enable_phone_login is True`（或 migration 测试）
4. 既有 `test_login_phone_code.py` / policy gate 回归

## 非目标

- OTP 存储/SMS SDK 迁 Go
- 强制开启充值前短信验证（`enable_recharge_phone_verification` 保持原值）
- 改变邮箱/令牌登录
