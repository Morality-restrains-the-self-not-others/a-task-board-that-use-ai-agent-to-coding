# 实施计划: 配置感知的 GitLab 仓库识别

> 设计: `docs/superpowers/specs/2026-05-30-config-aware-gitlab-repo-detection-design.md`  
> 价值流: `docs/superpowers/plans/2026-05-30-config-aware-gitlab-repo-detection-value-stream.md`

## Task 1: 后端 is_gitlab_repo_url 统一识别

- [ ] **1.1** 新增 `tests/test_git_utils_is_gitlab_repo_url.py`：tencent/synology/unknown/localhost 用例（RED）
- [ ] **1.2** 修改 `projects/git_utils.py`：`is_gitlab_repo_url` 委托 `GitProviderDetectionService`（GREEN）
- [ ] **1.3** 扩展 `test_project_branches_gitlab_auth_guard.py`：IP GitLab 无 token 返回鉴权提示而非 generic（GREEN）

**Verify:** `pytest task2app/Saas_project/tests/test_git_utils_is_gitlab_repo_url.py task2app/Saas_project/tests/test_project_branches_gitlab_auth_guard.py -q`

## Task 2: 前端 catalog 驱动 OAuth 识别

- [ ] **2.1** 扩展 `repoOAuthAuthorizeUtils.test.js`：catalog 匹配 IP GitLab（RED）
- [ ] **2.2** 实现 `loadGitOAuthProviderCatalog` + catalog 匹配（GREEN）
- [ ] **2.3** `ProjectDetail.vue` 挂载时预加载 catalog（GREEN）

**Verify:** `cd task2app/front_project/app && npx vitest run src/utils/repoOAuthAuthorizeUtils.test.js`

## Task 3: value-stream 与 Playwright

- [ ] **3.1** 更新 `value-stream.yaml` step `project-detail-config-aware-gitlab-repo-detection`
- [ ] **3.2** 新增 Playwright 测试（OAuth 按钮可见 + 分支 API 非 generic）

**Verify:** `cd task2app/playwright && npx playwright test front_project/tests/ProjectDetail.branch-preview-config-gitlab.playwright.test.js --project=chromium`
