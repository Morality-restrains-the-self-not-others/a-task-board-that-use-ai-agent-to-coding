# DDD 领域建模: SSO 401 Fix — Skip Declaration

> 输入:
> - 设计文档: `docs/design/sso-401-fix-design.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-29-sso-401-forward-auth-cookie-bridge-nfr-clarification.md`

## 跳过理由

**价值流增量不涉及新的业务概念。** 此修复为纯调用方变更：

- **Before:** `handleGatewayForwardAuth` 调用 `tokenFromRequest(r)` → `resolveTokenUserID(tokenKey)`
- **After:** `handleGatewayForwardAuth` 调用 `resolveTokenUserIDFromRequest(r)`

`resolveTokenUserIDFromRequest` 是已存在于 `oidc_handlers.go` 的领域服务，包含 4 层认证回退链：
1. `Authorization: Token xxx` header
2. `token` cookie
3. `Authorization: Bearer xxx` (OIDC JWT)
4. `userId` cookie (cross-port bridge)

本次修复仅为 **调用方迁移**——将 `handleGatewayForwardAuth` 从旧的 2 段式认证切换到统一的 4 层认证链。

## 已有领域模型（无需变更）

| 概念 | 位置 | 状态 |
|------|------|------|
| Token Resolution Chain (Domain Service) | `taskAuth/src/oidc_handlers.go: resolveTokenUserIDFromRequest` | 已存在，无需修改 |
| Gateway Forward-Auth Handler | `taskAuth/src/gateway_forward_auth.go: handleGatewayForwardAuth` | 调用方变更，领域语义不变 |
| userId Cookie Bridge (Value Object) | 内存 cookie 读取 | 已存在，无需修改 |
| User Active State Check | `taskAuth/src/oidc_handlers.go: userIsActive` | 已存在，无需修改 |

## 不需要新增的领域概念

- 无新实体
- 无新值对象
- 无新聚合
- 无新仓储接口
- 无新领域事件
- 无新领域服务

## 自检

- [x] 无新领域概念需要建模
- [x] 已有领域模型正确复用它处
- [x] 跳过理由文档化

**结论: 跳过 DDD 建模阶段，直接进入实施计划。**
