# 嵌套子仓层图提交 / Push / PR 识别

日期：2026-07-19  
状态：已批准（goal-mode 自动采用）  
作者：claude  
基于：`2026-07-18-nested-repo-clone-relocate-design.md`、`docs/dev/nested-repos-submodule-eval.md`

## 问题

任务详情层图 ztree：

1. 节点显示「N+ 个文件变化」，但「提交」按钮禁用；
2. 变动落在「相对父层（未出现在当前 Git 暂存/工作区）」；
3. 根因：子仓经 staging→移入父仓目录后，`layerGitWorkdirRootsForFileListing` 在父仓已有 `.git` 时只返回父仓根，不发现内嵌 `.git`；父仓 `.gitignore` 忽略子仓工作树 → 父层 FS diff 有路径，但 `git status` 无条目 → `git_layer_diff_only`；`git_worktree_dirty=false` → 提交门控关闭。

## 目标（验收）

1. 父仓内嵌子仓有未提交变更时，`git_worktree_dirty === true`，ztree「提交」可点。
2. 子仓内文件标注为该子仓的 staged/unstaged，不再误标 `git_layer_diff_only`（在子仓 git 可见的前提下）。
3. `POST …/git/commit`：对所有**脏**工作树提交（深路径子仓优先）；若父仓存在 `.gitmodules`，将本次提交的子仓 `path→HEAD` 写入 `.nested-repo-heads` 并在父仓可提交时一并 commit（**不**引入 mode=160000 gitlink）。
4. OAuth push / PR：仅对「脏或相对目标分支有可推送提交」的工作树推送；每个成功推送的仓可创建 PR。
5. 单测：嵌套发现、dirty 聚合、多仓 commit、最长前缀匹配。

## 非目标

- 不升级为完整 gitlink submodule 工作流（维持注册表 + 独立克隆）。
- 不改 Django 公网路由；不新增 Python API。
- 不强制架构 Plateau（容器内契约行为扩展，拓扑不变）。

## 方案

### A. 发现

扩展 `layerGitWorkdirRootsForFileListing`：在每个顶层 git 根下有界深度扫描内嵌 `.git`（跳过 `node_modules` / `.git` / `.bootstrap-staging` 等）；`relPrefix` 为相对层根的 posix 路径（如 `ram-work/docs`）。

### B. Diff 标注

`resolvePairedWorkdirsForDiff` 按 **最长 relPrefix** 匹配，使 `ram-work/docs/...` 归属 docs 子仓。

### C. Commit

新模块 `layerGitCommit.mjs`：收集脏根 → 深度降序 commit → 同步父仓 `.nested-repo-heads` → 再 commit 脏父仓。返回 `committed[]`。

### D. Push / PR

`layerGitOauthPush`：在现有多仓循环前过滤为「dirty 或 ahead」；避免对全量嵌套仓索取 token。

## 测试

| 层 | 内容 |
|----|------|
| JS | 嵌套 dirty→true；commit 多根；最长前缀；push 过滤干净仓 |
| 前端 | 现有 `layerChangesDirty` / ztree 门控保持；嵌套 dirty 解锁 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-19 | goal-mode 初版并自动采用 |
