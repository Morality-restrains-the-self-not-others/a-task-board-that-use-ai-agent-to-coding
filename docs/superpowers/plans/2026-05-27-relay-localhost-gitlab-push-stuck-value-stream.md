# Value Stream: relay 本地 GitLab 推送卡住修复

> Derived from design: `docs/superpowers/specs/2026-05-27-relay-localhost-gitlab-push-stuck-design.md`

## Value Summary

relay + somanyad（`localhost:8012`）任务在 zTree **推送** 时，容器能消费 SaaS 下发的 `oauth_auth_by_repo` 并完成 HTTPS push；失败在 90s 内返回可读错误，不再无限 busy。

## End-to-End Flow

[zTree 推送] → [SaaS container-layer-git-push + GitLab 换票] → [容器 oauth-access-push] → [git push + ASKPASS] → [200/400/502 → 用户 alert]

## Value Increments

### Increment 1: 容器识别 localhost GitLab（Thin Slice）
**Value to user:** 推送请求到达容器后实际执行 git push，而非 skip  
**Scope:** `resolveOAuthPushRepoContext` + `layerGitOauthPush.test.mjs`  
**Depends on:** nothing

### Increment 2: git push 超时治理
**Value to user:** 远端不可达时 ≤90s 失败，按钮恢复  
**Scope:** `GIT_PUSH_TIMEOUT_MS` in `gitExecAsync`  
**Depends on:** Increment 1

### Increment 3: relay Playwright 回归
**Value to user:** 自动化覆盖「直接启动 → 指令 → 提交 → 推送」  
**Scope:** `TaskDetail.relay-to-trae-somanyad-hello-world-push.playwright.test.js`  
**Depends on:** Increment 1–2

## 价值流配置影响

- 延续 `oauth-multi-repo-enhancement` / `task-detail-oauth-binding-guidance`
- 新增/关联测试：`trae-agent/onlineServiceJS/src/layerGitOauthPush.test.mjs`、`task2app/playwright/.../TaskDetail.relay-to-trae-somanyad-hello-world-push.playwright.test.js`
