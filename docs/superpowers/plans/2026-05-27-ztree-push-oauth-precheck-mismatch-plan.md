# zTree Push OAuth Precheck Fix — Implementation Plan

**Goal:** 移除 GitHub 专用前端 push 预检，对齐克隆/推送多 provider OAuth 判定。

## Tasks

- [ ] 1. 前端：删除 `layerGitGithubAppOauthConnected` push gate + vitest
- [ ] 2. 领域：`LayerPushOauthReadiness` + `resolve_layer_push_oauth_readiness`
- [ ] 3. 后端：扩展 `get_layer_git_push_auth_context` provider 字段
- [ ] 4. 后端：409 detail 使用 `layer_push_oauth_guidance_message`
- [ ] 5. pytest：`test_layer_git_push_with_gitlab_oauth_only` + auth context gitlab
- [ ] 6. 运行相关 pytest + vitest
