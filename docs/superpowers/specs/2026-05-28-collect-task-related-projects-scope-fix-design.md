# 设计文档：修复 collect_task_related_projects 返回范围

**日期：** 2026-05-28  
**状态：** 已批准（0-auto-flow 直接启动）  
**触发：** 任务 `846269443533955072` GitLab 推送成功后误跳转 `github.com/ruandao/somanyad/compare/...`

---

## 现象

- 任务仅关联 GitLab 项目 `gitlab-somanyad`（`http://localhost:8012/ljy/somanyad`）。
- 用户在 zTree 点击「推送」成功后，浏览器打开 GitHub compare 页（`ruandao/somanyad`）。
- 用户在「管理项目」中查看 **本任务关联的** `gitlab-somanyad`，未发现 GitHub 仓库。

## 根因（已验证）

`collect_task_related_projects(tenant_id, workspace_id, task_id)` **语义应为「任务关联项目」**，但实现先加载 **工作空间内全部项目**，再对 `TaskProject` 做几乎无效的补全：

```python
workspace_projects = Project.objects.filter(company_id=..., workspaces__id=...)
by_id = {全部 workspace 项目}
# TaskProject 循环仅当 project 不在 by_id 时才加入 — 实际永不触发
return list(by_id.values())
```

数据库实证（任务 `846269443533955072`）：

| 来源 | 项目数 | 说明 |
|------|--------|------|
| `projects_taskproject` | 1 | 仅 `gitlab-somanyad` |
| `collect_task_related_projects` 返回 | 3 | 含未关联任务的 `somanyad`、`somany 用户端和邮件转发`（含 GitHub URL） |

下游 `_first_github_repo_from_task` → `collect_task_github_repos` → `try_create_pull_request_after_layer_push` 因此扫到 **无关项目** 的 `https://github.com/ruandao/somanyad`，生成 `compare_url`；前端 `taskDetailLayerActions.js` 无条件 `window.open(compare_url)`。

---

## 目标

1. **`collect_task_related_projects` 只返回该任务在 `projects_taskproject` 中关联的项目**（限定 tenant + workspace）。
2. 修复 GitLab-only 任务推送后误跳 GitHub compare 页。
3. 所有依赖该函数的 Git OAuth / PR / push auth context 逻辑与任务详情 UI 语义一致。

## 非目标

- 不修改 `ProjectRepo` 表结构或项目管理工作流。
- 不在此变更实现 GitLab MR 自动创建。
- 不重构 `forward_container_layer_git_push` 全链路（除非测试暴露缺口）。

---

## 方案

### A. 修正 `collect_task_related_projects`（必须）

以 `TaskProject` 为唯一入口，JOIN `Project` 并校验 scope：

```python
def collect_task_related_projects(tenant_id, workspace_id, task_id) -> list[Project]:
    task_id_raw = str(task_id or "").strip()
    if not task_id_raw.isdigit():
        return []
    qs = (
        TaskProject.objects.filter(todo_id=task_id_raw)
        .filter(project__company_id=str(tenant_id))
        .filter(project__workspaces__id=str(workspace_id))
        .select_related("project")
        .prefetch_related("project__project_repos")
    )
    by_id: dict[str, Project] = {}
    for tp in qs:
        by_id[str(tp.project_id)] = tp.project
    return list(by_id.values())
```

- 删除「先查 workspace 全部项目」逻辑。
- 更新模块 docstring，删除误导性的「工作空间项目 ∪ …」表述。

### B. 前端 compare_url 打开策略（推荐，小改）

`taskDetailLayerActions.js`：仅当 `pr.html_url` 或用户明确需要手动 PR（`pr_api_error` 等）时打开 `compare_url`；**queued / 纯 skip 且任务无 GitHub 仓时不自动打开**。

可与 A 独立验收；A 修复后 GitLab-only 任务 `github_pull_request` 应为 `skipped: no_github_repo`，前端本不会收到 compare_url。

### C. 测试（必须）

| 层级 | 内容 |
|------|------|
| 单元 pytest | `test_collect_task_related_projects_returns_only_task_linked`：同 workspace 3 项目、任务只链 1 个 → 返回 1 |
| 单元 pytest | `test_collect_task_github_repos_gitlab_only_task`：仅 GitLab 关联 → `collect_task_github_repos` 空 |
| 回归 pytest | `test_github_pr_after_layer_push` GitLab-only → `skipped: no_github_repo` |
| 可选 | 现有 `test_layer_git_push_auth_context.py` / `test_layer_git_push_policy.py` 保持绿 |

---

## 价值流影响

读取 `value-stream.yaml`，受影响流：

| 流 / 步骤 | 影响 |
|-----------|------|
| `layer-oauth-fetch-multi-provider` / push auth context | repos 列表收窄为任务关联项目 |
| `github-pr-create-async` | GitLab-only 任务不再误触发 GitHub compare |
| `task-detail-oauth-binding-guidance` | OAuth 就绪判定与任务仓库一致 |
| `project-detail-repo-oauth-provider-routing` | 复用同一 project 聚合入口 |

**Cross-stream：** 与 `2026-05-27-ztree-push-oauth-precheck-mismatch` 同向——克隆/推送/PR 的 provider 与 **任务关联仓库** 一致。

**Fields：** 无 schema 变更；读路径 `projects_taskproject` → `projects_project` → `projects_projectrepo`。

---

## 领域概念（轻量，供 DDD）

| 概念 | 说明 |
|------|------|
| **Bounded Context: 任务协作** | Task ↔ Project 关联、层推送 |
| **Bounded Context: 项目仓库** | ProjectRepo、TaskProject.repo_address |
| **Entity: TaskProject** | 任务-项目关联根 |
| **Entity: Project** | 含 project_repos |
| **Value Object: TaskScope** | tenant_id + workspace_id + task_id |
| **Domain Event** | （可选）`TaskRelatedProjectsResolved` — 本次为查询语义修复，可不落事件 |

---

## 风险

| 风险 | 缓解 |
|------|------|
| 历史代码依赖「全 workspace 项目」行为 | grep 全部调用点（4 处）；补单元测试 |
| 任务未关联任何项目 → 空列表 | 下游已有 `no_github_repo` / 空 repos 分支；补测试 |
| TaskProject 关联项目不在当前 workspace | filter `project__workspaces__id` 排除跨 workspace 脏数据 |

---

## 验收

1. 任务 `846269443533955072`：`collect_task_related_projects` 仅返回 `gitlab-somanyad`。
2. GitLab 推送成功 → 响应无 GitHub `compare_url` → 浏览器不跳转 GitHub。
3. 任务关联 GitHub 项目时，PR 后续行为与修复前一致。
4. 新增/现有 pytest 全绿。
