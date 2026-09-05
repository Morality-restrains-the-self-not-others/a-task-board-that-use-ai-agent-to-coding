# 设计：项目文件树提交日志旁显示当前工作分支

- 日期：2026-07-12
- 状态：approved（goal-mode 自动采用）
- 相关页面：`task-detail/?relayToTrae=true` → 项目文件树
- 架构影响：**无拓扑变更**（不新增 Application Component / 不改数据流边界）；仅扩展既有 `GET …/git/log` 响应字段。不创建 v17 架构视图。

## 问题

任务详情「项目文件树」点击目录后，右侧展示「提交日志」，但无法一眼看到该目录对应仓库的**当前工作分支**，不利于确认 bootstrap checkout / 后续编辑推送所在分支。

## 目标行为

1. 用户点击文件树中的**目录**时，仍请求 `container-layer-git-log` 并展示提交日志。
2. 若该目录本身是 Git 仓库根（目录下存在 `.git`），则在「提交日志」标题旁显示当前工作分支名（`git rev-parse --abbrev-ref HEAD`）。
3. 若目录不是仓库根（例如仓内子目录），不显示分支（仍可显示该路径的 git log）。
4. detached HEAD 时显示 `HEAD` 或 hash（以 `rev-parse --abbrev-ref` 原始输出为准）；读取失败时不阻断日志展示，仅省略分支。

## 方案选型

| 方案 | 说明 | 结论 |
|------|------|------|
| A. 扩展既有 `git/log` 响应 | 同请求返回 `is_repo_root` + `current_branch` | **采用** |
| B. 新建 `git/current-branch` | 额外 HTTP 往返 + 网关/Django 注册 | 拒绝 |
| C. 前端用任务配置 `target_branch` | 非容器内实际 HEAD | 拒绝 |

## 仓库根判定

在 `resolveLayerGitLogContext` 结果上：

- `pathspec === null` → 判定目录 = `work`；`dirHasGit(work)` 则为仓库根。
- `pathspec` 非空 → 判定目录 = `path.join(work, pathspec)`；仅当该绝对路径 `dirHasGit` 为真时视为仓库根（嵌套仓场景）。

多仓并列时点击 `repo-a`：`pathspec=null`，显示分支。点击 `repo-a/src`：非根，不显示。

## 响应契约（向后兼容）

```json
{
  "text": "...",
  "commits": [],
  "is_repo_root": true,
  "current_branch": "feature/work"
}
```

- `is_repo_root`：boolean，始终返回（解析成功时）。
- `current_branch`：仅 `is_repo_root===true` 且读 HEAD 成功时出现；否则省略或 `null`。

## 改动清单

| 层 | 文件 | 变更 |
|----|------|------|
| onlineServiceJS | `layerFs.mjs` | 导出 `clickedPathIsGitRepoRoot(ctx)` |
| onlineServiceJS | `server.mjs` | `git/log` 附带分支字段；失败打日志 |
| Gateway | `l0_registry.go` | `container-layer-git-log` 转发 `path`/`limit` |
| Vue | `TaskDetailProjectFileTree.vue` | 解析并下传 `currentBranch` / `isRepoRoot` |
| Vue | `TaskDetailExecLayerChangePreview.vue` | 标题旁展示分支 |
| 测试 | layerFs 单测 + Playwright mock | 覆盖根/非根 |
| Intent | `018_project_file_tree_current_branch.*` | 意图与测试意图 |

## 非目标

- 不实现完整 `GET …/git/branches` 列表。
- 不改 bootstrap checkout 逻辑。
- 不在文件树节点上直接标注分支（仅预览面板标题旁）。

## 验收标准

1. 点击多仓根目录：标题为「提交日志」且可见当前分支名。
2. 点击仓内子目录：有提交日志，无分支展示。
3. 旧客户端忽略新字段仍可用。
4. Gateway 转发 path 后多仓 pathspec 过滤正确。
