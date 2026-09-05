# [运行时] 嵌套子仓已提交却无法推送 + 克隆层过早叠层

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-20
- 维护者：Trae AI 团队

## 现象

- 任务详情 ztree 显示「N 个文件变化」，「提交」禁用（title：暂无未提交变更）。
- 变动列表落在「相对父层（未出现在当前 Git 暂存/工作区）」；路径形如 `ram-work/task2app/...`、`ram-work/taskProjectService/...`。
- 「推送并创建PR」不出现（或 ahead 显示为空/0）。

## 根因

1. **提交禁用属预期**：子仓工作区已干净，差异仅相对父层（`git_layer_diff_only`）；`git_worktree_dirty=false` → 提交门控关闭。
2. **推送消失属缺陷**：`layerGitRemoteSnapshot` 只查 `layerPrimaryGitWorkdir`（父仓），未聚合嵌套子仓 ahead → 子仓有未推送提交时层级 `ahead` 仍为 0/`null`。
3. **叠层过早风险**：`anyLayerHasGitRepo()` 在父仓刚落盘即为 true，但 nested 仍在 staging；此时建任务会从「未移入子仓」的克隆层叠层。克隆层应在 **克隆 → 子仓移入父仓 → 工作分支切换** 后才锁定。

## 解决方案

1. `layerGitRemoteSnapshot`：对 `layerGitWorkdirRootsForFileListing` 各根求 ahead 并求和（`aggregateGitRemoteSnapshots`）。
2. `bootstrapCloneLayoutSeal`：`setBootstrapReposLayoutReady(true)` 仅在 relocate+checkout 结束后；`createJob` / `task-gate.clone_done` 在引导上下文下要求已密封。
3. 前端文案：仅父层差异时显示「N 个相对父层差异」，避免误以为可点提交。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/bootstrapCloneLayoutSeal.test.mjs \
  src/layerFsGitRemote.nestedAhead.test.mjs \
  src/layerFs.nestedGitWorkdirs.test.mjs
cd taskFE/app && npx vitest run src/utils/layerChangesDirty.test.js
```

容器侧改动须 `DOCKER_PUSH=1 ./buildDocker.sh` 并滚动任务容器后生效。

## 关联

- `docs/superpowers/specs/2026-07-19-nested-repo-layer-git-commit-design.md`
- `docs/superpowers/specs/2026-07-18-nested-repo-clone-relocate-design.md`
- `trae-agent/onlineServiceJS/src/bootstrapCloneLayoutSeal.mjs`
- `trae-agent/onlineServiceJS/src/layerFsGitRemote.mjs`
