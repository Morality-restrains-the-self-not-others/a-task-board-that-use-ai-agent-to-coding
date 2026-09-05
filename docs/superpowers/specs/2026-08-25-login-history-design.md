# 设计：登录历史记录（用户入口 + 管理员入口）

- **日期**: 2026-08-25
- **页面**: `/profile/login-history/`（账号中心侧边栏）；`/system-admin/users/:userId/login-history/`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 是（v111）— 无新服务；taskAuth 新增时间累积表 + 只读查询 API；taskFE 账号中心/超管用户表入口

## Context

当前 `auth_user.last_login` 只覆盖**最后一次**成功认证时间。用户无法核对自己账号从哪些 IP / 入口登录过；平台管理员也无法在用户入口与管理员入口之间区分登录痕迹。

既有能力：

- 客户登录：`POST /api/accounts/users/login/`、`POST /api/auth/` → `handleLogin` → `finalizeLogin`
- 管理员登录：`POST /api/auth/admin-login/` → `handleAdminLogin` → `finalizeLogin`
- 其它成功认证：微信扫码、访问令牌登录、注册后自动签发会话、`activate-session`
- `resolveClientIP` 已统一解析 XFF / X-Real-IP / RemoteAddr
- `USER_LOGGED_IN` 事件已存在，但不含 IP / 入口，且无持久化历史表
- 模拟登录明确**不得**把 `last_login` 打到目标用户（`auth_last_login.go`）

## Decision

1. **新表** `auth_login_history`（taskAuth 所有，前缀 `auth_`）：每次**成功**认证插入一行。失败登录不写入（避免把撞库噪声暴露给用户；失败审计另开 OPT）。
2. **记录字段**：`id`（Snowflake）、`user_id`、`logged_in_at`、`client_ip`、`user_agent`（截断 512）、`entry`、`method_type`。不存 identifier / token / 密码。
3. **入口枚举** `entry`：
   - `customer` — 用户登录入口（`handleLogin`）
   - `admin` — 管理员登录入口（`handleAdminLogin`）
   - `wechat` / `access_token` / `register` / `session_activate` — 其它成功认证
4. **写入点**：与 `touchLastLogin` 同路径，抽 `recordSuccessfulLogin`（fail-open：插入失败只打 `warn` 日志，不阻断登录）。模拟登录 start **不**调用。
5. **USER_LOGGED_IN** 向后兼容增补 `client_ip`、`entry`、`user_agent`（无新事件名）。
6. **用户查询**：`GET /api/auth/login-history/?limit=&offset=` — 仅当前会话用户自己的记录，按 `logged_in_at DESC`。网关 `auth_mode: token`，优先级高于 public `/api/auth/*`。
7. **管理员查询**：`GET /api/system-admin/users/{id}/login-history/?limit=&offset=` — `requireSuperuser`（与用户管理同闸）。
8. **前端**：
   - `UserCenterSidebar` 增加真实 `<a>`/`router-link`「登录历史」→ `/profile/login-history/`（及 `/user/:id/...`、`/tenant/:tenant/...` 变体）
   - 新页列表：时间、入口中文标签、登录方式、IP、User-Agent；空态；错误节点 `data-traceId`
   - 超管用户行增加真实链接「登录历史」→ `/system-admin/users/:id/login-history/`（`Anti-Replay-OK: real href`）
9. **PIPL**：个人数据导出增加 `login_history` section。
10. **冷热**：时间累积表；年增量按日活可达百万+ 行 → **首日按月 RANGE 分区**；PK `(id, logged_in_at)`；二级索引 `(user_id, logged_in_at)`。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 只更新 `last_login` / 覆盖式单字段 | 无法满足「历史」与多入口对照 |
| 消费 `USER_LOGGED_IN` 异步落库 | 事件不含可靠 IP；丢失/延迟不可接受；登录路径已有 request |
| 失败登录一并展示给用户 | 撞库噪声 + 信息泄露；本期只记成功 |
| Django 新接口 | 元规则 20：落 Go taskAuth |
| 无分区先上线 | 冷热分离元规则：时间累积表首日规划 |

## 契约

### GET `/api/auth/login-history/?limit=20&offset=0`

```
200 {
  results: [{ id, logged_in_at, client_ip, user_agent, entry, method_type, entry_label, method_label }],
  total, limit, offset
}
401 未鉴权
```

`limit` 默认 20，最大 100。`entry_label`/`method_label` 由服务端给出中文，前端不硬编码映射表凑测试。

### GET `/api/system-admin/users/{id}/login-history/?limit=20&offset=0`

同上；401 未鉴权 / 403 非超管 / 404 用户不存在。空历史：`results=[] total=0`（用户存在时不 404）。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|----------|--------|--------|--------|----------|
| 成功登录（各入口） | `USER_LOGGED_IN`（存量，增补字段） | `recordSuccessfulLogin` → `publishUserLoggedIn` | 既有消费者；本迭代无新消费者 | 不新增事件名，避免双投 |
| 用户/超管查看登录历史 | — | — | — | 纯 GET |

## 🐍 Python 新增接口清单与 Go 替代评估

**not_applicable** — 全部新接口落 Go taskAuth。无 Python endpoint。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 成功。无 `codegraph_explore` MCP；CLI `codegraph query handleLogin` / `callers publishUserLoggedIn`：

- `finalizeLogin`（客户+管理员密码登录）发布 `USER_LOGGED_IN` + `touchLastLogin`
- `handleWeChatCallback` 同样调用
- `handleLoginWithAccessToken` / 注册 / `handleActivateSession` 只 `touchLastLogin`，漏发事件 → 本迭代统一走 `recordSuccessfulLogin`（事件对后两者为新增，属登录成功系统事实）
- `UserCenterSidebar` 无登录历史项；`UserInbox` 为侧边栏列表页样板

## Value Stream Impact

挂入 `user-auth`。新增步骤 `login-history-record`、`login-history-self-view`、`login-history-admin-view`。

## 🏛️ 架构变更影响

- **迭代版本**: v111 🎯 target
- **迭代名称**: login-history
- **作者**: cursor
- **设计日期**: 2026-08-25 22:49
- **新增文件**（每个视图四类伴生格式）：
  - `docs/architecture/v111-application-integration-20260825-2249-cursor.puml`
  - `docs/architecture/v111-enterprise-landscape-20260825-2249-cursor.puml`
  - 伴生 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细**：
  - 🟢 [NEW] DataObject `auth_login_history`（按月分区）
  - 🟢 [NEW] GET `/api/auth/login-history/`、GET `/api/system-admin/users/{id}/login-history/`
  - 🟡 [MODIFIED] taskAuth 成功认证路径写入历史；`USER_LOGGED_IN` 增补字段
  - 🟡 [MODIFIED] taskFE 账号中心侧边栏 + 超管用户行链接
  - 🟡 [MODIFIED] taskGateway 将 `/api/auth/login-history/` 升为 token 路由
  - ⚠️ 无新 MQ topic
