# [运行时] ztree 推送成功但未创建 PR：Cloud prepare 缺 pr_base_branch

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-22
- 编号：75
- 维护者：Trae AI 团队

## 现象

- 任务详情 ztree「推送并创建PR」：仓库页已出现对应分支，但没有 PR。
- 典型任务：`task_13762772779307981876`（层 `20260721_183441_f95d41`）；
  merge 目标 `master`；工作分支含 `${taskId}`。
- 证据：`container-layer-git-push` **200**（trace `a26a46a0-2624-424a-b066-bf1d4afc3758`），
  prepare oauth ok、complete best-effort；响应无 `github_pull_request.html_url`。

## 根因

1. 公网路径已迁走 Gateway → Cloud `layer-git-push/prepare` → 容器 `oauth-access-push`；
   Django `forward_container_layer_git_push`（含 `wait_for_pr` / PR follow-up）不再承接。
2. Cloud Go prepare 只注入 `github_auth_by_repo` / `oauth_auth_by_repo`，**未**写入
   `pr_base_branch` / `pr_title` / `pr_body`（Django prepare 曾有此逻辑）。
3. 容器 `runLayerGithubOauthAccessPush` 仅在 `prBaseBranch` 非空且 ≠ head 时调用 GitHub pulls API；
   缺 base → **只推送、不建 PR**。
4. Gateway 把容器响应原样返回；前端读 `github_pull_request.html_url`，而容器只写
   `github_oauth_multirepo.repos[].pr`，UI 看不到 PR 链接。
5. 旁路：Django `resolve_layer_push_github_pr_metadata` 误用 `tid.isdigit()`，
   `task_*` ID 永远返回 None（影响 oauth-refresh-push / auto_run 换票附带 PR 元数据）。

## 解决方案

1. `taskCloudService`：`attachLayerGitPushPRMetadata` 从 taskTaskService `branch_strategy.merge_target_branch_name`
   解析 base/title/body，写入 prepare `push_body`。
2. `taskContainerGateway`：成功响应把 `repos[].pr` 汇总为 `github_pull_request`。
3. Django：去掉 `isdigit` 门禁；元数据拆到 `github_pull_request_branch_meta.py`。
4. onlineServiceJS：GitHub pulls **422** 时查找并复用已有 PR（对齐 GitLab MR）。

## 验证

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'LayerGitPushPrepare_AttachesPRBaseBranch|ResolveLayerPushGithubPRMetadata'
cd taskContainerGateway && go test ./src/ -count=1 -run 'GithubPullRequestSummary|EnrichesGithubPullRequest'
cd trae-agent/onlineServiceJS && node --test src/layerGitOauthPushPr.test.mjs
cd task2app && python3 -m pytest Saas_project/tests/test_github_pr_metadata_task_id_string.py -q

# 运行中 prepare（须含 pr_base_branch=master）
curl -sS -X POST 'http://127.0.0.1:8018/api/internal/layer-git-push/prepare' \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"…","workspace_id":"…","task_id":"task_…","layer_id":"…","user_id":"…","prefer_container_remote":true,"target_branch":"feature/…","repo_url":"https://github.com/…"}' \
  | jq '.push_body.pr_base_branch,.push_body.pr_title'
```

页面：同任务再点「推送并创建PR」→ 仓库出现 PR；ztree 可露出 PR 按钮。

## 关联

- `.ai/09_failure_experience/02_runtime_errors/67_ztree_push_terminal_prompts_disabled.md`
- `.ai/09_failure_experience/02_runtime_errors/74_ztree_push_unauthorized_gitoauth_bridge_secret.md`
- `.ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md`
- `taskCloudService/src/git_push_pr_metadata.go`
- `taskContainerGateway/src/handlers_git_push.go`
- `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
