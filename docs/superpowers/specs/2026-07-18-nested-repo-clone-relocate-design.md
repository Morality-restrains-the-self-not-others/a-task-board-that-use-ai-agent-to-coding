# 子仓克隆完成后移入父仓对应路径

日期：2026-07-18  
状态：已批准（goal-mode 自动采用）  
作者：claude  
基于：`2026-07-15-container-nested-git-repos-clone-design.md`、`2026-07-14-git-repo-clone-alias-design.md`

## 问题

容器 bootstrap 对 nested 子仓与父仓一样，直接 `git clone <url> <layer>/<sanitize(alias)>`。  
`sanitizeCloneDirName` 会把 `/` 换成 `-`，且子仓落在**层根并列目录**，不会出现在父仓工作树内的对应 `path`（如 `ram-work/task2app/`）。  
父仓检出后常见空占位目录，直接 clone 进该 path 也会因目录已存在而失败。

Django reclone 转发已支持 `parent_repo_url` + 允许 `/` 的 `clone_alias`，但容器 bootstrap / reclone 未落实「先克隆、再移位」。

## 目标

1. nested 子仓：**先**克隆到层内 staging，**成功后**再 `rename/move` 到 `{父仓本地目录}/{相对 path}`。
2. `git_repo_entries` 对 nested 项增加只读字段 `parent_repo_url`（enrich 时写入）。
3. `clone_alias` 作为相对路径：按段 sanitize，保留 `/`，拒绝 `..` 与绝对路径。
4. `POST /api/repos/reclone` 同样按 `parent_repo_url` + path alias 落盘（可 staging→move）。
5. 父仓克隆失败时：子仓保留在 staging（或层根 fallback），日志标明未 relocate，不拖垮其它成功仓的 relocate。

## 非目标

- 不改 nested 发现算法、不改 Django 公网路由
- 不持久化子仓到 `project_repos` 表
- 不强制架构 Plateau（契约字段扩展，拓扑不变）

## 方案（采用）

### A. Credential enrich

`RepoCloneEntry` 增加：

| 字段 | 说明 |
|------|------|
| `parent_repo_url` | nested 时为发现所用父仓 URL；顶层仓省略或空 |

`MergeNestedReposIntoSnapshots` 追加子仓时设置 `ParentRepoURL = parent`。

### B. onlineServiceJS 落盘

1. `collectRepoCloneJobs` 读取 `parent_repo_url` / `parentRepoUrl`。
2. 规划目录：
   - 无 parent：`finalDir = layer/<resolveRepoCloneDirName>`（与现网一致）
   - 有 parent：`cloneDir = layer/.bootstrap-staging/<unique>/`；`finalRel = <parentDirName>/<pathSegments>`
3. 并行 `git clone` 到 `cloneDir`（父仓直接 final，子仓 staging）。
4. 全部 clone 轮次结束后，对成功的 nested job：清空/替换目标占位 → `fs.rename`（失败则 copy+rm）到 `finalDir`，并更新 job.repoDir 供后续 work-branch checkout。
5. `resolveRepoCloneRelPath`：段级 sanitize；reclone 复用同一解析。

### C. 契约

更新 `machine_container.md` §4.4：说明 staging→move 与 `parent_repo_url`。

## 测试

| 层 | 内容 |
|----|------|
| Go | enrich 写入 `parent_repo_url` |
| JS | path 段 sanitize；relocate 后目录在父仓下；父仓失败不 relocate |
| 文档 | intent + machine_container |

## 补充（2026-07-18）

单仓/多仓克隆失败**不得** `throw` 阻断 `feature-params` / `BOOTSTRAP_COMPLETE`；否则前端易因引导失败或进程退出表现为「容器业务端点尚未就绪」。失败写入 clone log + 进度摘要，失败仓可 reclone。

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-18 | goal-mode 初版并自动采用 |
| 2026-07-18 | 补充：克隆失败非致命，引导继续 |
