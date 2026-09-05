# 领域模型补充: relay 本地 GitLab 推送

**Bounded Context:** 任务协作 / 容器 Git（onlineServiceJS 为应用适配层）

## 值对象

- **CanonicalRepoKey** — `repo_url` 规范化（去 `.git`、小写）；SaaS `oauth_auth_by_repo` 键与容器 `origin` 对齐规则。
- **LayerPushOauthReadiness** — 已有（`cloud/domain/value_objects/layer_push_oauth_readiness.py`）。

## 领域服务

- **resolve_layer_push_oauth_readiness** — SaaS 侧（已有）。
- **resolveOAuthPushRepoContext** — 容器侧纯函数；按 VO 映射选择 provider/token，非新聚合。

## 仓储 / 事件

无 schema 变更；无新领域事件。

## 防腐层

- SaaS `forward_container_layer_git_push` → 容器 `oauth-access-push` DTO：`oauth_auth_by_repo[CanonicalRepoKey] = { provider, access_token }`。
