# 设计文档：GitLab 同步 — 合并/单仓导入 eligibility 规则分化

**日期**：2026-07-05  
**状态**：已实现  
**迭代**：GitLab 项目同步 — 双按钮导入 eligibility  
**关联页面**：`/tenant/{tenantId}/projects/` →「从 GitLab 同步」

---

## 1. 背景与问题

### 1.1 现状

项目列表页「从 GitLab 同步」已支持两种创建方式（见上一轮实现）：

| 按钮 | API | 当前行为 |
|------|-----|----------|
| 合并为一个项目 | `POST .../combined-from-gitlab-repos/` | 跳过租户内**任意项目**已关联的仓库 |
| 每个仓库各建一个项目 | `POST .../batch-from-gitlab-repos/` | 同上 |

列表 API `GET .../gitlab-remote-repos/` 对每个仓库返回单一布尔字段 `already_imported`：只要 `ProjectRepo` 在租户内存在即标记为 true。前端据此：

- 禁用 checkbox
- 不计入「可导入」计数
- 合并/批量创建均排除这些仓库

### 1.2 业务诉求

用户希望两种模式的**可导入判定**不同：

| 模式 | 规则 |
|------|------|
| **合并为一个项目** | **不限制**仓库是否已在其他项目中导入；任意 GitLab 仓库均可勾选并关联到新合并项目 |
| **每个仓库各建一个项目** | 仅当仓库**尚未**被导入到「单仓库项目」时才允许；已在单仓库项目中的仓库不可再 1:1 创建 |

### 1.3 术语

| 术语 | 定义 |
|------|------|
| **单仓库项目** | 租户内 `Project`，且其 `project_repos` 数量为 **1** |
| **多仓库项目** | 租户内 `Project`，且其 `project_repos` 数量 **≥ 2** |
| **单仓已占用** | 仓库 URL（canonical key）已作为唯一 `ProjectRepo` 存在于某个单仓库项目中 |

> 说明：同一 `repo_url` 可出现在多个项目的 `ProjectRepo` 中（DB 约束为 `(project, repo_url)` 唯一，非租户级唯一）。合并模式允许跨项目重复关联同一 URL。

---

## 2. 目标与非目标

### 2.1 目标

1. 列表 API 区分「单仓已占用」与「任意已导入」，供前端分模式展示与校验。
2. 合并创建：不因仓库已在其他项目而 skip；所选仓库全部写入新项目的 `ProjectRepo`。
3. 批量（每仓一项目）创建：仅 skip/error 单仓已占用的仓库；未占用或仅存在于多仓库项目中的仓库可创建。
4. 前端 UI 与按钮 enabled 条件与上述规则一致。
5. 单元测试 + Playwright 覆盖新规则。

### 2.2 非目标

- 不修改 `ProjectRepo` 表结构或租户级唯一约束。
- 不阻止用户在项目详情页手动添加已存在于其他项目的仓库 URL（若现有能力支持）。
- 不涉及 GitLab OAuth / 远程列表拉取逻辑变更。
- **不更新 ArchiMate 架构**（纯业务规则 + API 字段细化，无新增服务组件）。

---

## 3. 领域概念清单（供 DDD 参考）

| 类型 | 概念 |
|------|------|
| Bounded Context | `projects` — 项目与仓库关联 |
| Entity | `Project`、`ProjectRepo` |
| 值对象 | `CanonicalRepoKey`（URL 规范化） |
| 领域规则 | `SingleRepoProjectOccupancy` — 判断 URL 是否被单仓库项目占用 |
| 领域服务 | `GitlabRemoteProjectSyncService` — 列表 eligibility + 两种创建策略 |

---

## 4. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `project-workspace` / `project-crud` | 项目创建 | GitLab 同步两条路径的 eligibility 分化 |
| 新增测试 | — | `test_gitlab_remote_projects_api.py`、`useGitlabProjectSync.test.js`、`Projects.gitlab-sync.playwright.test.js` |

