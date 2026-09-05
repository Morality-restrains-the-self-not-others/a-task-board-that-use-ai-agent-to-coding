# NFR Clarification: taskGateway（APISIX）

> 价值流：`docs/superpowers/plans/2026-06-02-taskgateway-apisix-value-stream.md`  
> 增量范围：**G1–G3**（G4+ 在后续增量复用本表）

## 支撑等级（L1–L5，本项采用 L2 默认，安全 L3）

| 类别 | 等级 | 质量场景（刺激 → 响应） | 领域/实现影响 |
|------|------|-------------------------|---------------|
| **安全** | **L3** | 外网请求无 Token 访问 `/api/accounts/users/profile/` → 网关 **401**，不到 django | forward-auth fail-closed；`/api/internal/*` → 403 |
| **安全** | **L3** | 直连 django:8001 伪造 `X-User-Id` → **仍须** Token resolve | `X-TaskGateway-Internal-Secret` 校验 |
| **性能** | L2 | 内网经网关 profile P95 ≤ **350ms**（含一次 forward-auth） | G3 后 Django 免二次 resolve |
| **可用性** | L2 | task-auth resolve 不可用 → 网关 **503** + Retry-After | 与 identity-service-unavailable 一致 |
| **可维护性** | L2 | 路由变更仅改 `routes.yaml` + `routes-apply` | `GatewayRouteTable` 版本化 |
| **可观测性** | L2 | 每条请求 access log 含 `trace_id`、`route_id`、`upstream` | G4 完整启用 |
| **兼容性** | L2 | pytest 直连 django（无网关头）行为不变 | `TASK_GATEWAY_TRUST_HEADERS=False` in settings_test |

## 明确不做（G1–G3）

- 网关集群 HA、多 AZ
- JWT 终止（仍 Token + forward-auth）
- 替换服务端 django→taskAuth internal HTTP

## 跨 NFR 权衡

- **安全 L3 vs 性能：** forward-auth 增加一跳；G3 Django 信任头抵消 django 侧二次 resolve。
