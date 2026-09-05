# Value Stream: GitHub Connection GET user_id 签名修复

> Derived from design: `docs/superpowers/specs/2026-05-27-github-connection-get-user-id-design.md`

## Value Summary

登录用户打开 Git 网站授权设置页时，能成功读取 GitHub/GitLab OAuth 绑定状态，不再因 500 报错而显示「暂时无法获取绑定状态」。

## End-to-End Flow

[用户打开 git-site-oauth 页] → [前端 GET user-scoped connection API] → [Django 视图吸收 user_id kwargs] → [gitOauth 摘要] → [200 + connected/connections] → [页面展示绑定状态]

## Value Increments

### Increment 1: connection GET 500 修复（Thin Slice）

**Value to user:** 授权设置页可正常加载绑定状态  
**Scope:** `GithubAppConnectionView.get` 增加 `user_id` 参数 + pytest 回归  
**Depends on:** nothing
