# [运行时] ztree 推送 unauthorized：gitOauth bridge secret 拒收

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-22
- 编号：74
- 维护者：Trae AI 团队

## 现象

- 任务详情关联项目显示「OAuth 已授权」「克隆账号已保存」，但点 ztree 推送报 `unauthorized`（HTTP 502）。
- 典型任务：`task_13762772779307981876`；trace：`60345685-9a51-49b6-be9b-de9489fb307d`、`83b6bca2-d57e-41f2-97af-3e886ae3eb7f`。
- 失败停在 `cloud_prepare_git_push` / `POST …/layer-git-push/prepare`，**未**到达容器 `oauth-access-push`。

## 根因

1. UI「OAuth 已授权」走 Django → gitOauth `summary-for-user`（Django 带正确 `X-GitOauth-Bridge-Secret`）。
2. 公网推送走 Gateway → `taskCloudService` prepare → gitOauth `access-for-user`。
3. `conf/taskCloudService/config.yaml` 原 `bridgeSecret: ""`，且进程无 `GITOAUTH_BRIDGE_JWT_SECRET` 回退 → **不发送** bridge header。
4. `taskGitOauth` `RequireBridgeSecret=true` → 返回 `401 {"detail":"unauthorized"}`。
5. 用户误以为「未拉取 AccessToken」；实际是 **换票 API 被 bridge 鉴权拦住**。

日志证据（`logs/task-git-oauth.log`）：

```text
[taskGitOauth] internal API bridge secret rejected path=/api/internal/github/oauth/access-for-user/
status=401
```

## 解决方案

1. `conf/taskCloudService/config.yaml`：`bridgeSecret` 与 Django/SSO 开发密钥对齐。
2. `taskCloudService/src/config.go`：与 `taskProjectService` 相同回退链  
   `GITOAUTH_BRIDGE_JWT_SECRET` → `TASK2APP_SSO_JWT_SECRET` → 本地默认密钥。
3. prepare 失败日志带 detail；`unauthorized` 映射为「bridge secret / 授权失效」可操作文案。
4. Django `fetch_github_access_via_gitoauth_for_user` 改用 `default_github_provider_key()`（避免硬编码 `"github"`）。

## 验证

```bash
# 单测
cd taskCloudService && go test ./src/ -count=1 -run 'FetchGitOauthAccessForUser_SendsBridgeSecret|LayerGitPushPrepare_UnauthorizedBridgeMapsGuidance'

# 运行中服务：access-for-user 不应再出现 bridge secret rejected
rg 'bridge secret rejected' logs/task-git-oauth.log | tail

# 页面：同任务再点推送，应进入 oauth-access-push；prepare 200 且带 github_auth_by_repo
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/67_ztree_push_terminal_prompts_disabled.md`
- `.ai/09_failure_experience/02_runtime_errors/41_github_oauth_ok_but_unbound_provider_key_mismatch.md`
- `taskCloudService/src/git_push_internal.go`、`taskCloudService/src/config.go`
- `conf/taskCloudService/config.yaml`
