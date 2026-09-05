# [运行时] 经网关 GET `/api/tenant/*/daydaymoney/resolve` 返回 404

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 现象

- `GET https://example.com/api/tenant/{tenant_id}/daydaymoney/resolve?service_id=task2app` → **404**
- 响应体（Django）：
  ```json
  {
    "detail": "未找到请求的 API 路径，请确认 URL 正确且服务端已部署包含该接口的代码版本",
    "path": "/api/tenant/.../daydaymoney/resolve"
  }
  ```
- 直连 `http://127.0.0.1:8016/api/tenant/.../daydaymoney/resolve?...`（taskProjectService）→ **200**

## 根因

1. taskProjectService 已实现 `GET .../daydaymoney/resolve` 与 `POST .../daydaymoney/parse-yaml`（`daydaymoney_handlers.go`）。
2. 网关 `taskGateway/routes/routes.yaml` 的 `task-project-service` 仅白名单 `projects` / `workspaces` / `workspace-access` / `deliverable-systems` 等，**未登记** `daydaymoney`。
3. 请求落入 `django-default`（`uri: /*`）→ Django 无路由 → `ApiJson404Middleware` / `handler404` 返回带 `detail`+`path` 的 JSON 404。

## 解决方案

在 `taskGateway/routes/routes.yaml` 的 `task-project-service.uris` 增加：

```yaml
- /api/tenant/*/daydaymoney
- /api/tenant/*/daydaymoney/
- /api/tenant/*/daydaymoney/*
```

然后：

```bash
bash taskGateway/run.sh routes-apply   # 须带 TASK_GATEWAY_APISIX_IN_DOCKER=1（run.sh 默认已设）
```

**注意**：勿裸跑 `python3 scripts/routes-to-apisix.py`（缺 `IN_DOCKER=1` 时会把 upstream / forward-auth 写成 `127.0.0.1`，容器内鉴权 403）。

## 预防

- Go 新增公网路径时，必须同步 `routes.yaml`，并经 `18081` 验收（勿只测直连上游端口）。
- `db/api_route_ownership.yaml` 已登记 `prefix: /api/tenant/*/daydaymoney/` → `taskProjectService`；网关白名单须与 ownership 对齐。

## 验证

```bash
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy
# 期望 200 + status=success（matches 可为空，取决于项目 tags 是否含 svc:{service_id}）
curl -sS -o /tmp/daydaymoney.json -w '%{http_code}\n' \
  -H "Authorization: Token <session>" \
  'http://127.0.0.1:18081/api/tenant/<tenant>/daydaymoney/resolve?service_id=task2app'
```

## 关联

- 同类漏路由：`21_gateway_access_tokens_post_405.md`、`12_container_auto_run_steps_http_501_route_gap.md`
- Chrome 插件调用：`taskChromePlugin/lib/api.js` → `aidevResolve`
