# 实施计划: 创建任务项目仓库可访问性标签

> Design: `docs/superpowers/specs/2026-05-30-create-task-project-repo-access-label-design.md`

## Task 1: 领域层 RepoAccessStatus + CheckService

- [ ] **1.1** 新增 `projects/domain/repo_access/value_objects/repo_access_status.py`
- [ ] **1.2** 新增 `projects/domain/repo_access/value_objects/project_repo_access_result.py`
- [ ] **1.3** 新增 `projects/domain/repo_access/services/project_repo_access_check_service.py` + 单测

## Task 2: 后端 API

- [ ] **2.1** RED: `tests/test_project_repo_access_check_api.py`
- [ ] **2.2** GREEN: `projects/services/project_repo_access_check.py` + views + urls
- [ ] **2.3** 旧 `github/repo/access-check` 委托新逻辑

## Task 3: validate-git-repo 收敛

- [ ] **3.1** 抽取 `resolve_repo_access_tokens` helper
- [ ] **3.2** validate_git_repo 使用 `is_gitlab_repo_url` 替代 netloc 启发式

## Task 4: 前端

- [ ] **4.1** `CreateTaskModal.vue` 切换 API + access_status 映射
- [ ] **4.2** 新增 `CreateTaskModal.repoAccess.test.js`

## Task 5: 验收

- [ ] **5.1** pytest 全绿
- [ ] **5.2** Vitest 通过
- [ ] **5.3** 更新 value-stream.yaml step
