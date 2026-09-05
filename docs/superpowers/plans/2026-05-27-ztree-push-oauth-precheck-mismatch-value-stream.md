# Value Stream: zTree 推送 OAuth 预检与克隆凭证对齐

> Derived from design: `docs/superpowers/specs/2026-05-27-ztree-push-oauth-precheck-mismatch-design.md`

## Value Summary

GitLab/GitHub 已授权且能克隆的用户，在 zTree 点击「推送」时不应被 GitHub 专用前端预检误拦；推送凭证判定与克隆路径一致，失败时给出 provider 感知提示。

## End-to-End Flow

[zTree 推送点击] → [前端身份/容器校验] → [POST container-layer-git-push] → [Django 按 provider 换票] → [容器 oauth-access-push] → [用户看到推送成功或明确 409/502]

## Value Increments

### Increment 1: 移除 GitHub 硬预检（Thin Slice）
**Value to user:** GitLab 已 OAuth + 已选克隆身份 → 点击推送能到达后端
**Scope:** 删除 `taskDetailLayerActions.js` GitHub-only gate；前端单元测试
**Depends on:** nothing

### Increment 2: 后端 auth context provider 感知
**Value to user:** 关联项目区/文档不再一律写「完成 GitHub OAuth」
**Scope:** 扩展 `get_layer_git_push_auth_context`；pytest
**Depends on:** Increment 1

### Increment 3: 推送 409 文案与 GitLab push 回归
**Value to user:** 真正缺授权或换票失败时，文案指向正确 Git 站点
**Scope:** `forward_container_layer_git_push` 409 detail；`test_layer_git_push_with_gitlab_oauth_only`
**Depends on:** Increment 1
