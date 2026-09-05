# 任务详情 GET 耗时 ~15s：根因与修复设计

- **日期**: 2026-08-31
- **作者**: cursor
- **状态**: ✅ approved（2026-08-31；两阶段：数据 GET 立即返回 + GitLab 探活独立 POST；不 bump 架构）
- **traceId**: `544a7a87-842d-42b9-aebd-c87b62e281c6`
- **请求**: `GET /api/tasks/todos/tenant_id/877397588196749312/workspace_id/ws_-2309487803472456748/task_880779438860562432/`
- **观测**: HTTP 200，客户端 15274ms；`task-task-service` `duration_ms=15014`
- **关联**: `docs/superpowers/specs/2026-08-23-task-detail-merge-request-30s-timeout-design.md`

## 评审修订（相对初稿）

初稿把「缩短 lite GET / 300ms 超时」塞进**同一条**任务详情请求。评审否决：页面数据必须立刻返回；**GitLab 探活标记必须是独立请求**，不得挡住详情拉取与首屏渲染。

| 初稿 | 修订后 |
|------|--------|
| 任务 GET 内仍同步打 project（即使 lite + 短超时） | 任务 GET **只读本服务库**，不等 project / GitLab |
| 探活仍可能发生在数据 GET 上（`handleGetProject` enrich） | `GET project` / `GET task` **禁止**同步探 GitLab；探活用已有 `POST /api/projects/validate-git-repos/`（页面 paint 后发） |

## 🔍 Trace 日志分析 (traceId: `544a7a87-842d-42b9-aebd-c87b62e281c6`)