`value-stream.yaml` 暂无 dedicated `gitlab-sync` step；可在后续 value-stream 步骤中补充 `projects_projectrepo.repo_url` 的同步场景说明（非阻塞）。

---

## 5. 方案设计

### 5.1 后端 — 单仓占用检测

在 `gitlab_remote_projects.py` 新增：

```python
def _collect_single_repo_occupied_keys(company_id) -> set[str]:
    """
    返回已被「单仓库项目」占用的 repo canonical keys。
    实现：ProjectRepo JOIN Project，按 project_id 分组，count==1 的项目的 repo_url。
    """
```

可选辅助（列表展示用）：

```python
def _collect_any_imported_keys(company_id) -> set[str]:
    """租户内任意 ProjectRepo 出现过的 URL（保留用于展示「已在多仓项目中」）。"""
```

**性能**：单次查询 + Python 聚合，或 ORM `annotate(Count('project_repos'))` 过滤 `repo_count=1` 后收集 URL。租户项目量级通常可接受。

### 5.2 后端 — 列表 API 响应字段

`GET /api/tenant/{tid}/projects/gitlab-remote-repos/` 每个 repo 对象：

| 字段 | 类型 | 含义 |
|------|------|------|
| `imported_in_single_repo_project` | bool | **批量模式**禁用依据 |
| `imported_in_any_project` | bool | 可选，用于 UI 提示「已在其他项目中」 |
| ~~`already_imported`~~ | — | **废弃**（前端/tests 迁移至新字段；可短期双写 `already_imported = imported_in_single_repo_project` 兼容旧前端，本迭代直接切换） |

**示例**：

```json
{
  "name": "valuestream",
  "http_url_to_repo": "http://183.250.1.132:8012/group/valuestream.git",
  "imported_in_single_repo_project": true,
  "imported_in_any_project": true
}
```

### 5.3 后端 — 合并创建 `combined-from-gitlab-repos`

变更点：

- **移除**「`key in existing_keys` → skip」逻辑。
- 对请求内每个合法 URL：规范化后直接 `ProjectRepo.objects.create(project=new_project, repo_url=...)`。
- 请求内 URL 去重（canonical key）。
- 若规范化失败 → `errors`。
- **不再**因租户内已存在而 skip。
- 成功条件：至少 attach 1 个仓库（仍要求 `len(repos) >= 2` 的 API 校验不变）。

返回体 `skipped` 在合并模式下通常为空；保留字段结构兼容。

### 5.4 后端 — 批量创建 `batch-from-gitlab-repos`

变更点：

- 将 skip 条件从「任意已导入」改为「`key in single_repo_occupied_keys`」。
- `skipped[].reason` 由 `already_imported` 改为 `single_repo_project_occupied`（或保留 reason 字符串，文档说明即可）。

仍保留「项目名称已存在」等既有校验。

### 5.5 前端 — `useGitlabProjectSync.js`

| 计算属性 / 逻辑 | 变更 |
|----------------|------|
| `batchSelectableRepos` | `repos.filter(r => !r.imported_in_single_repo_project)` |
| `mergeSelectableRepos` | `repos`（全部） |
| `selectedReposForBatch` | 已勾选 ∩ batchSelectable |
| `selectedReposForMerge` | 已勾选（全部，含单仓已占用） |
| `selectedCount` | 已勾选总数（全选旁「已选 N / M」与两按钮括号内展示） |
| `batchEligibleCount` | 已勾选且可批量创建的数量 |
| 预勾选 | 打开弹窗：默认勾选**全部**仓库（合并无限制）；或仅勾选 batch 可用 — **决策：默认勾选全部**，便于合并场景 |

`createSelectedProjects` 提交 `selectedReposForBatch`。  
`createCombinedProject` 提交 `selectedReposForMerge`（≥2）。

### 5.6 前端 — `GitlabSyncProjectsModal.vue`

