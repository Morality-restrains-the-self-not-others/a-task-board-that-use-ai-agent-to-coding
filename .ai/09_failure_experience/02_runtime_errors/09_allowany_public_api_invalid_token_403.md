# 公开 AllowAny 接口因残留无效 Token 返回 403

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-14
- 最后修改：2026-07-14
- 维护者：Trae AI 团队

## 现象

- 登录页显示「系统尚未发布隐私条款/服务协议」，登录按钮禁用。
- Network：`/api/privacy-policy/public/current/`、`/api/license-agreement/public/current/`、`/api/public/system-feature-policy/` → **403** `{"detail":"Invalid token."}`。
- 同一接口无 `Authorization` 头时为 **200**。

## 根因

1. `apiFetch` 会把 `localStorage.authToken` 附到所有请求的 `Authorization: Token …`。
2. DRF 对 `AllowAny` 视图仍会先跑默认认证；**无效 Token 在鉴权阶段直接 403**，不会进入视图逻辑。
3. 会话过期或 Token 被撤销后，公开接口被「脏 Token」打挂，前端误判为「未发布条款」。

## 修复与预防

1. **公开视图**：`authentication_classes = []`（隐私条款、服务协议、公开特性策略）。
2. **前端**：`apiFetch` 收到 403 `Invalid token` 时清除 `authToken` 并无 Authorization 重试一次。
3. **回归**：`tests/test_privacy_policy_flow.py::test_public_current_ignores_invalid_authorization_header` 等；`apiUtils.test.js` 重试用例。

## 关联

- 静态资源白屏（可并发出现）：`04_public_spa_static_js_404_after_vite_build.md`
