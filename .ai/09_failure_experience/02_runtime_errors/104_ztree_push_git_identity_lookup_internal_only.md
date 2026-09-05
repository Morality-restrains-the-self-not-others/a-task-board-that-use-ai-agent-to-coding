# [运行时] ztree 推送失败：git identity lookup 403 internal only

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-21
- 编号：104
- 维护者：Trae AI 团队

## 现象

- 任务详情「提交成功但推送失败」，可见文案：
  `saas identity lookup failed: git identity lookup status=403 body={"error":"internal only",...}`
- 典型任务：`task_878544011252494336`；trace：`539d4fdc-53bf-49bf-9ecc-2270f92285d1`
- 失败停在 Gateway `cloud_prepare_git_push` / Cloud `POST /api/internal/layer-git-push/prepare`（502，duration_ms=1），**未**到达容器 `oauth-access-push`。
- Loki 同 trace 无 `task-task-service` 行：Cloud→Task 出站未透传 `trace_id`。

## 根因

1. Cloud `realFetchUserCompanyGitIdentity` 调 Task `POST /api/internal/git-identities/lookup/`。
2. lookup 鉴权是 `isInternalCall`（`X-Auth-User-Id==internal`）**或** `X-Internal-Secret`。
3. 两端 `conf/.../config.yaml` `shared.internalSecret` 均为空 → Cloud 不发 secret。
4. 同仓其它 Cloud→Task 调用（auth-context / feature-params / PR metadata）会设 `X-Auth-User-Id: internal`；**lookup 漏设**。
5. Task 立即 403 `internal only`；Cloud 包装为 502 `saas identity lookup failed`。

本地复现：

```bash
# 403
curl -sS -X POST http://127.0.0.1:8017/api/internal/git-identities/lookup/ \
  -H 'Content-Type: application/json' \
  -d '{"identity_id":"x","user_id":"y","company_id":"z"}'
# 200 {"found":false}
curl -sS -X POST http://127.0.0.1:8017/api/internal/git-identities/lookup/ \
  -H 'Content-Type: application/json' -H 'X-Auth-User-Id: internal' \
  -d '{"identity_id":"x","user_id":"y","company_id":"z"}'
```

## 解决方案

1. Cloud lookup 请求固定设置 `X-Auth-User-Id: internal`；secret 非空时仍带 `X-Internal-Secret`。
2. HTTP Client `Transport.Proxy = nil`（禁止继承环境代理）。
3. 回归测：空 secret + 无 internal 头 → 403；带 `X-Auth-User-Id: internal` → 放行。

## 验证

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'TestRealFetchUserCompanyGitIdentity'
cd taskTaskService && go test ./src/ -count=1 -run 'TestInternalLookupGitIdentity'
```

精准编译重启 `task-cloud-service` 后，同任务再点推送：prepare 不应再因 identity lookup 403 返回 502。

## 关联

- `.ai/09_failure_experience/02_runtime_errors/74_ztree_push_unauthorized_gitoauth_bridge_secret.md`
- `taskCloudService/src/git_repo_identities_prepare.go`
- `taskTaskService/src/git_identity_ensure.go`、`tenant_membership.go`（`isInternalCall`）
