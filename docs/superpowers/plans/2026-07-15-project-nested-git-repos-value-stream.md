# Value Stream: 项目详情展示子 Git 仓库列表

> 设计文档: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-design.md`  
> 权限分析: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-permission-analysis.md`  
> 日期: 2026-07-15

## Value Summary

有项目读权限的用户打开项目详情时，对每个已关联的父仓 URL，可直接看到从 `.gitignore` / `.gitmodules` 发现到的**子 Git 仓库列表**（路径 + 推断远程 URL + 来源），无需手工把每个子仓加入 `project_repos`，也无需启动任务/容器。

## Related Value Streams

- **project-git-repos / branch-preview**（既有）：**扩展** — 复用 OAuth token 拉取与分支预览 UX；nested-git-repos 为同级只读发现
- **git-repo-clone-alias**（2026-07-14）：** sibling** — 详情页 Git 区域 UI 同区块；alias 管克隆目录名，本需求管子仓发现展示
- **project-detail-inline-edit**（`ProjectDetailGitReposSection`）：**dependency** — Git OAuth/分支已拆至该组件，子仓列表挂载其下

## End-to-End Flow

```text
[租户成员] 打开项目详情 /tenant/{tid}/projects/{pid}/
  → [Vue] ProjectDetailGitReposSection 挂载 / 关联仓变化
    → [Vue composable] GET /api/tenant/{tid}/projects/{pid}/nested-git-repos/?repo_url=…
      → [Gateway] JWT / forward-auth（tenant + user）
        → [Go taskProjectService] 校验 project 归属 tenant
          → [Go] fetchGitAccessToken(当前用户) + 可选 _gitlab_session
            → [GitLab/GitHub] GET 父仓根 .gitignore、.gitmodules（raw）
              → [Go] parseNestedFromGitignore + parseGitmodules + sibling URL 推导 + merge
                → [Vue] 展示「子 Git 仓库」列表（path / url / source）；Loading / error / 空三态
```

**失败分支（不阻断详情页）**：

```text
无 OAuth → 200 { nested_repos: [], error: "请先连接 GitLab/GitHub…" }
.gitignore/.gitmodules 404 → 视为空段，继续合并
项目无关联仓 → 400 No repository URL provided
```

## 最小增量切片列表

### Slice 1: 解析纯函数（无网络）

**Value:** 发现算法可单测、可复现，与 ram-work 真源一致

**Scope:**
- `parseNestedFromGitignore` — `# Nested git repos` 段
- `parseGitmodules` — 标准 ini
- `deriveSiblingRepoURL` — 单段 path 兄弟 URL
- `mergeNestedRepos` — 按 path 去重，gitmodules 优先

**Depends on:** 无

**验证:** `go test taskProjectService/src/ -run 'Nested|Gitmodules|Sibling|Merge' -v`

---

### Slice 2: 远端拉文件 + HTTP handler

**Value:** 后端可返回真实 nested 列表（或 OAuth 失败友好文案）

**Scope:**
- `fetchRepoRawFile`（GitLab/GitHub，main→master fallback）
- `handleProjectNestedGitRepos` + 路由 `nested-git-repos`
- OpenAPI schema 注册
- handler 单测（mock 远端）

**Depends on:** Slice 1

**验证:** `go test taskProjectService/src/ -run 'NestedGitRepos|HandleProjectNested' -v`

---

### Slice 3: 治理登记 + 前端展示

**Value:** 用户可在详情页看到子仓列表并刷新

**Scope:**
- `db/api_route_ownership.yaml` 登记
- `useProjectNestedGitRepos.js` composable
- `ProjectDetailGitReposSection.vue` 子区块 + testid
- intents 文档

**Depends on:** Slice 2

**验证:** 前端 unit + 可选 Playwright `project-detail-nested-git-repos`

---

### Slice 4: 架构 SSOT + 可选 E2E

**Value:** 架构可追踪；回归不破坏详情页

**Scope:**
- v28 application-integration 制品
- `VERSION_HISTORY.md` 条目
- Playwright 冒烟（可选）

**Depends on:** Slice 3

**验证:** Archi `--loadModel` + CI ownership check

## Fields Impact

| 字段 / 资源 | 变更 |
|-------------|------|
| `project_repos` | **无写入**（只读发现） |
| API 响应 `nested_repos[].path/url/source` | 新增只读 DTO |
| 前端 `data-testid="project-detail-nested-git-repos"` | 新增 UI 锚点 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版价值流与四切片 |
