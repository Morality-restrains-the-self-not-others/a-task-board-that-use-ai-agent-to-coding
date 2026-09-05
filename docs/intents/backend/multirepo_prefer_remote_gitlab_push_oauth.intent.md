# 多仓 prefer_container_remote GitLab 推送无凭证



## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 多仓 GitLab 推送补齐 OAuth 凭证 | RepoCloneCredentialsFetchSucceeded | repo_clone_credentials_fetch_succeeded | 容器推送 / OAuth token 同步 | 凭证缓存与后续 clone/push | — |
## 变更记录
- 2026-07-12：首次记录。traceId `0e74e58d-719b-4a6b-9fa1-a46dbd9000a4` 复现：多仓任务推送带 `prefer_container_remote=true`，prepare 跳过 OAuth，容器裸 `git/push` 对 `https://gitlab.daydaymoney.com` 报 `terminal prompts disabled`。

## 根因（已验证，traceId `0e74e58d-719b-4a6b-9fa1-a46dbd9000a4`）

1. 多仓前端发送 `prefer_container_remote=true` + 空 `identity_id`；旧 prepare 直接裸 `git/push`。
2. 容器 HTTPS origin 无持久凭据 → `could not read Username … terminal prompts disabled`。
3. Go prepare 未解析 task 的 `projects[]`（只认 `project_ids`），换票时仓库列表为空。
4. GitLab provider key 未命中用户已绑定的 `gitlab:daydaymoney-gitlab`（YAML website 模板未展开时回落 `gitlab:default`）。

## 修复

- `prefer_container_remote=true` 时仍 best-effort OAuth；有 token 则 `oauth-access-push`。
- 解析 task `projects[].stored_repo_address`。
- GitLab 换票按候选 `provider_key` 列表尝试，直至命中已连接凭据。


## 范围
- `taskCloudService`：`POST /api/internal/layer-git-push/prepare`
- Django 对等：`prepare_layer_git_push_auth`（Gateway 旁路后的兼容路径）
- 不改变前端 `prefer_container_remote` 请求体约定（多仓仍可发 true）

## 非目标
- 不在此实现 GitLab MR 自动创建
- 不要求容器持久化 clone 时的 ephemeral ASKPASS

## 验收
1. 多仓 + GitLab OAuth 已连接 + `prefer_container_remote=true` → prepare 返回 `use_oauth_access_push=true` 且含 `oauth_auth_by_repo`
2. 同配置无 OAuth → 仍可回退裸 `git/push`（SSH/本地凭据场景）
3. Playwright：任务详情推送 HTTP 非 400「could not read Username」
