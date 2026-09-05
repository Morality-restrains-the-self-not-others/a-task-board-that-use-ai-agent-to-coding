# 意图：镜像市场管理 SSO 经 /api/ 网关认证

## 目标

已登录的系统超级管理员在主站系统管理页点击「镜像市场管理（SSO）」时，应完成 bridge 换票并进入 Ai Provider 管理端，**不得**被重定向到 `/auth/login/?next=...`。

## 规范路径

| 角色 | 方法与路径 |
|------|------------|
| 超管管理端 | `GET /api/accounts/sso/ai-provider/admin/` |
| 厂商门户 | `GET /api/accounts/sso/ai-provider/vendor/` |

旧路径 `/accounts/sso/` **已清理**（不再挂载、不做兼容重定向）。

## 认证约束

- 生产边缘：`location /api/` → APISIX forward-auth → Django `GatewayAuthMiddleware`（仅网关头）。
- **不使用** userId/token cookie 路径级兜底；未经网关的直连 Django 视为未登录。

## 验收标准

1. 经 gateway（带 Cookie / forward-auth）访问 admin SSO → 302 到 provider `#sso_bridge=...`，**不**进 `/auth/login/`。
2. 无认证访问 → 302 `/auth/login/?next=/api/accounts/sso/...`。
3. 前端侧栏/厂商门户 href 为 `/api/accounts/sso/...`。
4. 仓库内无活跃代码再引用 `/accounts/sso/`（历史文档除外）。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：SSO 网关认证路径修复，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 意图：镜像市场管理 SSO 经 /api/ 网关认证 | — | — | — | — | SSO 网关认证路径修复，无新增业务事件 |
## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-07-13 | 初版：cookie 回退 + nginx 特判 | 生产复现踢回登录页 |
| 2026-07-13 | 统一 `/api/accounts/sso/`；清理旧路径；移除 cookie 兜底 | 与边缘 `/api/` 分流对齐 |
