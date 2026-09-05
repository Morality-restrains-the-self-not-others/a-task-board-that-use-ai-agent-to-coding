# [运行时] ztree「推送并创建PR」报 terminal prompts disabled

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-20
- 编号：67
- 维护者：Trae AI 团队

## 现象

- 任务详情点 ztree「推送并创建PR」错误弹层：
  `推送失败：fatal: could not read Username for 'https://github.com': terminal prompts disabled`
- 选择器：`p.taskplugin-el-highlight`（`showRequestError`）
- 典型任务：`task_13646028863068037877`；上游曾走 `…/layers/…/git/push`（非 oauth-access-push）。

## 根因

1. 多仓前端设 `prefer_container_remote=true`，且 **清空 `repo_url`**。
2. `prepare` 若 `collectTaskGitRepos` 未得到 githubRows、又无 `repo_url` hint → **不换 OAuth token**。
3. `prefer_container_remote` 且无 token 时旧逻辑回退 **裸 `git/push`**（`GIT_TERMINAL_PROMPT=0`）→ 正是该 Git 报错。

## 解决方案

1. prepare：无 OAuth 时默认 **409**；仅 `allow_bare_git_push=true`（本地 `/ui/dev-local-token`）才裸推。
2. 前端：多仓仍 `prefer_container_remote`，但传 **GitHub `repo_url` hint**、保留 `identity_id`，`allow_bare_git_push` 仅本地 dev。

## 验证

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'LayerGitPushPrepare'
cd taskFE/app && npx vitest run src/composables/taskDetail/taskDetailLayerActions.test.js
# 部署：重启 taskCloudService；前端 runall-lifecycle build
# 再点推送：应走 oauth-access-push；失败时应为「未能换取 Git OAuth 凭据…」而非 terminal prompts
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md`
- `taskCloudService/src/git_push_internal.go`
- `taskFE/app/src/composables/taskDetail/taskDetailLayerActions.js`
