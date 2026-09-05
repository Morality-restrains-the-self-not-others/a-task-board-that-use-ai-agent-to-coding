# [运行时] 服务间调用经公网触发 APISIX deny-internal → 403 `{"detail":"forbidden"}`

## 现象

同类症状（响应体均为精确字符串 `{"detail":"forbidden"}`，或上游改写后的业务 403）：

1. **登录**：`POST https://www.daydaymoney.com/api/auth/` → 403（密码已校验通过后的 enrich 阶段）。
2. **GitLab 同步列表**：`GET .../projects/gitlab-remote-repos/` → 403；Grafana/Loki 中 `task-project-service` 有同 `trace_id` 的 `http_request status=403`，但 **saas-backend 无对应 internal 访问日志**（请求未到达 Django）。
3. **项目详情页**：`GET .../installed-images/`、`GET .../cloud-platform/.../cloud/regions/` → 403 `{"message":"无权访问该租户资源","status":"error"}`；Loki `{job="task-cloud-service"}` 可见 `forward_stage=django_internal path=/api/internal/taskproject/resolve-user-member/ django_status=403`。

## 根因

1. Go 服务（taskAuth / taskProjectService / taskCloudService 等）调用 Django `POST /api/internal/...`。
2. `djangoInternalApi` / `djangoInternalApiBase` **误配为公网**（如 `https://api.daydaymoney.com`），或进程仍持有旧公网配置而磁盘已改回 loopback。
3. 公网 APISIX 路由 `deny-internal`（`uri: /api/internal/*`）对外部请求 `fault-injection` 固定返回 403 `{"detail":"forbidden"}`。
4. 上游服务将该 403 原样（或经 `djangoPost` / `ensureTenantMember`）返回给浏览器。

典型误配示例：

- `conf/auth/task-auth/config.yaml` → `djangoInternalApiBase: ${scheme}://${subdomains.api}`
- `conf/taskProjectService/config.yaml` / `conf/taskCloudService/config.yaml` → `shared.djangoInternalApi: https://api.daydaymoney.com`
- 启动日志：`django=https://api.daydaymoney.com`（正确应为 `django=http://127.0.0.1:8001`）

## 解决方案

- 将 internal API 基址改为 Django loopback：`http://127.0.0.1:8001`。
- 同步修正 `conf/core/django/config.yaml` 的 `internalApiBase`，避免再同步出公网地址。
- **重启**对应服务（`task-auth` / `task-project-service` / `task-cloud-service`）使配置生效；仅改 yaml 不重启无效。
- taskProjectService / taskCloudService 启动时会校验 `djangoInternalApi` 必须为 loopback/私网，误配公网会直接 fail-fast。
- taskCloudService 的 `resolveUserMember` **优先**读 saas SQLite `accounts_company_member`（`lookupTenantMemberLocal`），避免 membership 热路径依赖可能被 deny 的公网 Django。

## 预防

- **禁止**把「浏览器可达的 API 公网域名」当作服务间 internal API 基址。
- 回归测例：`task2app/playwright/front_project/tests/Login.daydaymoney-auth-api-not-403.playwright.test.js`。
- 排查时可对比：直连 `http://127.0.0.1:8001/api/internal/...` 应到达 Django；经本地 APISIX `http://127.0.0.1:18081/api/internal/...` 或公网 `https://api.../api/internal/...` 必被 deny。
- Grafana：按 `trace_id` 查 `{job="task-project-service"}` 或 `{job="task-cloud-service"}`；若有 `django_status=403` 而无 saas-backend，优先怀疑 internal 基址指到了公网。

## 验证

```bash
# 登录路径
curl -sS -o /tmp/a.json -w '%{http_code}\n' -X POST 'https://www.daydaymoney.com/api/auth/' \
  -H 'Content-Type: application/json' \
  --data '{"username":"<email>","password":"<pbkdf2_hash>"}'
# 期望 200 且含 token

# deny-internal 对照
curl -sS -o /dev/null -w '%{http_code}\n' -X POST \
  'http://127.0.0.1:18081/api/internal/taskproject/gitlab-remote-repos/' \
  -H 'Content-Type: application/json' -d '{}'
# 期望 403 {"detail":"forbidden"}

# 项目详情 membership（直连 cloud，带网关用户头）
curl -sS -o /tmp/img.json -w '%{http_code}\n' \
  'http://127.0.0.1:8018/api/tenant/850256677331562496/installed-images/' \
  -H 'X-Auth-User-Id: 850256676127797248'
# 期望 200

bash task2app/playwright/front_project/tests/Login.daydaymoney-auth-api-not-403.playwright.test.sh
```
