# 设计：镜像市场 SSO 统一到 `/api/accounts/sso/`

**日期**: 2026-07-13  
**类型**: 路径迁移 / Bug 根治  
**状态**: 已实现（本会话）

## 目标

已登录超管点击「镜像市场管理（SSO）」经边缘 `/api/` → APISIX forward-auth 完成换票，不再踢回登录页。

## 方案（已选定）

1. Django 挂载改为 `api/accounts/sso/`；**删除**旧 `accounts/sso/`（无兼容重定向）。
2. 前端 / Ai Provider 入口 / Playwright / ownership / 文档全部改用新路径。
3. APISIX `django-accounts-sso`：`/api/accounts/sso/*`，`auth_mode: token`（Cookie → forward-auth）。
4. **不使用** Django `GatewayAuthMiddleware` cookie 路径兜底；仅 Session / 网关头。
5. nginx 示例删除 `/accounts/sso/` 特例（由既有 `location /api/` 覆盖）。

## 验收证据

- 单元：`test_gateway_auth_middleware_sso_api_path.py`（网关头认证；纯 cookie 不认证）
- 冒烟：`GET :18081/api/accounts/sso/ai-provider/admin/` + `userId` cookie → 302 `provider...#sso_bridge=`
- 无 cookie → APISIX 401；直连 Django 无网关头 → 不签发 bridge

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-13 | 落地统一 `/api/`；清理旧路径；移除 cookie 兜底 |
