# 设计：多账号切换后私有资源隔离

**日期**: 2026-07-15  
**类型**: 认证安全 / Bugfix  
**状态**: 已批准（goal-mode 自动采用）  
**迭代**: `multi-account-isolation-fix`  
**作者**: claude  
**基于**: `2026-07-14-navbar-multi-account-switcher-design.md`

## 问题

用户 A 切换为用户 B 后，B 仍可能看到 A 的私有资源（项目、任务、凭证、/me 资料等）。

## 根因

1. **Django DRF 鉴权顺序**：`SessionAuthentication` 在 `CustomTokenAuthentication` 之前；浏览器仍携带 A 的 `sessionid`（`credentials: include`）时，请求身份仍是 A，已切换的 Token B 被忽略。
2. **切换不清理 `sessionid`**：`activate-session` / 前端切换只更新 `authToken` + `userId`，不删 `sessionid`；enrich-login 在内部请求上建会话，不向浏览器回写新 cookie。
3. **租户/工作空间 URL 残留**：`resolveSwitchHref` 仅替换 `/user/{id}/`，在 `/tenant/...` 切换时仍停在 A 的租户上下文（意图 AC7 已写、代码未落地）。

## 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 切换账号后浏览器不再携带旧 `sessionid` | Vitest：切换路径调用清 cookie |
| S2 | 从 `/tenant/...` 切换落到 `/user/{B}/profile/`，并去掉 `workspace_id` | Vitest T10 |
| S3 | 请求同时带 Session(A) + Token(B) 时，API 身份为 B | Django 测试 / 鉴权顺序断言 |
| S4 | `me()` 的 path `user_id` 与 `request.user.id` 不一致时 403 | Django 测试 |
| S5 | 意图/测试意图/价值流图已同步隔离测试点 | 文档对照 |

## 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 仅前端清 sessionid + 离开租户 URL | 小、对齐已有意图 | 不修根因：其它路径仍可能 Session 压 Token | 必要但不充分 |
| B. 仅后端 Token 优先 | 根因级 | 旧 session 仍可能在无 Token 请求上误用 | 必要但不充分 |
| C. A + B + me() 路径校验 | 纵深防御 | 改动面含多处 authentication_classes | **采用** |

## 设计决策

### D1 — 前端切换前清理会话 Cookie

在 `activateSavedAccountSession` 成功写回凭据后（或 `handleSwitchAccount` 跳转前）调用 `clearCookie('sessionid')`（host-only + shared domain，复用 `cookieUtils.clearCookie`）。登出后自动切槽路径同样清理。

### D2 — `resolveSwitchHref` 落实 AC7

- path 含 `/tenant/` → 跳转 `/user/{newUserId}/profile/`
- 去掉 query `workspace_id`
- 仍替换 `/user/{old}/` → `/user/{new}/`

### D3 — Django Token 优先于 Session

- `DEFAULT_AUTHENTICATION_CLASSES`：`CustomTokenAuthentication` 在前
- 所有显式 `[Session..., Token...]` 覆写改为 `[Token..., Session...]`，与网关 forward-auth「Token 优先」一致
- 无 Authorization 时仍可用 Session（单账号登录体验不变）

### D4 — `me()` 路径身份校验

`GET /api/user/{user_id}/accounts/users/me/`：若 `str(request.user.id) != str(user_id)` → 403。

### 非本期

- activate-session 向浏览器 Set-Cookie 新 sessionid（可后续增强）
- 跨设备账号槽同步

## 架构影响

需更新 Application Integration：认证顺序原则「Token > Session」+ 切换清 sessionid。见 v28 架构产物。

## 业务意图 → 事件

| 业务意图 | 事件 | 例外 |
|---------|------|------|
| 切换账号隔离私有资源 | — | 纯认证/会话边界修复，无新业务状态变更 |

## python_api_approval

n/a — 不新增 Python 公网接口；仅调整既有鉴权顺序与 `me()` 校验。
