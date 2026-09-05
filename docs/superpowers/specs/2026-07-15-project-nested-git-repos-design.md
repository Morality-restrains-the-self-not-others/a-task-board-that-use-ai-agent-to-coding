# 项目详情：展示 Git 仓库的子 Git 仓库列表

日期：2026-07-15  
状态：已批准（goal-mode 自动采用；2026-07-17 对齐 Git 标准，无 gitignore 回退）  
作者：claude

## 问题

项目详情页（例：`/tenant/.../projects/proj_-5247879312070945751/`）对从 GitLab 同步的元仓（如 `ram-work`）仅展示**单一**关联仓库 URL。该元仓在本地/工作区布局下包含大量**独立嵌套 Git 子仓库**（`task2app/`、`docs/`、`runAll/` 等，各自有 `.git`）。用户期望在项目详情 **Git 仓库** 区域**直接看到子 Git 仓库列表**，无需手工把每个子仓加入 `project_repos`。

## 目标

1. 有项目读权限的用户打开项目详情时，对每个已关联的父仓 URL，可看到其发现到的**子 Git 仓库列表**（路径 + 远程 URL + 发现来源）。
2. 发现逻辑符合 Git 对 submodule 配置的约定：以 `.gitmodules` 的 `path` + `url` 为**唯一** SSOT；相对 URL 按 `git help submodule` 相对父仓 default remote 解析。
3. 复用现有 GitLab/GitHub OAuth 拉文件能力；无授权时给出与分支预览一致的可操作提示。
4. 零 Django 新公网路由；落点 Go `taskProjectService`。

## 非目标

- 不自动把子仓写入 `project_repos`（不做「一键关联」MVP）。
- 不扫描容器内已克隆目录（详情页无任务/容器依赖）。
- 不改变多仓 CRUD、OAuth、clone_alias、分支预览既有语义。
- 不强制父仓对子路径登记 gitlink / 执行 `git submodule update`（元仓仍可独立克隆子目录；`.gitmodules` 作 path↔url 注册表）。
- **不**从 `.gitignore` 推断子仓列表或 URL（无 `.gitmodules` → 空列表）。

## 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 读父仓 `.gitmodules` + Git 相对 URL 解析 | 符合 Git submodule 配置与 URL 语义 | 元仓需维护 `.gitmodules` | **唯一路径，采用** |
| B. 读父仓 `.gitignore` Nested 段 | 早期无清单文件 | **不符合** Git 约定；ignore ≠ 子仓注册 | **不采用、不回退** |
| C. 读专用 `nested-git-repos.yaml` | 机器友好 | 非 Git 标准 | 不做 |
| D. GitLab Group 下列出全部项目 | 覆盖广 | 噪声大、非「该仓子目录」语义 | 不用 |

**采用：仅 A。**

## 发现算法

### 1. 拉取父仓根文件

对 `repo_url`（默认项目 `git_repos[0]`，或 query `repo_url`）：

- GitLab：`GET /api/v4/projects/{path}/repository/files/{file_path}/raw?ref={default_branch}`（先尝试 `main`，失败再 `master`）
- GitHub：`GET /repos/{owner}/{repo}/contents/{path}` → raw
- 鉴权：复用 `fetchGitAccessToken` + 可选 GitLab session cookie（与分支列表一致）

文件：**仅** `.gitmodules`（404 / 空 → `nested_repos: []`，不读 `.gitignore`）。

### 2. 解析 `.gitmodules`（唯一 SSOT）

标准 ini：`[submodule "x"]` → `path` + `url`。

`url` 解析（对齐 `git help submodule`）：

- 绝对 URL（含 `scheme://` 或 scp-like `user@host:path`）：原样使用。
- 相对 URL：相对**父仓 remote**，且「按相对目录」求值——父仓 remote 视为目录。因此与 `ram-work.git` 同级的兄弟仓须写 `../foo.git`（不是 `./foo.git`）。
- 例：父仓 `https://host/group/ram-work.git` + `../task2app.git` → `https://host/group/task2app.git`。

### 3. 输出

按 `path` 字典序排序；`source` 恒为 `gitmodules`。

## API

```
GET /api/tenant/{tenant_id}/projects/{project_id}/nested-git-repos/
```

Query：

| 参数 | 说明 |
|------|------|
| `repo_url` | 可选；缺省为项目第一个关联仓 |

响应 200：

```json
{
  "parent_repo_url": "https://gitlab.daydaymoney.com/example-user/ram-work.git",
  "nested_repos": [
    {
      "path": "task2app",
      "url": "https://gitlab.daydaymoney.com/example-user/task2app.git",
      "source": "gitmodules"
    }
  ],
  "error": ""
}
```

`source` ∈ `gitmodules`。  
无授权 / 拉文件失败：`nested_repos: []`，`error` 为用户可读中文（对齐分支预览口吻）。  
项目无关联仓：400 `No repository URL provided`。

**纯查询例外**：不投递领域事件（只读发现，无业务状态变更）。

Swagger：在 `taskProjectService` OpenAPI schema 注册该 path。

## 前端

`ProjectDetailGitReposSection.vue`：

- 在已有关联仓列表下方新增 **「子 Git 仓库」** 区块（`data-testid="project-detail-nested-git-repos"`）。
- 进入页面 / 关联仓变化后自动请求；提供「刷新」按钮。
- 每行：`path` + 可点击 `url`（无 url 时仅 path + 错误提示）。
- Loading / error / 空列表（「未发现子仓库」）三态。

逻辑放 `useProjectDetailGitRepos.js` 或新 composable `useProjectNestedGitRepos.js`（控制组件行数）。

## 落点与所有权

| 项 | 落点 |
|----|------|
| HTTP | Go `taskProjectService` |
| 表 | 无新表（只读） |
| 路由登记 | `db/api_route_ownership.yaml` + Swagger |
| 意图 | `docs/intents/frontend/project_nested_git_repos.intent.md` + test-intent |
| 元仓清单 | 根目录 `.gitmodules`（唯一 SSOT）；`.gitignore` 仅忽略工作树，不参与发现 |

## 架构变更影响

- **需要**：新增 Rel_Flow：`taskProjectService` → GitLab/GitHub「读仓库根文件」；前端 → 新只读 API。
- **迭代版本**: v28 🎯 target（application-integration）
- **说明**: 无新微服务、无新表；零 Python 公网接口。

### 伴生架构文件（将写入）

- `docs/architecture/v28-application-integration-20260715-1135-claude.puml`
- `.archimate` + `.mermaid.md`
- 更新 `VERSION_HISTORY.md`

## 测试

| 层 | 内容 |
|----|------|
| Go unit | gitmodules；Git 相对 URL（含 `../sibling.git`、scp-like）；无 `.gitmodules` 时空列表（不读 gitignore） |
| Go handler | mock 远端 `.gitmodules` → 200 列表；仅有 gitignore → 空列表；无 token → error 文案 |
| 前端 unit | composable / 区块三态 |
| Playwright（可选增量） | 详情页可见 nested section testid |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | goal-mode 初版并自动采用（主路径曾为 gitignore + 兄弟 slug 拼装） |
| 2026-07-17 | 纠正为 Git 标准：`.gitmodules` SSOT + `git-submodule` 相对 URL 解析 |
| 2026-07-17 | 移除 `.gitignore` 回退：无 `.gitmodules` 一律空列表 |
