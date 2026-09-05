# 实施计划: Create Project 页面 GitLab OAuth 仓库校验闭环

> 输入:
> - 设计文档: `docs/specs/create-project-gitlab-oauth-validation-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-26-create-project-oauth-validation-loop-ddd.md`

## Increment 1: 后端 token_status 区分 + 文案优化

### Task 1.1 — 创建 TokenStatus 值对象
- [ ] 创建 `projects/domain/repo_access/value_objects/token_status.py`
- [ ] 定义 4 个枚举值: `NOT_BOUND`, `TOKEN_ERROR`, `TOKEN_AVAILABLE`, `NOT_APPLICABLE`
- [ ] 定义 `DISPLAY_MESSAGES` 映射
- [ ] 验证: 运行 `python -c "from projects.domain.repo_access.value_objects.token_status import TokenStatus; print(TokenStatus.values())"`
- **文件**: `token_status.py` (已创建于 DDD 步骤)

### Task 1.2 — 扩展 ProjectRepoAccessResult 值对象
- [ ] 新增 `token_status`, `oauth_provider`, `oauth_service_provider` 字段 (均带默认值，向后兼容)
- [ ] 更新 `needs_auth()`, `not_accessible()`, `accessible()`, `uncheckable()`, `not_configured()` factory 方法
- [ ] 验证: 运行现有测试 `pytest tests/test_project_repo_access_check_api.py -v`
- **文件**: `project_repo_access_result.py` (已修改于 DDD 步骤)

### Task 1.3 — resolve_repo_access_tokens 返回 token_status
- [ ] 返回值 dict 增加 `token_status` 字段
- [ ] 区分 `not_bound` (gitOauth 404) vs `token_error` (网络错误/HTTP错误)
- [ ] `token_available` 当成功获取 token 时
- [ ] `not_applicable` 当非 OAuth 仓库时
- [ ] 验证: `pytest tests/test_validate_git_repo_api.py -v`
- **文件**: `projects/services/repo_access_token_resolver.py`

### Task 1.4 — validate_git_repo_view 响应增加字段
- [ ] 响应 JSON 增加 `token_status`, `oauth_provider`, `oauth_service_provider`
- [ ] `project_repo_access_result_to_response()` 同步增加字段
- [ ] 验证: `pytest tests/test_validate_git_repo_api.py -v`
- **文件**: `projects/views/utility_views.py`, `projects/services/project_repo_access_check.py`

### Task 1.5 — _check_gitlab_repo 文案优化
- [ ] `not_bound` 场景文案: "此仓库需要 GitLab 授权后才能访问，请点击下方「OAuth 授权」按钮完成授权"
- [ ] `token_error` 场景文案: "授权令牌获取失败，请稍后重试或重新授权"
- [ ] `token_available` + 仍 404: "授权成功但仓库不存在或无权访问，请检查仓库地址和权限"
- [ ] 验证: `pytest tests/test_validate_git_repo_api.py -v`
- **文件**: `projects/git_utils.py`

### Task 1.6 — 更新现有测试
- [ ] `test_validate_git_repo_api.py`: 新增 not_bound / token_error / token_available 状态断言
- [ ] `test_project_repo_access_check_api.py`: 验证新字段存在于响应中
- [ ] 验证: `pytest tests/test_validate_git_repo_api.py tests/test_project_repo_access_check_api.py -v`
- **文件**: 上述测试文件

## Increment 2: 前端 OAuth 回调自动重校验

### Task 2.1 — CreateProject.vue onMounted 添加 OAuth 回调检测
- [ ] `onMounted` 中调用 `applyOAuthCallbackFromRoute(route, router, callbacks)`
- [ ] success 回调: 遍历 `gitRepoRows` 触发 `scheduleValidateGitRepoRow`
- [ ] error 回调: toast 显示错误信息
- [ ] 验证: 手动在 URL 加 `?gitlab=ok` → 确认重校验触发
- **文件**: `front_project/app/src/views/CreateProject.vue`

### Task 2.2 — 修复 finally 块中的无意义校验
- [ ] 移除 `startRepoOAuthConnect` 的 `finally` 块中 `scheduleValidateGitRepoRow` 调用
- [ ] OAuth 启动后仅做跳转，校验由 Task 2.1 的回调检测接管
- [ ] 验证: OAuth 按钮点击后跳转前不再发出 validate-git-repo 请求
- **文件**: `front_project/app/src/views/CreateProject.vue`

### Task 2.3 — 前端 domain model 测试
- [ ] 新增 `oauth_callback_create_project_revalidate.test.js`
- [ ] 测试: mock route 含 `gitlab=ok` → 断言 `validateGitRepoRow` 被调用
- [ ] 测试: mock route 含 `gitlab=bad_state` → 断言 toast error 显示
- [ ] 测试: mock route 无 OAuth params → 断言无重校验
- [ ] 验证: `npx vitest run oauth_callback_create_project_revalidate.test.js`
- **文件**: `front_project/app/src/tests/domain/oauth_callback/oauth_callback_create_project_revalidate.test.js` (新建)

## Increment 3: token_status 驱动按钮智能显隐

### Task 3.1 — API 响应消费 token_status
- [ ] `validateGitRepoRow` 中解析 API 返回的 `token_status`, `oauth_provider`
- [ ] 存储到 `gitRepoRowTokenStatus` reactive map
- [ ] 验证: 在浏览器 console 检查 token_status 已存储
- **文件**: `front_project/app/src/views/CreateProject.vue`

### Task 3.2 — shouldShowRepoOAuthButton 按 token_status 决策
- [ ] `token_status=not_bound` → 显示 "OAuth 授权" 按钮
- [ ] `token_status=token_error` → 显示 "重试" 按钮 + 错误提示
- [ ] `token_status=token_available` + 不可访问 → 不显示 OAuth 按钮（仓库不存在/无权）
- [ ] `token_status=not_applicable` → 不显示 OAuth 按钮
- [ ] 验证: 针对每种 token_status mock API 返回，检查按钮状态
- **文件**: `front_project/app/src/views/CreateProject.vue`

### Task 3.3 — E2E Playwright 测试
- [ ] 已有 `CreateProject.oauth-button.playwright.test.js` — 更新以覆盖 token_status 驱动逻辑
- [ ] 验证: `npx playwright test CreateProject.oauth-button.playwright.test.js`
- **文件**: `playwright/front_project/tests/CreateProject.oauth-button.playwright.test.js`

## 依赖图

```
Task 1.1 (TokenStatus VO)
  → Task 1.2 (ProjectRepoAccessResult 扩展)
  → Task 1.3 (resolve_repo_access_tokens)
    → Task 1.4 (validate_git_repo_view 响应)
    → Task 1.5 (_check_gitlab_repo 文案)
    → Task 1.6 (更新测试)

Task 1.4 (API 响应含 token_status)
  → Task 2.1 (CreateProject OAuth 回调检测)
    → Task 2.2 (修复 finally 块)
    → Task 2.3 (前端 domain model 测试)

Task 1.4 (API 响应含 token_status)
  → Task 3.1 (前端消费 token_status)
    → Task 3.2 (按钮智能显隐)
    → Task 3.3 (E2E Playwright)
```

## 执行顺序

1. Increment 1 全部任务 (1.1 → 1.6) — **后端基础**
2. Increment 2 全部任务 (2.1 → 2.3) — **前端核心闭环**
3. Increment 3 全部任务 (3.1 → 3.3) — **前端增强**
