# Review: taskGateway Federated Swagger (D1–D3)

**Verdict:** 通过（无 critical）

## 对照设计

| 项 | 状态 |
|----|------|
| 分服务门户 `/gateway/docs/` | ✅ codegen + nginx |
| 仅 dev（`docs.enabled` / env） | ✅ 默认关，CI 断言 |
| upstream `docs` 契约 R1–R4 | ✅ validate + pytest |
| django/gitOauth server URL | ✅ SWAGGER_SETTINGS / SERVERS |
| taskAuth D4 | ✅ `/api/schema/`、`/api/swagger/`、门户 taskAuth 链接 |

## 备注

- 启用文档：`conf/task-gateway/config.yaml` 设 `docs.enabled: true` 或 `TASK_GATEWAY_DOCS_ENABLED=1` 后 `routes-apply`。
- `docker-compose` 新增 `docs-portal` 服务，需先 `routes-apply` 生成门户 HTML。
