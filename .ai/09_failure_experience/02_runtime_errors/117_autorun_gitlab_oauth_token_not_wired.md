# [运行时] auto_run 第 4 步未创建 PR：GitLab token 未写入 oauth_auth_by_repo

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：117
- 维护者：Trae AI 团队

## 现象

- 任务详情「自动运行步骤说明」第 4 步写明「推送远端并创建 PR」。
- ztree 首指令 **completed**，层上仍有「提交并创建PR」，无 PR 按钮。
- 典型：`task_878932440129761280`（GitLab `gitlab-tencent-sh-1.daydaymoney.com/example-user/somanyad`）。

## 时间线证据（Loki）

| 时间 (CST) | 事件 |
|------------|------|
| 17:29:14 | `AUTO_RUN_FIRST_INSTRUCTION_START` layer `20260822_092711_55994b` |
| 17:45:07 | `AUTO_RUN_DELIVERY_BEGIN` layer `20260822_092914_76165f` |
| 17:45:08 | `AUTO_RUN_DELIVERY_FAILED` http_status=400 still_ahead=true detail=`该仓库未找到可用的 OAuth access_token` |

trace_id：`dbb0c8bf71de832d09fb005a`

## 根因

1. credential `layer-github-oauth-access-tokens` 已返回 `git_auth_by_repo_match_key`（字符串图）。
2. `runLayerOauthRefreshPush` 把该图当作 `accessTokenByRepoSlug`（GitHub owner/repo），**不传** `oauthAuthByRepo`。
3. `resolveOAuthPushRepoContext` 对 GitLab **只读** `oauthAuthByRepo[canonicalUrl].accessToken`；`gitlab-tencent-sh-1.*` 也不匹配旧 `GL_SLUG_RE`。
4. 交付失败不写回 Agent 评论，页面只剩手动「提交并创建PR」。

## 解决方案

1. `buildOauthAccessPushAuthFromTokenPayload`：match-key 字符串 → `{provider, access_token}`，同时按 canonical URL 与 match key 索引。
2. GitLab 查找回退：slug / match key / canonical key；host 含 `gitlab` 时用 path slug。
3. 交付失败回填挂载 Agent 评论（`failed` + detail）。
4. `autoRunStep.md` 标明第 4 步依赖 OAuth 绑定。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/layerGitOauthPushAuthMaps.test.mjs \
  src/layerGitOauthPush.test.mjs \
  src/layerGitOauthRefreshPush.test.mjs \
  src/autoRunDeliveryHooks.test.mjs \
  src/autoRunPrBackfill.test.mjs
# 部署：DOCKER_PUSH=1 ./buildDocker.sh 后对该任务 stop-vm → start-vm
# Loki：AUTO_RUN_DELIVERY_COMPLETE 或评论出现「交付失败：…」
```

## 关联

- [66](./66_auto_run_delivery_done_locks_unpushed.md)
- `trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.mjs`
- `trae-agent/onlineServiceJS/src/layerGitOauthPushAuthMaps.mjs`