- **Grafana Trace Dashboard**: [打开](http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=544a7a87-842d-42b9-aebd-c87b62e281c6&var-tempo_trace_id=544a7a87842d42b9aebdc87b62e281c6)
- **Grafana 日志搜索**: [打开](http://10.2.150.68:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"544a7a87-842d-42b9-aebd-c87b62e281c6\"","queryType":"range"}]})
- **时间范围**: 2026-08-30 21:43:59 UTC → 21:44:14 UTC（+08 05:43:59 → 05:44:14）
- **涉及服务**: task-auth、task-project-service、task-task-service、task-cloud-service、task-tenant-service、task-ai-comment、taskGitOauth

### Loki

Loki `/ready` 正常，**7 天 `{job=~".+"}` 0 条**（ingest 空）。权威时间线来自 `/tmp/ram-deploy/logs/`。Tempo 仅 task-auth forward-auth 0.15ms。

### 日志摘要

| 时刻 (+08) | 服务 | 事件 | 耗时 |
|---|---|---|---|
| 05:43:59.145 | task-auth | forward-auth | 0ms |
| 05:43:59.152 | task-project-service | workspace-permissions | 4ms |
| 05:43:59.153 | task-project-service | deliverable-systems lookup 404（未带本 trace） | 0ms |
| **~15s 空窗** | | 无完成的 `GET /api/projects/.../proj_*` | |
| 05:44:02–11 | taskGitOauth | `gitlab_unreachable`；`115.29.110.74:80` i/o timeout | 3–12s |
| 05:44:14.157–161 | cloud / tenant / ai-comment | lookup / members / comments | 0ms |
| 05:44:14.161 | task-task-service | 本 GET 200 `duration_ms=15014` | **15014ms** |

同窗口其它任务详情同样 **15013–15017ms**。05:27 同一路径曾为 **12–19ms**。评论列表 / subtree 同秒 **6–8ms**（不走 `loadProjects`）。

### 根因

`taskToJSON` → `loadProjects` → `getProjectRepoURLs` → **全量 `GET /api/projects/.../{id}`**。`handleGetProject` 在写 JSON 前同步：

1. `enrichProjectGitReposStatus` → `validateGitReposForUser(..., probeAccess=true)`（打 GitLab，`gitHTTPClient` 默认 25s）
2. `enrichProjectGitRepoDiskSizes`（GitLab statistics）

调用方 `projectHTTP.Timeout = 15s`，与 `duration_ms=15014` 吻合。超时后仍 200（`loadProjects` 忽略错误），用户只看到详情转圈 15 秒。

**项目详情页前端已经用独立 `POST /api/projects/validate-git-repos/` 拉 OAuth/探活**（`useProjectDetailGitRepos.js`），但服务端 GET project **又同步做了一遍**，于是任何依赖 GET project 的路径（含任务详情）都会被 GitLab 拖死。

## 当前架构理解

- current：`v121` application-integration / enterprise-landscape。
- 相关链路：`taskFE` → APISIX → **taskTaskService** →（错误地同步）**taskProjectService GET project** → **taskGitOauth** → 租户 GitLab `115.29.110.74`。
- 已有探活入口：`POST /api/projects/validate-git-repos/`（Go，公网，probe_access 可选）。
- 本次 **不 bump** 架构版本：无新服务；把已有 Rel_Flow 从「数据 GET 内嵌探活」改成「数据 GET 与探活 GET/POST 分离」。

## 🕸️ Code Review Graph 分析

```yaml
crg_status: skipped_unavailable
reason: "CRG unavailable: .code-review-graph/graph.db 不存在"
symbols_of_interest:
  - taskTaskService/src/task_store.go::loadProjects
  - taskProjectService/src/project_handlers.go::handleGetProject
  - taskProjectService/src/project_repo_status.go::enrichProjectGitReposStatus
  - taskFE/.../useProjectDetailGitRepos.js::fetchGitReposOAuthStatus
  - taskFE/.../taskDetailFetchFns.js::fetchTaskDetail
```

## 设计决策

### 目标

1. **页面数据请求立即返回**：`GET` 任务详情、`GET` 项目详情只读各自数据库（及已有的非 GitLab 聚合：ACL、镜像 lookup、评论）。GitLab 宕机时任务详情 P99 回到 **数十毫秒**（05:27 基线 12–19ms）。
2. **探活标记独立请求**：GitLab 可达 / `token_status` **只**由独立 HTTP 填充；该请求失败或超时 **不回滚、不挡住** 已渲染的任务/项目数据。首屏可显示「探活中…」，超时后显示失败/未知（带 `data-traceId`）。

### 两阶段交互

```
用户打开任务详情
  ├─ 阶段 A（阻塞首屏，必须快）
  │    GET /api/tasks/todos/.../{taskId}/     ← 仅 task DB；projects[] 用 stored_repo_address
  │    （并行、已有）工作区项目列表 / 评论 feeds …
  │    → 立即渲染标题、描述、关联仓库地址、评论
  └─ 阶段 B（不设 taskDetailLoading；不 await 在 A 内）
       POST /api/projects/validate-git-repos/  ← probe_access=true；urls=关联仓库
       → 更新「GitLab 探活」标记（可达 / 未授权 / 站点不可达）
```

项目详情页：**GET project 去掉同步 enrich**；阶段 B 沿用现有 `fetchGitReposOAuthStatus`（已独立）。磁盘占用若依赖 GitLab statistics，同样移出 GET，可并入阶段 B 或后续独立 GET（本增量以探活标记为准）。

### 后端

| 组件 | 改动 |
|------|------|
| `handleGetProject` | **删除**对 `enrichProjectGitReposStatus` / `enrichProjectGitRepoDiskSizes` 的同步调用。响应只含 DB 中的 `git_repos` / `git_repo_entries`。`git_repos_status` 可省略或空数组。 |
| `loadProjects`（taskTaskService） | **停止**为 mismatch 去 GET 全量 project。`projects[]` 只出任务表字段：`project_id`、`stored_repo_address`、分支。`project_repo_url` 空串；`repo_address_mismatch` 默认 false。现网 URL / mismatch 由前端用工作区项目列表（DB `git_repos`，列表接口本就不探 GitLab）计算——`buildTaskProjectsWithDetails` 已支持。 |
| `POST validate-git-repos` | **保持**为探活 SSOT。短超时失败须 200+ 每 URL 状态（含 unreachable），禁止把调用方拖到 15–25s 而不返回。可单独收紧 probe 超时（如 3s/URL），但不把探活塞回 GET。 |
| 出站 HTTP | `projectHTTP` / git client：`Transport.Proxy = nil`。任务详情路径不再使用 15s `projectHTTP` 等 GitLab。 |
| ctx / trace | 若仍有出站，禁止 `context.Background()` 丢 trace。 |

### 前端

| 页面 | 阶段 A | 阶段 B |
|------|--------|--------|
| 任务详情 | `fetchTaskDetail` **不得** `await` 探活。`taskDetailLoading` 仅覆盖任务 GET（及现有评论三路；评论已并行，不属本单 GitLab）。 | paint 后对关联 `stored_repo_address` / catalog URL 调 `POST validate-git-repos`；badge「探活中」→ 结果。失败不清任务数据；错误节点 `data-traceId`。 |
| 项目详情 | GET project 变快后，现有 `fetchGitReposOAuthStatus` 已是独立请求，确认 **不再依赖** GET 体内的 `git_repos_status` 才能显示仓库行（仓库 URL 仍来自 GET 的 DB 字段）。 | 不变：POST validate。磁盘数字若从 GET 消失，阶段 B 再拉或暂不显示。 |

### 非目标

- 不把修通 `115.29.110.74` 当作本单完成条件。
- 不修 Loki ingest。
- 不新增 Python 接口；不新增领域事件。
- 不在任务 GET 里用 300ms lite 再同步打一次 project（评审已否）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 打开任务详情（GET 数据） | — | handleGetTask | — | 纯查询 |
| 探活 GitLab（独立 POST） | — | handleValidateGitRepos | — | 纯查询；不改系统事实 |

## Domain Concept Inventory

- **Bounded Contexts**: 任务协作、项目仓库、Git OAuth。
- **Key Entities**: Task（含 stored_repo_address 快照）、Project（DB git_repos）、GitRepoProbeStatus（读模型，非新表）。
- **Candidate Aggregates**: 无新聚合。
- **Domain Events**: 无。

## 🐍 Python 新增接口

未触发。Go only。

## Value Stream Impact

- 任务详情打开、项目详情打开：首屏不再等 GitLab。
- 探活失败不影响任务/项目写入。
- 测试：GET project 在 fake GitLab hang 时仍快；GET task 不发 GET project；前端任务页 loading 结束早于探活返回。

## 🏛️ 架构变更影响

不 bump v123（与 2026-08-23 超时设计相同）。语义变化：数据 GET 与 GitLab Rel_Flow 解耦；探活只走已有 validate-git-repos。

## 权限影响分析

见 `.claude/skills/2-role-permission/permission-analysis-2026-08-31-task-detail-gitlab-probe.md`。绿灯：无新角色；GET 任务/项目与 POST validate-git-repos 沿用既有检查。

## 安全审查结论

无新写接口、无 IDOR 放宽。探活失败不得泄露 token。GET 不再返回同步 `git_repos_status`（项目页独立 POST）。

## 验收

1. 单测：`handleGetProject` 在 GitLab handler 挂起时 **<200ms** 200，且响应无等待探活。
2. 单测：`loadProjects` / GET task **不**调用 `GET /api/projects/.../{id}`。
3. 前端：`fetchTaskDetail` 返回后 `taskDetailLoading=false`，探活 POST 仍可 in-flight。
4. 探活 POST 失败：任务标题/评论仍在；badge 失败态带 `data-traceId`。
5. 回归：`buildTaskProjectsWithDetails` 仍能用工作区 catalog 标 mismatch。
6. `Transport.Proxy=nil` 于相关 client。
