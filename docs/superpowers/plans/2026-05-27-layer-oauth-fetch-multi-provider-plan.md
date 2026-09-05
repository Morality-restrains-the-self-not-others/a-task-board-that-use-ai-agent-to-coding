# 实施计划: 容器层 OAuth 拉取 AccessToken 多 Provider

> 设计: `docs/superpowers/specs/2026-05-27-layer-oauth-fetch-multi-provider-design.md`

## Task 1: Django repo_match_key 工具与 GitLab slug
- [ ] `projects/services/repo_match_key.py`
- [ ] 更新 `gitlab_repo_slug_from_url`
- [ ] `tests/test_repo_match_key.py`

## Task 2: Django 容器 API 多 provider 换 token
- [ ] `resolve_git_auth_by_repo_match_keys_for_container_task`
- [ ] `container_layer_github_oauth_views.py` 接受 `repo_match_keys`
- [ ] 集成测试 localhost:8012

## Task 3: onlineServiceJS fetch/refresh
- [ ] `repoMatchKey.mjs`
- [ ] `layerGitOauthFetchTokenFiles.mjs`
- [ ] `layerGitOauthRefreshPush.mjs`
- [ ] 单测 GitLab localhost

## Task 4: 回归
- [ ] 运行 pytest + node --test
