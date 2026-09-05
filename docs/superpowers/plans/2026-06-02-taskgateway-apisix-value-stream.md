# Value Stream: taskGateway（APISIX）

> 设计：`docs/superpowers/specs/2026-06-02-taskgateway-apisix-design.md`

## Related Value Streams

| Stream | 关系 |
|--------|------|
| `user-auth` | 浏览器 login/profile 入口改为经网关 |
| `task-container-gateway` | container-git-commit 改经网关 upstream |
| `frontend-auth-guard-redirect` | profile 校验 URL 改为 `https://gateway/api/...` |

**类型：** 扩展 + 新域 `task-gateway`（边缘路由）

## 端到端价值流

```text
[用户打开 Vue :4000]
  → [apiFetch → https://gateway:8443/api/...]
  → [taskGateway：TLS + 路由 +（G3）forward-auth]
  → [upstream: task-auth | git-oauth | django | task-sse | t-c-gateway]
  → [用户完成登录 / OAuth / 业务操作]
```

**交付点：** 用户无需知道后端端口，单一 HTTPS API 入口可用。

## 价值增量

| Inc | 名称 | 用户可感知价值 | 依赖 |
|-----|------|----------------|------|
| **G1** | gateway-skeleton-tls | `curl -k https://gateway/api/health/` 通 django | task-auth、saas-backend |
| **G2** | gateway-route-split | OAuth start/callback 经网关到 git-oauth | G1、git-oauth |
| **G3** | gateway-forward-auth | 无 token 被拒；Django 信任 X-User-Id | G1、task-auth internal |
| **G4** | gateway-cors-limit-trace | 前端跨域无 CORS 错误 | G3 |
| **G5** | frontend-api-base-switch | Vite 去掉 /api proxy；vue apiBaseUrl=https gateway | G4 |
| **G6** | gateway-ci-guard | 路由 YAML 与 apisix 生成物 CI 校验 | G5 |

**薄切片（G1）：** 全路径默认 `/* → django`，仅验证 TLS + runAll 编排。

## YAML 登记

见根目录 `value-stream.yaml` 新增 `task-gateway` stream。
