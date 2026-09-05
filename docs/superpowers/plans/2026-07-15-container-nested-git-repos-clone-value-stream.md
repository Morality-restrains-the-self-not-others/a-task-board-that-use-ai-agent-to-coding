# Value Stream: 容器 task-detail 下发最新子 Git 仓库并并发克隆

> 设计文档: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`  
> 权限分析: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-permission-analysis.md`  
> 日期: 2026-07-15

## Value Summary

任务关联元仓（如 `ram-work`）启动容器时，bootstrap 在 `task-detail` 即可拿到**最新发现**的嵌套子 Git 仓库列表与可用 OAuth 凭证，并在容器内**并发**克隆到正确本地目录（`clone_alias`=子路径名）；元仓下数十独立子仓无需手工加入 `project_repos`。nested 发现失败**不阻断**父仓克隆。

## Related Value Streams

- **project-nested-git-repos**（2026-07-15）：**dependency** — 发现算法 SSOT（`listNestedGitRepos` / 解析 gitignore+gitmodules）；本流在容器链复用
- **git-repo-clone-alias**（2026-07-14）：**dependency** — `git_repo_entries[].clone_alias` 决定本地目录名；nested 项 alias=path
- **task-detail-repo-clone-credentials**（2026-05-24）：**extension** — 凭证覆盖率校验仍成立；子仓 URL 纳入 expected
- **onlineServiceJS bootstrap transient retry**（2026-07-10）：**sibling** — 同属 bootstrap 阶段；并发限流与之互补

## End-to-End Flow

```text
[用户] 任务详情启动容器
  → [onlineServiceJS] POST …/server-container-token/task-detail/
    → [taskCredentialService] FetchTaskRepos + FetchRepoIdentities
      → 对每个父仓 URL（去重）GET taskProjectService /api/internal/nested-git-repos/?repo_url=&user_id=
        → [taskProjectService] listNestedGitRepos（OAuth 用任务身份 user_id）
          → merge 有效 url 进 project_repos[].git_repos / git_repo_entries（clone_alias=path；已存在 URL 跳过）
    → 返回 ContainerTaskDetail（含扩展后的仓库列表）
  → [onlineServiceJS] POST …/repo-clone-credentials/
    → [taskCredentialService] BuildRepoCloneCredentials
      → 对子仓 URL 无独立 identity：继承同任务 UserID/GitIdentityID，仅换 RepoURL
  → [onlineServiceJS] cloneReposIntoSharedLayer
    → 信号量 BOOTSTRAP_CLONE_CONCURRENCY（默认 8）限制并行
    → Promise.all 池化并发 git clone；日志「并行克隆 N 仓，并发上限 C」
    → 单仓失败记录日志；父仓仍继续
```

**失败分支（nested 不阻断）**：

```text
internal nested API 超时/404/OAuth 失败
  → enrich 跳过该父仓的 nested 项；父仓 URL 仍在 git_repos
  → task-detail 200；父仓 clone + 凭证正常
子仓 clone 失败（权限/网络）
  → 该仓 error 日志；其余仓继续；bootstrap 按既有策略处理（不新增 409 条件）
```

## 最小增量切片列表

### Slice 1: taskProjectService internal nested API

**Value:** Credential 可服务间拉取 nested 列表，无需 Django 公网

**Scope:**
- `GET /api/internal/nested-git-repos/?repo_url=&user_id=`
- 复用 `listNestedGitRepos`
- handler 单测 + OpenAPI internal 标记

**Depends on:** project-nested-git-repos 解析/handler（Slice 1–2 已交付或同迭代）

**验证:** `go test taskProjectService/src/ -run 'InternalNested|NestedGitRepos' -v`

---

### Slice 2: taskCredentialService enrich + 凭证继承

**Value:** task-detail / repo-clone-credentials 响应含子仓 URL 与可克隆凭证

**Scope:**
- 共用 enrich：`FetchTaskRepos` → 取首个 `UserID>0` 身份 → 调 internal nested → merge
- `FetchTaskDetail` 与 `BuildRepoCloneCredentials` 共用 enrich 函数
- 子仓凭证 inherit（复制 UserID/GitIdentityID，换 RepoURL）
- 发现失败跳过；不阻断父仓

**Depends on:** Slice 1

**验证:** `go test taskCredentialService/... -run 'Nested|Enrich|Inherit' -v`

---

### Slice 3: onlineServiceJS 并发加固

**Value:** 数十子仓克隆可控并行，不打爆 Git 主机

**Scope:**
- `BOOTSTRAP_CLONE_CONCURRENCY`（默认 **8**）信号量
- `cloneReposIntoSharedLayer` 池化 `Promise.all`
- 日志「并行克隆 N 仓，并发上限 C」
- `collectRepoCloneJobs` 含 nested alias

**Depends on:** Slice 2（列表含子仓 entries）

**验证:** `node --test trae-agent/onlineServiceJS/src/bootstrap.*.test.mjs`

---

### Slice 4: 契约文档 + intents

**Value:** 容器集成方可追踪 nested enrich 语义

**Scope:**
- `docs/intents/backend/container_nested_git_repos_clone.intent.md`
- `docs/intents/backend/container_nested_git_repos_clone.test-intent.md`
- `machine_container.md` §4.4 补充 nested / 并发说明

**Depends on:** Slice 2–3

**验证:** 文档存在 + CI intent 对照表

## Fields Impact

| 字段 / 资源 | 变更 |
|-------------|------|
| `project_repos` 表 | **无写入** |
| `task-detail` 响应 `git_repos[]` / `git_repo_entries[]` | enrich 追加 nested URL + `clone_alias` |
| `repo_clone_credentials` | 键扩展为含子仓 URL |
| 容器环境 `BOOTSTRAP_CLONE_CONCURRENCY` | 新增可选配置 |
| MQ 领域事件 | **无**（只读 enrich + 容器侧克隆） |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版价值流与四切片 |
