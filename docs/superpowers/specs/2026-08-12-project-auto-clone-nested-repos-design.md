# 创建项目可选「自动克隆子仓库」

- **日期**: 2026-08-12
- **状态**: accepted（goal-mode 自动采用）
- **作者**: cursor
- **入口**: `/goal` 页面元素调整 — CreateProject 允许选择是否自动克隆子仓库
- **相关意图**: `project_nested_git_repos`、`container_nested_git_repos_clone`

## 问题

1. 容器 bootstrap 通过 `taskCredentialService.MergeNestedReposIntoSnapshots` **无条件**把 `.gitmodules` 发现的子仓合并进克隆列表；元仓（如 ram-work）会并发克隆数十子仓，成本高且不可关闭。
2. 创建项目页无任何开关；用户只能事后在任务/容器侧被动承受。
3. 项目详情「未发现子仓库」空态与「是否自动克隆」是正交概念：前者是发现结果，后者是克隆策略；需在 UI 上区分，并允许创建时选定策略。

## 目标（验收）

1. **创建项目**表单在填写 ≥1 个 Git 仓库 URL 时，展示勾选「自动克隆子仓库」，`data-testid="create-project-auto-clone-nested-repos"`。
2. 勾选状态随 `POST /api/projects/...` 字段 `auto_clone_nested_repos`（boolean）持久化到 `project_entries`。
3. **默认 `true`**（存量与新建默认保持现网「会 enrich 克隆」行为，向后兼容）。
4. 项目详情可查看/切换同一开关（PATCH），且切换即时影响后续容器 `task-detail` / `repo-clone-credentials` enrich。
5. `auto_clone_nested_repos=false` 时：`MergeNestedReposIntoSnapshots` **跳过**该项目；父仓仍克隆；发现 API（`nested-git-repos`）行为不变。
6. 空态「未发现子仓库」保留；当开关关闭时额外提示「已关闭自动克隆，容器将仅克隆父仓库」。
7. **任务详情**「关联项目」下：开关关闭时**不**展示「子仓库克隆状态」进度面板（避免误导为正在克隆），改为提示「已关闭自动克隆子仓库：容器将仅克隆父仓库」。

## 非目标

- 不把子仓持久化进 `project_repos`（仍只读 enrich）。
- 不改 `git clone --recurse-submodules` 语义（仍按 `.gitmodules` 注册表独立 clone）。
- 不强制新 MQ 业务事件。
- 不改动 APISIX / Django 公网路由（沿用现有 projects CRUD）。

## 方案（采用）

### A. Schema

`dataMigrate/taskProjectService/012_auto_clone_nested_repos.sql`：

```sql
ALTER TABLE project_entries
  ADD COLUMN auto_clone_nested_repos TINYINT NOT NULL DEFAULT 1
  COMMENT '1=容器 bootstrap 自动 enrich 并克隆 .gitmodules 子仓';
```

表前缀合规：`project_`。冷热：配置型小表，无分区需求。

### B. taskProjectService API

- Create / Update / Detail / List detail 读写 `auto_clone_nested_repos`（JSON bool）。
- 缺省 / 旧客户端不传 → 视为 `true`。
- 内部 batch-get / 详情供 taskTaskService 读取。

### C. 下游传播

```
CreateProject(FE)
  → taskProjectService.project_entries.auto_clone_nested_repos
  → 生产：taskCredentialService SQLiteBusinessRepository 直读 project_entries
     （HTTP fallback：taskTaskService container-snapshot → HTTPBusinessRepository）
  → MergeNestedReposIntoSnapshots：仅当 AutoCloneNestedRepos==true 才 fetch+merge
  → onlineServiceJS collectRepoCloneJobs：flag=false 时跳过 parent_repo_url 子仓
```

### D. 前端

| 面 | 行为 |
|----|------|
| CreateProject | checkbox，有仓库 URL 时显示；默认勾选；提交带字段 |
| ProjectDetailGitReposSection | 开关 + 文案；PATCH 保存；空态区分「未发现」vs「已关闭自动克隆」 |
| TaskDetailNestedReposCloneStatus | `auto_clone_nested_repos=false` 时隐藏克隆状态，展示关闭提示 |

### E. 架构 / ADR

- **无新服务/新组件拓扑** → 不升 enterprise-landscape 版本。
- **No-ADR**: trivial tech choice（项目级 bool 配置），无架构选型。

## 方案对比（已决策）

| 方案 | 说明 | 结论 |
|------|------|------|
| A 项目级 bool + enrich 门控 | 创建/详情可配；传播经 snapshot | **采用** |
| B 仅前端本地、不持久化 | 无法影响容器 | 拒：无效 |
| C 任务级开关 | 每次建任务重选 | 拒：用户要求创建项目时选择 |
| D 默认 false | 破坏现网元仓体验 | 拒：默认 true 兼容 |

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 已执行（软依赖通过）。
- 影响符号：`MergeNestedReposIntoSnapshots` → callers `BuildRepoCloneCredentials` / `FetchTaskDetail`；`handleCreateProject` / `loadProjectDetail`；FE `CreateProject.vue` / `ProjectDetailGitReposSection.vue`。
- 爆炸半径：taskProjectService + taskTaskService(container-snapshot) + taskCredentialService + taskFE；无跨计费/鉴权新面。

## 测试策略

| 层 | 内容 |
|----|------|
| Go Project | create/update/get 字段；缺省 true |
| Go Credential | Merge 跳过 `AutoCloneNestedRepos=false` 项目 |
| Go Task | snapshot 透传字段 |
| FE unit | CreateProject checkbox 提交；Detail 开关 PATCH |
| Playwright | CreateProject 勾选可见并可提交 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-12 | goal-mode 初版并自动采用 |
| 2026-08-13 | 任务详情：关闭自动克隆时隐藏「子仓库克隆状态」，展示禁用提示 |
| 2026-08-15 | 修复生产 SQLite 直连路径写死 true；镜像 collectRepoCloneJobs 增加 false 门控 |
| 2026-08-15 | identities 为空时仍 enrich（匿名 GitHub）；任务级状态不再用 bootstrapCloneDone 假完成 |
| 2026-08-16 | 私有仓 nested 发现/克隆凭证改用评论 created_by_id；禁止 task owner_id 回退 |
