# 实施计划：collect_task_related_projects 范围修复

## Task 1 — 失败测试：仅返回任务关联项目

- [ ] Create `task2app/Saas_project/tests/test_task_branch_projects.py`
- [ ] 场景：workspace 2 项目，任务只 TaskProject 链 1 个 → `len(collect_task_related_projects(...)) == 1`
- [ ] Run: `cd task2app/Saas_project && pytest tests/test_task_branch_projects.py -v` → RED

## Task 2 — 实现修正

- [ ] Modify `projects/services/task_branch_projects.py` — TaskProject 为入口，filter tenant+workspace
- [ ] Run pytest Task 1 → GREEN

## Task 3 — PR 后续 GitLab-only 回归

- [ ] Extend `tests/test_github_pr_after_layer_push_async.py` — GitLab-only task → `skipped: no_github_repo`
- [ ] Run: `pytest tests/test_github_pr_after_layer_push_async.py tests/test_task_branch_projects.py -v`

## Task 4 — 全量回归

- [ ] Run: `pytest tests/test_layer_git_push_auth_context.py tests/test_layer_git_push_policy.py -v`
