# [运行时] 任务详情「保存 GitHub 账号失败」→ taskCloudService HTTP 501

## 基本信息

- 案例编号：FE-20260718-GITHUB-CRED-APPROVE-501
- 录入日期：2026-07-18
- 最后更新：2026-07-18
- 关联服务：taskCloudService（:8018）、Django `github_credential_approve`、APISIX `task-cloud-service`

## 失败现象

- 页面：任务详情「关联项目」内仓库行「保存」GitHub 账号
- 可见文案：「保存 GitHub 账号失败」
- 错误节点：`p.mt-2.text-[11px].text-red-600`，`data-traceId="14d41324-5bb9-497e-8b37-9060dac64d01"`
- 网关日志：`POST …/cloud/compute/github-credential-approve/` → upstream `172.17.0.1:8018` **HTTP 501**，响应体约：
  `compute action not yet ported to taskCloudService: compute/github-credential-approve`

## 根因

1. 前端 `TaskDetailLinkedProjectsPanel.saveGithubRepoBinding` 调用  
   `POST /api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/github-credential-approve/`
2. APISIX 将该路径路由到 **taskCloudService**（与 `github-credential-status` 同前缀）
3. Go 侧仅注册了 `handleGithubCredentialStatus`（代理 Django）；**未注册** `github-credential-approve`
4. 落入 catch-all → 固定 **501** `not yet ported`；前端无 `detail` 时展示默认「保存 GitHub 账号失败」
5. Django `github_credential_approve` 业务逻辑本身仍可用，只是流量到不了

同类模式见 `12_container_auto_run_steps_http_501_route_gap.md`。

## 修复

在 `taskCloudService/src/compute_handlers.go`：

1. `handleCloudTaskRoutes` 增加 `compute/github-credential-approve` 分支
2. 新增 `handleGithubCredentialApprove`：校验 POST + 上下文后 `proxyDjangoRequest` 至 Django 同路径（与 status 一致）
3. 单测：`TestGithubCredentialApproveProxiesToDjango`、`TestGithubCredentialApproveRejectsNonPOST`
4. 重建并重启 `taskCloudService`（`./build.sh` + 重启 :8018）

## 验收

```bash
# 不再 501；无真实绑定时可为业务 400
curl -sS -o /tmp/r.json -w '%{http_code}' -X POST \
  "http://127.0.0.1:8018/api/tenant/.../task/.../cloud/compute/github-credential-approve/" \
  -H 'Content-Type: application/json' \
  -H 'X-Auth-Tenant-Id: …' -H 'X-Workspace-Id: …' -H 'X-Task-Id: …' \
  -d '{"repo_slug":"owner/repo","github_user_id":"…"}'
# 期望：非 501；成功时 200 且 body.approved=true

cd taskCloudService && go test ./src/ -run 'TestGithubCredentialApprove' -count=1
```

本机复现任务 `task_13419614134018489866`：approve 200 → status `all_repo_bound=true`。

## 预防

- OpenAPI / 设计已写「经 taskCloudService 代理至 Django」的路径，**必须同时**在 `handleCloudTaskRoutes` 登记；勿只写 status 漏掉对称 write 动作
- 迁服清单：对每个 Django `cloud/compute/*` 动作做「status 已代理 → approve/create/delete 是否也已代理」对照
- 前端在 `!response.ok` 时优先展示 `data.detail` / `data.message`，避免一律默认文案掩盖 501
- 变更后：`go test` + 对真实任务路径 curl，确认非 `not yet ported`
