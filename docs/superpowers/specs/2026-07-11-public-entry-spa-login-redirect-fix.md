# daydaymoney 登录后回跳登录页 — 设计与修复

- **日期**: 2026-07-11
- **迭代**: multi-entry-spa-login-redirect-loop
- **状态**: 实施方案 A（公网入口 SPA shell，不做服务端 session 302）

## 问题现象

打开 `https://www.daydaymoney.com/login/?next=/tenant/<id>/projects/`，邮箱密码登录成功后，又回到同一登录 URL。

## 根因

1. 登录 API（经 gateway → taskAuth）返回 **Token**；前端写入 `localStorage.authToken` + `userId` cookie，**不写 Django `sessionid`**。
2. 登录成功后 `window.location` 到 `/tenant/.../projects/`。
3. 该路径命中 Django `frontend_app` 的 `@with_workspaces`（内含 `@login_required`）视图，而非 Vite。
4. 无 session → Django **302** `Location: /login/?next=/tenant/.../projects/`。
5. 浏览器永远到不了 Vue Router；本地 Vite `:4000` 不受此中间件影响，故此前未暴露。

证据：`curl -I https://www.daydaymoney.com/tenant/.../projects/` → 302 到 `/login/?next=...`；同 Token 调 `/api/.../me/` → 200。

## 方案对比

| 方案 | 做法 | 优点 | 缺点 |
|------|------|------|------|
| **A（采用）** | 公网入口 Host 对非 API 的 GET 前端路径直接渲染 `spa.html`，跳过 `login_required` | 与 Vite/Token 模型一致；一次覆盖所有深链 | 须排除 `/api`/`admin`/`callback` 等 |
| B | 登录时额外写 Django session | 保留服务端跳转 | 与 taskAuth Token 双轨；跨域 Cookie 复杂度高 |

**不新增 Python HTTP 接口**（`python_api_approval: n/a`）。

## 架构影响

- **不新建 architecture target**：属 v16 多入口同源 SPA 落地缺陷修复，无新组件。
- 行为对齐：公网入口 HTML = SPA shell；认证由 Vue Router + `AuthSessionGuardService`（Token/`/me/`）负责。

## 实现要点

- 新增 `PublicEntrySpaShellMiddleware`（在 Session/Auth 之后、业务视图之前亦可；须在视图 `login_required` 之前短路）。
- Skip：`/api/`、`/static/`、`/media/`、`/admin/`、`/swagger`、`/redoc`、`/callback/`、`/api-auth/`、`/skillList`。
- 复用现有 `_render_spa` / `is_public_entry_request`。

## 验收

- Playwright：登录后最终 URL 为 `/tenant/.../projects/`，非 `/login/`。
- `curl -I` 公网入口 `/tenant/.../projects/` → **200** HTML（含 `/static/main-`）。
