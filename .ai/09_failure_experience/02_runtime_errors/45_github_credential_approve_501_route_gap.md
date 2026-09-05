# [运行时] 任务详情「保存 GitHub 账号失败」→ taskCloudService 501

## 基本信息

- 案例编号：FE-20260718-GITHUB-CREDENTIAL-APPROVE-501
- 录入日期：2026-07-18
- 关联服务：taskCloudService（:8018）、Django `github_credential_approve`、前端 `TaskDetailLinkedProjectsPanel`
- 相关：同目录 `12_container_auto_run_steps_http_501_route_gap.md`（同一类「OpenAPI 已写、路由未注册 → 501」）

## 失败现象

- 页面：任务详情「关联项目」→ 为仓库选择 GitHub 账号后点保存
- 可见文案：`保存 GitHub 账号失败`（`p.mt-2.text-[11px].text-red-600`，带 `data-traceId`）
- 网关日志（trace `14d41324-5bb9-497e-8b37-9060dac64d01`）：
  - `POST …/cloud/compute/github-credential-approve/` → **upstream 501**
  - `route_id=task-cloud-service`，`content-length: 116`

## 根因

1. 前端保存调用：
   `POST /api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/github-credential-approve/`
2. APISIX 将 `/…/cloud/*` 打到 **taskCloudService**
3. `handleCloudTaskRoutes` 仅注册了 `compute/github-credential-status`（代理 Django），**未**注册 `compute/github-credential-approve`
4. 落入 catch-all：

```json
{"status":"error","message":"compute action not yet ported to taskCloudService: compute/github-credential-approve"}
```

5. 前端 `!response.ok` 时若无字符串 `detail`，展示兜底文案「保存 GitHub 账号失败」
6. Django 侧 `github_credential_approve` 与 OpenAPI 条目均已存在，属**迁服遗漏**，非业务校验失败

## 修复

在 `taskCloudService/src/compute_handlers.go`：

1. switch 增加 `compute/github-credential-approve` → `handleGithubCredentialApprove`
2. 实现与 status 对称：校验 POST + 上下文后 `proxyDjangoRequest` 到 Django 同路径
3. 单测：`TestGithubCredentialApproveProxiesToDjango`、`TestGithubCredentialApproveRejectsNonPOST`
4. 重建并重启 `taskCloudService`（`:8018`）

## 验收

```bash
# 不再 501；业务校验可达 Django（错误仓库 → 400 detail）
curl -sS -o /tmp/r.json -w '%{http_code}' -X POST \
  'http://127.0.0.1:8018/api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/github-credential-approve/' \
  -H 'Content-Type: application/json' \
  -H "X-Auth-Tenant-Id: {t}" -H "X-Workspace-Id: {w}" -H "X-Task-Id: {task}" \
  -H "X-User-Id: {uid}" -H "Authorization: Token {token}" \
  -d '{"repo_slug":"owner/repo","github_user_id":"<id>"}'
# 合法仓库 + 已绑定账号 → 200 {"approved":true,...}；status 接口 all_repo_bound=true
```

本案例任务 `task_13419614134018489866`：`task2money/ram-work` + `github_user_id=1321779` → **200**，`all_repo_bound=true`。

## 预防

- OpenAPI / Django 已声明的 `cloud/compute/*` 写操作，必须在 `handleCloudTaskRoutes`（或 `isDjangoProxiedComputeSub`）显式登记；禁止只写文档
- 新增对称 GET/POST 对时（status/approve）一次性注册两条
- 前端对 501 且 body 含 `not yet ported` 可展示更明确文案（可选加固，非本修复必须）
