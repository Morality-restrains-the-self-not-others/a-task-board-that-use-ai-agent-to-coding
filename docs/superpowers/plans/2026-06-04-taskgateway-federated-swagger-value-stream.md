# Value Stream: taskGateway 分服务 Swagger 门户

> 设计：`docs/superpowers/specs/2026-06-04-taskgateway-federated-swagger-design.md`

## Value Summary

开发者在单一 HTTPS 网关入口 `/gateway/docs/` 发现并调试各 upstream 的 Swagger UI，新增服务时仅声明 `upstreams.*.docs` 即可自动出现在门户中。

## Related Value Streams

| Stream | 关系 |
|--------|------|
| **task-gateway** | **扩展** — 在 G1–G6 路由/鉴权之上增加 docs 反代与 codegen |
| **user-auth** | 交叉 — taskAuth OpenAPI 为 D4 增量，不阻塞 D1–D3 |

## End-to-End Flow

```text
[开发者打开 https://gateway:8443/gateway/docs/]
  → [门户 HTML 列出 django / gitOauth / …]
  → [点击 /gateway/docs/django/… → APISIX 反代 Swagger UI]
  → [Try it out → https://gateway:8443/api/… → forward-auth + 路由拆分]
  → [看到与线上一致的 API 响应]
```

**交付点：** 无需记各服务端口即可经网关调试 API 文档。

## Value Increments

### Increment 1: D1 — gateway-docs-config-codegen（薄切片）

**Value to user:** `docs.enabled=true` 时门户可打开，django/gitOauth 文档页经网关加载。  
**Scope:** `conf` + `routes.yaml` docs 契约 + `routes-to-apisix` 生成路由 + 门户 HTML + dev nginx。  
**Depends on:** 现有 taskGateway 骨架。

### Increment 2: D2 — gateway-docs-django-gitoauth

**Value to user:** Swagger Try it out 默认打到 `publicBase`。  
**Scope:** django `SWAGGER_SETTINGS`、gitOauth `SPECTACULAR_SETTINGS['SERVERS']`。  
**Depends on:** D1。

### Increment 3: D3 — gateway-docs-upstream-contract-ci

**Value to user:** 新 upstream 漏声明 `docs` 时 CI 失败。  
**Scope:** pytest R1–R4 + `docs.enabled=false` 时无 `/gateway/docs` 路由。  
**Depends on:** D1。

### Increment 4: D4 — taskauth-openapi-stub（Future / 可选）

**Value to user:** taskAuth 出现在门户且可浏览 schema。  
**Scope:** Go `openapi.yaml` + `/api/schema/`。  
**Depends on:** D1；V1 可保持 `docs: false`。

## YAML 登记

追加至根目录 `value-stream.yaml` → `task-gateway` stream（见实现 PR）。
