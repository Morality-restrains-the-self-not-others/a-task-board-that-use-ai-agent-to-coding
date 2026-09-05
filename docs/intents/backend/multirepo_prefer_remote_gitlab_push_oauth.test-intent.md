# 多仓 prefer_container_remote GitLab 推送无凭证 — 测试意图

## 变更记录
- 2026-07-12：与功能意图同步新增。

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | Go unit：`prefer_container_remote` 且无仓库/无 OAuth | `use_oauth_access_push=false`，200 |
| T2 | Go unit：`prefer_container_remote` + 双 GitLab 仓 + OAuth connected | `use_oauth_access_push=true`，`oauth_auth_by_repo`≥2，同 provider 只换票 1 次 |
| T3 | Django：prefer_remote + summary 未连接 | 仍 200，路径为 `git/push`，但会调用 summary |
| T4 | Django：prefer_remote + GitLab connected | 路径 `git/oauth-access-push`，body 含 `oauth_auth_by_repo` |
| T5 | Playwright E2E：任务页推送 | 响应非 Username/terminal prompts disabled；成功或可读业务错误 |

## 关联
- 功能意图：`multirepo_prefer_remote_gitlab_push_oauth.intent.md`
- 历史：`docs/superpowers/specs/2026-05-27-relay-localhost-gitlab-push-stuck-design.md`
