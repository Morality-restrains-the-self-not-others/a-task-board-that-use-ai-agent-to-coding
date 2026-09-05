# [运行时] ztree 推送失败：Git 身份 lookup 用了 internal_gateway 哨兵

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-21
- 编号：105
- 维护者：Trae AI 团队

## 现象

- 任务详情「提交并创建 PR」后可见：
  `提交成功但推送失败：Git 身份不存在或不属于当前租户`
- 元素：`p.taskplugin-el-highlight`，`data-traceId=39e84a49-8629-4418-9ea4-a79827d7654e`
- 任务：`task_878583341551480832`；租户 `877397588196749312`
- Loki 同 trace：`task-cloud-service` `POST /api/internal/layer-git-push/prepare` **404，1ms** → Gateway `cloud_prepare_git_push` 404。未到达容器 `oauth-access-push`。
- MySQL：该用户确有 `task_git_identities`（`gi_877397592462356480`，`user_id`=auth 用户，`company_id`=租户）。用真实 auth user lookup `found=true`；用 `internal_gateway` 则 `found=false`。

## 根因

1. 公网 git-push 经 Cloud `copyProxyHeaders` 转到 Container Gateway。Cloud **不转发 Cookie**（避免 Token IP 绑定 401），但会加 `X-TaskContainerGateway-Internal-Secret`，并复制 `X-Auth-User-Id`。
2. Gateway `authorizeContainerRequest` 密钥匹配后**提前返回** `UserID: "internal_gateway"`，丢掉已转发的浏览器用户。
3. `handleContainerLayerGitPush` 把该哨兵写入 prepare 的 `user_id`。
4. Cloud `lookupUserCompanyGitIdentity(identity_id, "internal_gateway", tenant)` → `found=false` → 用户可见 404「Git 身份不存在或不属于当前租户」。
5. 多仓 `prefer_container_remote=true` 会跳过 identity 门禁，所以多仓看起来正常、单仓必挂。

与 [104](./104_ztree_push_git_identity_lookup_internal_only.md) 文案相近但根因不同：104 是 Cloud→Task lookup 缺 `X-Auth-User-Id: internal` 导致 403；本条是 Gateway 把**业务 lookup 的 user_id** 写成哨兵。

## 解决方案

1. Gateway `forwardedTrustedUserID`：内部密钥旁路优先用 `X-Auth-User-Id` / `X-User-Id`（排除 `internal` / `internal_gateway`）；仅只读路径无用户头时才回退哨兵。
2. Cloud prepare：`user_id` 为哨兵时返回明确 401「内部推送缺少真实用户身份」，禁止再伪装成身份 404。
3. identity miss 日志带 `identity_id` + `user_id` + `tenant_id`。

## 验证

```bash
cd taskContainerGateway && go test ./src/ -count=1 \
  -run 'TestAuthorizeContainerRequest_InternalBypass|TestHandleContainerLayerGitPush_InternalBypass|TestForwardedTrustedUserID'
cd taskCloudService && go test ./src/ -count=1 \
  -run 'TestLayerGitPushPrepare_SentinelUserIDRejected|TestCopyProxyHeaders_ForwardsAuthUserId'
```

精准编译重启 `task-container-gateway`、`task-cloud-service` 后，同任务再点「提交并创建 PR」：prepare 不应再因哨兵 user_id 返回该 404。

## 关联

- `.ai/09_failure_experience/02_runtime_errors/104_ztree_push_git_identity_lookup_internal_only.md`
- `.ai/09_failure_experience/02_runtime_errors/74_ztree_push_unauthorized_gitoauth_bridge_secret.md`
- `taskContainerGateway/src/auth_cloud_clients.go`
- `taskCloudService/src/git_push_internal.go`
- `taskCloudService/src/container_gateway_proxy.go`
