# 意图：reclone HTTPS 规范化 + 旧 UI token 自愈

- **日期**: 2026-07-10
- **状态**: 已实现

## 验收标准

- [x] `POST /api/repos/reclone` 对 SSH URL + OAuth 凭证走 `https_clone_url` HTTPS 克隆（与 bootstrap 共用 `prepareOauthHttpsGitClone`）
- [x] `GET /ui/{stale_bootstrap}` 在换票后 302 到 `/ui/{current}`（仅记住的旧 token）
- [x] `GET /api/session/ui-redirect` + 控制台 SSE onerror 自愈跳转
- [x] Playwright API + CDP 9222 核验通过
- [x] 镜像已重新 `DOCKER_PUSH=1 ./buildDocker.sh`


## 业务意图 → 事件对照

> 精修（2026-07-15）：事件名对齐仓库 MQ / domain events；同步写路径或非 MQ 副作用在例外理由标注「证据豁免」。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| reclone HTTPS 规范化成功 | RepoCloneCredentialsFetchSucceeded | repo_clone_credentials_fetch_succeeded | 容器/relay 启动路径 | 克隆消费者 | — |
| UI 陈旧 token 刷新 | ContainerUiContextRefreshed | container_ui_context_refreshed | onlineServiceJS / Cloud | 前端上下文刷新 | — |
