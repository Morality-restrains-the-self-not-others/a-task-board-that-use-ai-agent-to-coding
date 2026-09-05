# Implementation Plan: relay bootstrap GitLab clone auth

- [ ] **Task 1** Django: `_build_repo_clone_credentials` 增加 `provider`、`git_http_username`
- [ ] **Task 2** Django tests: gitlab/github username 断言 + stale URL 回归
- [ ] **Task 3** bootstrap.mjs: `buildHttpAuthFromRepoCredential` 使用凭证字段
- [ ] **Task 4** Node tests + e2e: oauth2 断言
- [ ] **Task 5** 运行 pytest + node test 验证