| UI 元素 | 行为 |
|---------|------|
| Checkbox | **不再**因导入状态 disabled；所有仓库可勾选 |
| Badge | `imported_in_single_repo_project` →「单仓已占用」；`imported_in_any_project && !single` →「已在多仓项目中」（灰色提示，仍可合并） |
| 「每个仓库各建一个项目」按钮 | `disabled` 当 `batchEligibleCount === 0` |
| 「合并为一个项目」按钮 | `disabled` 当 `selectedCount < 2`（与导入无关） |

### 5.7 行为矩阵（验收用）

假设租户内：仓库 A 仅在单仓项目 P1；仓库 B 仅在多仓项目 P2；仓库 C 未导入。

| 仓库 | 合并可选 | 批量可选 | 合并创建 | 批量创建 |
|------|----------|----------|----------|----------|
| A | ✅ | ❌ | ✅ attach 到新项目 | skip |
| B | ✅ | ✅ | ✅ attach 到新项目 | ✅ 新建单仓项目 |
| C | ✅ | ✅ | ✅ | ✅ |

合并选中 A+B：新项目含 2 条 `ProjectRepo`（A 在 P1 仍存在，不删除）。

---

## 6. API 变更摘要

| 端点 | 变更 |
|------|------|
| `GET .../gitlab-remote-repos/` | repo 项新增 `imported_in_single_repo_project`、`imported_in_any_project`；移除 `already_imported` |
| `POST .../combined-from-gitlab-repos/` | 不再 skip 已导入 URL |
| `POST .../batch-from-gitlab-repos/` | 仅 skip 单仓已占用 URL |

Swagger：同步更新上述三端点的 response/request 说明（遵循 API Swagger 元规则）。

---

## 7. 测试计划

### 7.1 后端 `test_gitlab_remote_projects_api.py`

- 单仓项目占用 → 列表 `imported_in_single_repo_project=true`
- 仅多仓项目占用 → `imported_in_single_repo_project=false`, `imported_in_any_project=true`
- 合并创建：单仓已占用 URL 仍写入新项目
- 批量创建：单仓已占用 skip；多仓已占用可 create

### 7.2 前端 Vitest

- composable 分模式 selected 集合
- 合并 API payload 含单仓已占用 repo

### 7.3 Playwright

- Mock：A 单仓已占用 + B 未占用 → 合并按钮可点且 POST combined 含 A、B；批量仅 POST B
- Badge 文案更新

---

## 8. 风险与注意事项

| 风险 | 缓解 |
|------|------|
| 同一 URL 多项目关联导致任务/推送语义混淆 | 合并场景为用户主动操作；文档说明「合并不迁移旧项目上的任务关联」 |
| 列表 API 多一次聚合查询 | 与 membership 列表相比开销小；必要时缓存 single_repo keys |
| 旧客户端读 `already_imported` | 本迭代同仓部署前后端；无外部 API 消费者 |

---

## 9. 实施清单（批准后）

- [ ] `gitlab_remote_projects.py` — 检测函数 + 两 create 路径
- [ ] `utility_views.py` — 无路由变更
- [ ] `GitlabSyncProjectsModal.vue` — checkbox/badge/按钮
- [ ] `useGitlabProjectSync.js` — 分模式 selection
- [ ] 测试三件套 + Swagger
- [ ] `Projects.gitlab-sync.playwright.test.js.testIntent` 更新

---

## 10. 架构变更影响

**无** — 不新增/移除 Application 组件，不修改 `docs/architecture/`。当前基线 v5 ✅ shipped。

---

## 11. 待确认（可选）

1. **合并模式默认勾选**：是否默认勾选全部仓库（含单仓已占用）？本文建议 **是**。
2. **多仓项目中的仓库再 1:1 创建**：按需求允许；若未来需禁止「任意已导入」，可再加开关。

---

**批准后即可进入 `/8-build-构建` 或 `/3-value-stream-价值流` 流水线。**
