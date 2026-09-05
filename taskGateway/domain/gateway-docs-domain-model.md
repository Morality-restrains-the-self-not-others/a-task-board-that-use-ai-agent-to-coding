# Domain Model: Gateway Docs Portal

> NFR: `docs/superpowers/plans/2026-06-04-taskgateway-federated-swagger-nfr-clarification.md`

## Bounded Context: Edge Gateway (Docs)

### Aggregate: GatewayRouteTable

- **Root:** 路由表版本 + `docsEnabled` 开关
- **Invariant:** `docsEnabled=false` ⇒ 不得存在任何 `ProxiedDocsRoute`
- **Invariant:** 每个 `Upstream` 必须声明 `DocsDescriptor | DocsDisabled`

### Entity: Upstream

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 与 routes.yaml 键一致（django、gitOauth） |
| host | string | 逻辑主机 |
| port | int | 端口 |
| docs | DocsDescriptor \| DocsDisabled | 公开文档契约（`/gateway/docs/`） |
| docsInternal | DocsDescriptor \| 缺省 | 运维内部文档（仅 `/gateway/ops/docs/`，token 保护） |

### Value Object: DocsDescriptor

| 字段 | 规则 |
|------|------|
| uiPath | 以 `/` 开头 |
| schemaPath | 以 `/` 开头，可含 query |

### Value Object: DocsDisabled

显式 `false`，门户展示「未提供 OpenAPI」。

### Entity: ProxiedDocsRoute

| 字段 | 说明 |
|------|------|
| id | `gateway-docs-{upstream}-ui` |
| publicPrefix | `/gateway/docs/{upstream}/` |
| upstreamId | `up-{upstream}` |

### Entity: ProxiedOpsDocsRoute

| 字段 | 说明 |
|------|------|
| id | `gateway-ops-docs-{upstream}-ui` / `gateway-ops-docs-portal` |
| publicPrefix | `/gateway/ops/docs/{upstream}/` |
| auth | token + forward-auth（不得匿名浏览） |
| invariant | **不**出现在公开门户 HTML |

### Domain Event (optional V1)

`GatewayDocsRoutesPublished` — codegen 写入 `apisix.yaml`、`generated/docs-portal/index.html` 与 `generated/docs-portal/ops/index.html` 后触发（V1 仅 Git 审计）。

## Repository (codegen boundary)

`routes-to-apisix.py` 读取 `GatewayRouteTable` 源文件（YAML），输出 APISIX 配置与门户 HTML。
