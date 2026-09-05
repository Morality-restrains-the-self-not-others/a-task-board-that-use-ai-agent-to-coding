# Implementation Plan: relay 预检 token 换发失败分流

- [x] **T1** 测试：identity 存在 + mock token err → 502 `REPO_CLONE_TOKEN_REFRESH_FAILED`
- [x] **T2** 实现 `_build_repo_clone_credentials_with_diagnostics` + view 分流
- [x] **T3** relay precheck 透传 502/字段
- [x] **T4** 前端 `ServerConfig.logic.vue` 错误分流 + Playwright
- [x] **T5** `TaskDetailLinkedProjectsPanel` relay 模式显示克隆身份选择器
- [x] **T6** 回归 `test_container_runtime_tokens.py` + proxy tests
