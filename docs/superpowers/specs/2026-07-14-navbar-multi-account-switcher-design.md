# 设计：导航栏多账号切换（Navbar Multi-Account Switcher）

**日期**: 2026-07-14  
**类型**: 认证 / 前端 UX  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `navbar-multi-account-switcher`  
**作者**: claude

## 目标

在 `https://www.daydaymoney.com/user/{id}/profile/` 等已登录页面，导航栏顶部昵称位置支持下拉：

1. **同时保留多个已登录账号**（本机浏览器会话槽）
2. **仅一个账号为当前激活态**，所有业务 API / 页面以其身份运行
3. 可一键切换激活账号、添加新账号、从列表移除、退出当前

## 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 昵称区域单击展开下拉、双击收起（符合前端下拉规范） | Vitest + 手工 |
| S2 | 下拉列出本机已保存账号；当前激活项有明确标识 | UI 断言 |
| S3 | 切换账号后 `userId` cookie + `authToken` 指向目标账号，页面以新身份工作 | 单元/集成 |
| S4 | 「添加账号」进入登录页且不丢已有槽；登录成功 upsert 槽 | 登录流测试 |
| S5 | 最多 5 个账号槽；超限提示并阻止新增 | 单元测试 |
| S6 | `POST /api/accounts/users/activate-session/` 落 **taskAuth (Go)**，Swagger 可见 | Go 测试 + schema |
| S7 | 无效/过期 token 切换失败时从槽移除或标记需重新登录，不静默成功 | 单元测试 |

## 现状摘要

- 前端：Vue 3 `Navbar.ui.vue` 昵称为 `router-link`，无账号下拉
- 会话：单层 `userId` cookie + `localStorage.authToken` +（服务端）Django session（enrich-login 内部建）
- 无多用户账号切换；仅有公司切换器 / 工作空间切换器
- API 主鉴权：`Authorization: Token …`（`CustomTokenAuthentication`）

## 方案对比与选型

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 纯前端换 token/cookie，无新 API | 改动最小 | 无法校验 token 是否仍有效；无 enrich 自愈 | 否 |
| B. 客户端账号槽 + Go `activate-session` | 校验 token、复用 enrich-login、Go-first | 需新 endpoint | **采用** |
| C. 服务端多 session 表 | 跨设备一致 | 范围大、改 cookie 模型 | 否（本期不做） |

**选定 B**：本机 `localStorage` 保存多账号槽（含各账号 token）；切换时调用 taskAuth 激活接口校验并 enrich，再写回激活凭据并整页刷新。

## 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | 用户与认证（taskAuth + 前端会话） |
| AccountSlot（值对象集合） | 本机保存的 `{userId, username, avatarUrl, token, addedAt}` |
| ActiveAccount | 当前激活槽 = cookie `userId` + `authToken` |
| ActivateSession | 用已有 token 重新建立服务端会话并返回用户信息 |
| Domain Events（可选本期） | 无跨服务事件；前端 `account-switched` 仅窗口内 |

## 接口设计（Go / taskAuth）

### `POST /api/accounts/users/activate-session/`

- **归属**: taskAuth（扩展现有服务）
- **鉴权**: `Authorization: Token <target_account_token>`（要激活的账号 token，不必是当前 cookie 用户）
- **请求体**: `{}` 或可选 `{ "user_id": "<期望 userId>" }`（若提供则须与 token 解析出的 userId 一致，防串号）
- **成功 200**: 与登录类似 `{ "user": {...}, "token": "<same or refreshed>", "redirect_url": "..." }`
- **行为**:
  1. 解析并校验 Token → userId
  2. 用户须 `is_active`
  3. 可选校验 body.user_id
  4. `djangoEnrichLogin`（复用现有 internal enrich-login，含公司自愈）
  5. 返回 user + token（**不**删除其他账号的 token）
- **错误**: 401 无效/过期；403 禁用；400 user_id 不匹配
- **Swagger**: OpenAPI schema 同步

**不新增 Python 公网接口**（`python_api_approval: n/a`）。

## 前端设计

### 存储

- Key: `savedAccounts`（JSON 数组）
- 结构: `{ userId: string, username: string, avatarUrl: string|null, token: string, addedAt: number }`
- 上限: **5**
- 与现有 `authToken` / `userId` 共存：激活账号的 token 仍写入 `authToken`，userId 写入 cookie

### 模块

| 文件 | 职责 |
|------|------|
| `domain/auth/services/saved_accounts_store.js` | CRUD、上限、upsert、remove、list |
| `domain/auth/services/activate_session_service.js` | 调 activate-session、写凭据 |
| `components/AccountSwitcherDropdown.vue`（或 Navbar 内拆分） | 下拉 UI（单击开/双击关） |
| `Navbar.logic.vue` / `Navbar.ui.vue` | 接入下拉；替换纯 router-link 昵称 |
| `Login.vue` | 登录成功 upsert；`?add_account=1` 不强制清其它槽 |

### UX

1. 触发器：头像 + 昵称 + 小三角
2. 菜单项：各账号（头像/昵称；当前 ✓）；「添加账号」；「个人资料」；「退出当前」
3. 切换中显示 loading，成功后 `window.location` 刷新（保留 path 时替换 `/user/{oldId}/` → `/user/{newId}/`）
4. 「添加账号」→ `/auth/login/?add_account=1&next=...`
5. 「退出当前」→ 现有 logout + 从槽移除当前；若还有其它槽则自动 activate 第一个，否则跳转登录页

### 下拉交互

- 单击展开；双击收起（`.ai/04_frontend_development`）
- 点击外部关闭

## 安全与合规

- Token 存 localStorage：与现状一致；XSS 风险不扩大面，但多账号放大泄露面 → 上限 5 + 退出即删槽 token
- 切换必须经服务端校验，禁止仅改 cookie 冒充
- 日志：activate-session 记 userId（脱敏 token）；禁止打出完整 token
- 跨域：沿用现有 API 基地址与 credentials；不新增异域 Cookie 名

## 价值流影响（预览）

- 影响域：`user-auth`
- 新增步骤：`multi-account-slot`、`activate-session-switch`
- 字段：`task-auth.accounts_customtoken.key`、`task-auth.accounts_user.id`（复用）
- 测试：前端 Vitest + taskAuth Go test；可选 Playwright 冒烟

## 🏛️ 架构变更影响

- **迭代版本**: v24 🎯 target
- **迭代名称**: navbar-multi-account-switcher
- **作者**: claude
- **设计日期**: 2026-07-14 15:26
- **新增文件**:
  - `docs/architecture/v24-application-integration-20260714-1526-claude.puml`
  - `docs/architecture/v24-application-integration-20260714-1526-claude.archimate`（含 Plateau/Gap/WP）
  - `docs/architecture/v24-application-integration-20260714-1526-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `POST /api/accounts/users/activate-session/`（taskAuth）
  - 🟢 [NEW] Vue `savedAccounts` 槽 + AccountSwitcher 下拉
  - 🟡 [MODIFIED] Navbar / Login 接入多账号
  - 🟡 [MODIFIED] taskAuth OpenAPI schema

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v21 | Current 基线（单会话 Token） |
| Plateau v24 | Target — 多账号槽 + activate-session |
| Gap | 无多账号切换；昵称不可下拉切换身份 |
| WorkPackage | 实现 activate-session + Navbar 下拉 + 槽存储 |

## 非目标（本期不做）

- 跨设备账号列表同步
- 服务端持久化「设备账号列表」
- 同一页面内无刷新热切换（必须整页刷新保证全局状态干净）
- 修改公司切换器语义

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-14 | 初稿；goal-mode 自动批准方案 B |
