# [运行时] 任务详情「租户不存在」：feature-params 仍走 Django companies/exists

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-20
- 最后修改：2026-07-20
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/{id}/workspace/{ws}/task-detail/{task}/`
- 层级指令区琥珀字：`租户不存在`，节点带 `data-traceId`
- 同 `trace_id`：`GET /api/tenant/{id}/feature-params/?view=summary` → **404**
- saas-backend：`POST /api/internal/companies/{id}/exists/` → **404**（Django「路径未找到」，非业务「公司无记录」）

## 根因

1. `accounts_company` SSOT 已迁至 **taskTenantService**（`/api/internal/tenant/companies/creator`）。
2. `taskCloudService.verifyCompanyExists` 仍调已删除的 Django `companies/{id}/exists/`。
3. 任意非 200 被映射为「租户不存在」，真实存在的租户也被误杀。

## 解决方案

1. `verifyCompanyExists` 改为 `GET taskTenantService /api/internal/tenant/companies/creator?company_id=`。
2. `conf/taskCloudService/config.yaml` 增加 `services.taskTenantService`（8020）。
3. 单测 mock 同步挂 `TaskTenantURL`。

## 验证

```bash
curl -sS -o /tmp/fp.json -w '%{http_code}\n' \
  'http://127.0.0.1:8018/api/tenant/850256677331562496/feature-params/?view=summary' \
  -H 'X-Auth-User-Id: 850256676127797248'
# 期望 200

# 日志应见 forward_stage=tenant_internal path=/api/internal/tenant/companies/creator tenant_status=200
# 不应再出现 django_internal .../companies/.../exists/
```

## 预防

- 公司/租户存在性校验禁止再直连 Django 已迁出表的 internal 路径。
- 迁表 PR 须同步扫描调用方（`rg companies/.*/exists` / `verifyCompanyExists`）。
