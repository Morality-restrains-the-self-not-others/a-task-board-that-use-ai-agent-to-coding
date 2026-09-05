# 角色权限分析：容器 task-detail 下发最新子 Git 仓库并并发克隆

**日期**: 2026-07-15  
**设计文档**: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`  
**状态**: 完成

## 结论摘要

本需求为**只读 enrich + 容器侧克隆执行**，不写入 `project_repos`、不新增 Django 公网路由、不引入新 RBAC 角色。权限边界分三层：

1. **内部发现 API**（`GET /api/internal/nested-git-repos/`）：仅服务间调用；`user_id` 须来自任务已绑身份，禁止任意用户 ID 探测他人私有仓。
2. **容器契约 API**（`task-detail` / `repo-clone-credentials`）：沿用既有 **server-container-token** 任务作用域；enrich 后的子仓 URL 凭证**继承**同任务父仓身份，不得跨任务/跨用户复用 token。
3. **容器克隆**：在已换票成功的 bootstrap 上下文内执行；子仓目录名由 `clone_alias`（= nested path）决定，不参与鉴权。

nested 发现失败**不阻断**父仓克隆；无 privilege escalation 面。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET /api/internal/nested-git-repos/?repo_url=&user_id=` | taskCredentialService（服务间） | Internal → Git Provider（经 ProjectService） | read | 内部 HTTP；非公网暴露 | ⚠️ 须约束 | 仅监听内网/loopback；校验 `user_id>0`；`repo_url` 须为 HTTPS/SSH 合法 Git URL；拒绝空/非法参数 400 |
| internal API 的 `user_id` 来源 | taskCredentialService enrich | Task → Git Identity | read | enrich 取首个 `UserID>0` 的 `FetchRepoIdentities` | ⚠️ 须对齐 | **禁止**调用方随意传他人 `user_id` 读私有 nested；Credential 侧写死从任务身份取 user，internal 调用不传外部 user_id 或仅作断言 |
| `FetchTaskDetail` merge nested | server-container-token 持有者 | Task → Project Repos（响应 enrich） | read | 换票 scope = task_id；既有 task-detail 鉴权 | ✅ 模式已有 | merge 为响应 enrich，不写 DB；失败跳过 nested 项 |
| `BuildRepoCloneCredentials` 子仓凭证 | server-container-token 持有者 | Task → Repo OAuth | read | 既有凭证构建 + 覆盖率校验 | ⚠️ 须补 | 子仓无独立 identity 时复制父仓/任务 `UserID`+`GitIdentityID`，仅换 `RepoURL`；不得为未在任务上下文内的 URL 发证 |
| 凭证继承边界 | 同任务成员已绑 Git 身份 | Task → GitIdentity | read | `FetchRepoIdentities` | ⚠️ 须单测 | 禁止用任务 A 的身份为任务 B 的子仓 URL 换票；禁止跨 provider 混用 |
| `POST …/task-detail/` 响应扩展 | 容器 bootstrap | Task | read | server-container-token exchange | ✅ | `git_repos` / `git_repo_entries` 语义扩展；不新增写接口 |
| `POST …/repo-clone-credentials/` | 容器 bootstrap | Task → Repo | read | 同上 + 409 完整性校验 | ✅ | 子仓 URL 纳入 expected 集合；缺凭证仍 409，不因 nested enrich 放宽 |
| `cloneReposIntoSharedLayer` 并发克隆 | 容器进程（已换票） | 容器本地 FS | write（本地） | bootstrap 内；OAuth ephemeral token | ✅ | 子仓 `clone_alias`=path；sanitize 防路径穿越（沿用 alias 规则） |
| `BOOTSTRAP_CLONE_CONCURRENCY` | 运维/部署 | 容器运行时 | config | 环境变量 | ✅ | 非权限面；仅限流 |
| Swagger / machine_container §4.4 | — | — | — | 文档契约 | ⚠️ 交付项 | 注明 nested enrich 与 `clone_alias` 语义；不新增公网路由 |

## 与既有端点对齐

| 参照端点 / 能力 | 权限模式 | 本需求对齐方式 |
|----------------|----------|----------------|
| `POST …/task-detail/` | server-container-token → task scope | enrich 在既有 handler 内；不扩 scope |
| `POST …/repo-clone-credentials/` | 同上 + repo 覆盖率 | 子仓 URL 纳入 expected；继承 identity |
| `GET …/nested-git-repos/`（租户公网） | 租户成员 + 当前用户 OAuth | internal API **复用** `listNestedGitRepos` 算法；OAuth 用任务身份 user_id，非浏览器用户 |
| `git-repo-clone-alias` | server-container-token 读 entries | nested 项 `clone_alias=path`；sanitize 同既有 |
| `project-nested-git-repos` | 项目读 + 当前用户 OAuth | 发现算法 SSOT 在 ProjectService；容器链走 internal + 任务身份 |

## IDOR / 串租户 / Token 风险

| 风险 | 缓解 |
|------|------|
| internal API 被外网直接调用 | 仅内网监听；网关不暴露 `/api/internal/` |
| 传任意 `user_id` 读他人私有 nested | Credential enrich 从 `FetchRepoIdentities` 取 user；internal 调用链固定 |
| 子仓 URL 注入未授权远程 | 仅 merge `listNestedGitRepos` 返回且 `url` 非空的项；发现失败跳过 |
| 用父仓 token 克隆用户无权访问的子仓 | Git Provider 侧 401/403；容器日志记录失败；**不阻断**父仓 |
| enrich 写入 `project_repos` 造成写权限绕过 | **非目标**；禁止 INSERT/UPDATE project_repos |
| 大批量并发克隆打爆 Git / 耗尽 token | `BOOTSTRAP_CLONE_CONCURRENCY` 默认 8；日志可观测 |

## 新角色

无。不引入新 RBAC 角色或权限粒度。

## 实现检查清单（Build 阶段）

- [ ] internal handler 不对公网注册；OpenAPI 标记 internal
- [ ] enrich 中 `user_id` 来自任务 `FetchRepoIdentities`，非请求参数信任
- [ ] 子仓凭证 inherit 单测：同 UserID/GitIdentityID、不同 RepoURL
- [ ] 单测：nested API 失败 → task-detail 仍含父仓 + 父仓凭证可用
- [ ] 单测：未在任务 identity 内的 URL 不生成凭证
- [ ] machine_container §4.4 更新 nested / alias 说明

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版：internal 发现 + task-detail enrich + 凭证继承 + 容器克隆 |
