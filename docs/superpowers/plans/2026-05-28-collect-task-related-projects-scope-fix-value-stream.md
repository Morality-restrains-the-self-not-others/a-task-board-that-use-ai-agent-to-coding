# 价值流：collect_task_related_projects 范围修复

> 设计：`docs/superpowers/specs/2026-05-28-collect-task-related-projects-scope-fix-design.md`

## Related Value Streams

| 既有流 | 关系 |
|--------|------|
| `2026-05-27-relay-localhost-gitlab-push-stuck` | 同任务 846269443533955072；推送容器侧已修，PR 误扫 workspace 为剩余根因 |
| `2026-05-27-ztree-push-oauth-precheck-mismatch` | 同向：任务仓库 provider 与后续逻辑一致 |

## 用户价值流

```
[任务关联 GitLab 项目] → [zTree 推送成功] → [PR 后续仅扫任务关联项目] → [无 GitHub 仓则 skip] → [不跳转 GitHub]
```

## 增量

### Increment 1 — 修正项目聚合查询（必须）

- 修改 `projects/services/task_branch_projects.py`
- 新增 `tests/test_task_branch_projects.py`
- 验收：同 workspace 多项目、任务只链 1 个 → 返回 1

### Increment 2 — PR 后续回归（必须）

- 扩展 `tests/test_github_pr_after_layer_push_async.py` 或新用例：GitLab-only TaskProject → `skipped: no_github_repo`
- 验收：任务 846269443533955072 场景不再生成 compare_url

## 测试映射

| 步骤 | test_file |
|------|-----------|
| 任务项目聚合 | `tests/test_task_branch_projects.py` |
| GitHub PR skip | `tests/test_github_pr_after_layer_push_async.py` |
| 回归 | `tests/test_layer_git_push_auth_context.py`, `tests/test_layer_git_push_policy.py` |
