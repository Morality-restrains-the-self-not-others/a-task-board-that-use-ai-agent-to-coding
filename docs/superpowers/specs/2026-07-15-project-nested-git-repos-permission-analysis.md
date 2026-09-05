# 角色权限分析：项目详情展示子 Git 仓库列表

**日期**: 2026-07-15  
**设计文档**: `docs/superpowers/specs/2026-07-15-project-nested-git-repos-design.md`  
**状态**: 完成

## 结论摘要

新增 `GET /api/tenant/{tenant_id}/projects/{project_id}/nested-git-repos/` 为**只读查询**，权限边界须与**项目详情读权限**一致：调用者必须是**该租户成员**且**能读取目标 project**；禁止跨租户 IDOR；出站 GitLab/GitHub 请求仅使用**当前登录用户**的 OAuth token（`fetchGitAccessToken(userID, …)`），不得复用他人 token 或全局服务账号读取私有仓内容。

无新 RBAC 角色；不写入 `project_repos`；无 privilege escalation 面。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET …/nested-git-repos/` | 租户成员（viewer / editor / admin 等具备项目读权者） | Tenant → Project | read | 网关 JWT / forward-auth → `X-Auth-User-Id` + `X-Auth-Tenant-Id`；路径 `tenant_id` 与 JWT tenant 一致（APISIX / taskAuth） | ⚠️ 须与 branches 对齐 | handler 内 `loadProjectDetail(projectID)` + `detail.company == tenantID`，否则 404（同 `handleProjectBranches`） |
| 路径 `tenant_id` 与 project 归属 | 租户成员 | Tenant / Project | read | URL 段 `tenant_id` 写入 `X-Auth-Tenant-Id`；JWT tenant 由网关校验 | ✅ 网关层 | handler 二次校验 project.company_id，防 IDOR |
| 跨租户 project_id 枚举 | 非成员 / 外租户用户 | Project | read | 404 而非 403（与现有 project 读一致） | ✅ 模式已有 | 保持 404，不泄露 project 是否存在 |
| `repo_url` query 可选覆盖 | 租户成员 | Project → Git Repo（元数据） | read | 缺省取项目第一个关联仓；须为该项目已关联 URL 或合法 query | ⚠️ 须补 | 若提供 `repo_url`，校验其属于该项目 `git_repos` / `git_repo_entries`，否则 400 |
| 出站 GitLab/GitHub raw 文件 | 当前用户 | 外部 Git Provider | read | `fetchGitAccessToken(getAuthUser(r), provider, …)`；可选 `_gitlab_session` cookie（与分支预览一致） | ✅ 复用既有 | **禁止**用 project 所有者或其他用户 id 取 token；单测断言 userID 来自当前请求 |
| 无 OAuth / token 失效 | 租户成员 | Project | read | 200 + `nested_repos: []` + `error` 中文提示（对齐分支预览） | ✅ 设计已含 | 不得因无 token 返回 401（项目读权仍成立）；不得回显 token |
| 前端 `ProjectDetailGitReposSection` 自动拉取 | 能打开项目详情页的用户 | Project | read | 前端路由 `/tenant/…/projects/{id}/` 既有租户守卫 | ✅ | composable 仅传当前 tenant/project；不缓存跨账号 |
| `db/api_route_ownership` 登记 | — | — | — | Go `taskProjectService` owner | ⚠️ 交付项 | T4 登记，禁止 Django 新增公网路由 |
| Swagger / OpenAPI | — | — | — | taskProjectService schema | ⚠️ 交付项 | T3 同步 path 与响应 schema |

## 与既有端点对齐

| 参照端点 | 权限模式 | nested-git-repos 对齐方式 |
|----------|----------|---------------------------|
| `GET …/projects/{id}/` | 租户成员 + project 归属 | 相同 project 读边界 |
| `GET …/projects/{id}/branches/?repo_url=` | project 归属 + 当前用户 OAuth | **完全对齐**：同一 `loadProjectDetail` + `getAuthUser` + `fetchGitAccessToken` |
| `GET …/repo-access-check/` | project 归属 + 用户 token 探测 | 读权一致；nested 不额外扩权 |

## IDOR / 串租户 / Token 风险

| 风险 | 缓解 |
|------|------|
| 用 A 租户 token 读 B 租户 project | `detail.company != tenantID` → 404 |
| 用用户 A 的 session 读用户 B 有权限而 A 无权限的私有仓 | OAuth token 仅 `getAuthUser(r)`；Git 侧 401/404 映射为 `error` 文案，不升级项目读权 |
| `repo_url` 指向项目未关联的外部仓 | query 须校验属于项目关联列表，防 SSRF/越权探测 |
| 子仓 URL 推导泄露未授权命名空间 | 仅展示解析结果；无 url 时 `resolve_error`，不伪造可点击链接 |
| 自动写入 `project_repos` 造成写权限绕过 | **非目标**；Constraint：只读发现，handler 禁止 INSERT/UPDATE project_repos |

## 新角色

无。不引入新 RBAC 角色或权限粒度。

## 实现检查清单（Build 阶段）

- [ ] handler 复制 `handleProjectBranches` 的 project 归属校验
- [ ] `fetchGitAccessToken` 第一个参数必须为 `getAuthUser(r)`
- [ ] 单测：跨 tenant project_id → 404
- [ ] 单测：无 token → 200 + empty list + error 文案
- [ ] 单测：mock 远端时 assert 未调用 DB 写 `project_repos`

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：对齐项目详情读权限 + 当前用户 OAuth |
