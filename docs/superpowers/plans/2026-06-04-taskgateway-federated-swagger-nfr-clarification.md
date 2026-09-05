# NFR 澄清: taskGateway 分服务 Swagger 门户

> 输入:
> - 设计: `docs/superpowers/specs/2026-06-04-taskgateway-federated-swagger-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-04-taskgateway-federated-swagger-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 文档页加载 < 3s（本地 dev） |
| 可伸缩性 | L0 | 不适用（仅 dev） |
| 可用性 | L1 | 文档不可用不影响业务 API |
| 安全性 | L3 | 生产不暴露 docs；内部 API 不出现在 schema |
| 数据一致性 | L0 | 不适用 |
| 可观测性 | L2 | docs 请求走现有 access log |
| 可维护性 | L2 | 新 upstream 仅改 YAML + codegen |

## 质量场景

### QS-01: 生产关断

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激 | CI 以 `docs.enabled=false` 运行 codegen |
| 响应 | `apisix.yaml` 无 `gateway-docs` 路由 id |
| 响应度量 | pytest 断言 route_ids 不含 `gateway-docs-portal` |

### QS-02: 开发门户可达

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激 | `TASK_GATEWAY_DOCS_ENABLED=1` + routes-apply |
| 响应 | 生成 `gateway-docs-portal` 与 `gateway-docs-django-ui` |
| 响应度量 | pytest 快照 route_ids |

## 领域模型影响

| NFR 决策 | 模型影响 |
|----------|---------|
| 安全 L3 生产关断 | `DocsDescriptor` 与 `GatewayRouteTable` 同版本发布；`enabled` 为聚合不变量 |
| 可维护 L2 | `Upstream` 聚合根必须含 `docs` 子对象或显式 `false` |

## 权衡与边界

- 不在 V1 做 schema 运行时 merge。
- 文档路由 `auth_mode: none` 仅限 dev 网络假设。
- taskAuth OpenAPI 延后至 D4。

## 跳过声明

- 可伸缩性、数据一致性：纯 dev 文档代理，无持久化状态。
