# 意图：任务容器 SSH 仓库 HTTPS 克隆 + selected_image Token 同步

- **日期**: 2026-07-10
- **关联页面**: `http://183.250.1.132:4000/tenant/.../task-detail/task_12590983282794675865/?relayToTrae=true`
- **状态**: 已实现

## 问题

1. bootstrap 对 `git@183.250.1.132:...` 直接 SSH 克隆 → `Host key verification failed`（容器无 known_hosts/私钥；契约应为 OAuth HTTPS）。
2. selected_image 模式下 go_relay 用已作废 bootstrap token 做 status-push → `TOKEN_ACCESS_INVALID` / unregister。

## 验收标准

- [x] CRED `repo_clone_credentials` 对 SSH URL 下发 `https_clone_url`（如 `http://183.250.1.132:8012/...`）
- [x] onlineServiceJS bootstrap 有 OAuth 时将 SSH URL 规范为 HTTPS 再克隆
- [x] go_relay selected_image：挂载 state 目录、等待 `container_refresh_token.json`、同步后再 register/status-push
- [x] 镜像已 `DOCKER_PUSH=1 ./buildDocker.sh` 推送 `x86_64-latest` / `arm64-latest`
- [x] go_relay / CRED 已用新二进制重启



## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| SSH 仓转 HTTPS 克隆并同步 selected_image token | RepoCloneCredentialsFetchSucceeded | repo_clone_credentials_fetch_succeeded | go_relayToTrae / onlineServiceJS | 克隆与 token 缓存 | — |
| OAuth token 拉取失败可观测 | OauthTokenFetchFailed | oauth_token_fetch_failed | cloud 凭证路径 | 审计 / 重试 | — |
## 变更要点

| 组件 | 变更 |
|------|------|
| taskCredentialService | `ResolveHttpsCloneURL` + `https_clone_url` 字段 |
| onlineServiceJS | `normalizeRepoUrlForHttpsClone` + bootstrap 使用 |
| go_relayToTrae | selected_image volume + token-sync A1 |
| machine_container.md | 契约补充 `https_clone_url` |
