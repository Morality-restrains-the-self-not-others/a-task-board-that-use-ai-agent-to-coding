# [运行时] work-panel 打开后 work-panel-filters 接口 401

## 失败现象

登录后打开 `https://www.daydaymoney.com/tenant/{tenant}/work-panel`，浏览器 Network 中：

```text
GET /api/tenant/.../workspaces/.../work-panel-filters/  →  401
{"error":"unauthorized","message":"missing user"}
```

同页其它接口（workspaces、todos、machine-summary 等）为 200。

## 环境与上下文

- 新增接口：`taskProjectService` `handleWorkPanelFilters`（过滤栏持久化）
- 鉴权路径：浏览器 Cookie → APISIX `forward-auth` → taskAuth → 注入 upstream header → Go 服务
- APISIX `upstream_headers` 注入的是 **`X-User-Id`**（见 `taskGateway/scripts/routes-to-apisix.py`）
- 服务内约定头：`gatewayauth.HeaderAuthUserID` = **`X-Auth-User-Id`**
- APISIX `proxy-rewrite` 另注入 **`X-TaskGateway-Internal-Secret`**

## 排查过程

1. 复现确认仅 `work-panel-filters` 401，其余同网关 token 路由正常
2. 对照同服务其它 handler（`project_handlers` / `branch_handlers` 等）均有：
   `getAuthUser(r)` 为空时回退 `r.Header.Get("X-User-Id")`
3. `handleWorkPanelFilters` 只读 `getAuthUser`（`X-Auth-User-Id`），无 `X-User-Id` 回退
4. 直连 `:8016` 不带用户头 → 同样 `missing user`

## 根因

新 handler 只认 `X-Auth-User-Id`，未兼容网关实际注入的 `X-User-Id`，导致已登录用户被判「missing user」。各 handler 各自回退 `X-User-Id` 也缺少网关 secret 校验，可被伪造头冒充。

## 解决方案

1. **`gatewayUserMiddleware`**：调用 `gatewayauth.ApplyGatewayUser`（校验 `X-Gateway-Auth-Verified` + `X-TaskGateway-Internal-Secret`）后写入 `X-Auth-User-Id`
2. **`getAuthUser`**：仅读 `X-Auth-User-Id`（不再裸读 `X-User-Id`）
3. 删除 `project_handlers` / `utility_handlers` / `branch_handlers` / `workspace_access_handlers` 内重复回退
4. 单测：`TestGatewayUserMiddlewarePromotesXUserId`、`TestWorkPanelFilterAcceptsGatewayXUserId`、`TestWorkPanelFilterRejectsUnverifiedXUserId`

## 预防措施

- Go 服务读用户身份只走 `getAuthUser`；网关身份只经 `ApplyGatewayUser` 提升
- 禁止在 handler 里裸读 `X-User-Id` 当作已认证用户
- 新增需「当前用户」的 handler 时，补「已校验网关头」与「伪造 X-User-Id」两条回归测例
