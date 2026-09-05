# [运行时] 经网关 POST `/api/accounts/users/access-tokens/` 返回 405

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-16
- 最后修改：2026-07-16
- 维护者：Trae AI 团队

## 现象

- `POST http://<gateway>:18081/api/accounts/users/access-tokens/` → **405** `{"detail":"Method \"POST\" not allowed."}`
- 直连 `http://127.0.0.1:8003/api/accounts/users/access-tokens/`（taskAuth）→ **201** 正常

## 根因

1. taskAuth 已实现访问令牌 CRUD（`POST/GET /access-tokens/`、`DELETE /access-tokens/{id}/`）。
2. 网关 `routes.yaml` 仅有 `taskauth-get-user`（**仅 GET** `/api/accounts/users/*`）与公开登录类路由；**未登记** access-tokens。
3. POST 落入 `django-default`（`uri: /*`）→ Django DRF 无对应允许 POST 的动作 → 405。
4. 前端个人资料走 `/api/accounts/users/profile/access-tokens/`（Django 代理），故 UI 正常；插件/脚本直连 taskAuth 公网路径则失败。

## 解决方案

在 `taskGateway/routes/routes.yaml` 增加：

```yaml
- id: taskauth-access-tokens
  priority: 851
  uris:
    - /api/accounts/users/access-tokens/
    - /api/accounts/users/access-tokens/*
  upstream: taskAuth
  auth_mode: token
```

然后 `bash taskGateway/run.sh routes-apply`（生成 `apisix.yaml` + hot reload）。

同步：`db/api_route_ownership.yaml`、`docs/architecture/api-route-to-owner.md`。

## 预防

- taskAuth 新增公网方法/路径时，必须同时在 `routes.yaml` 登记，并确认不会被「仅 GET」通配或 `django-default` 吞掉。
- 验证：经 18081 测 POST/GET/DELETE，勿只测直连 :8003。

## 验证

```bash
# 期望 201 + at_…
curl -sS -o /tmp/at.json -w '%{http_code}\n' -X POST \
  'http://127.0.0.1:18081/api/accounts/users/access-tokens/' \
  -H "Authorization: Token <session>" -H 'Content-Type: application/json' \
  -d '{"name":"verify"}'
```

## 关联

- Chrome 插件令牌登录：`20_chrome_plugin_popup_login_storage_hang.md`
- 类似漏路由：`12_container_auto_run_steps_http_501_route_gap.md`
