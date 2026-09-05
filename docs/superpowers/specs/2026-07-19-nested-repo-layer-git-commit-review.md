# Review：嵌套子仓层图提交 / Push / PR

日期：2026-07-19  
对应设计：`2026-07-19-nested-repo-layer-git-commit-design.md`

## 结论

**通过（无 critical）**。实现与设计一致；单测覆盖发现 / dirty / commit / pin / oauth 过滤。

## 检查项

| 项 | 结果 |
|----|------|
| 嵌套 `.git` 发现 | ✅ `expandGitWorkdirRootsWithNested` |
| 提交门控（dirty） | ✅ 复用 `gitWorktreeDirty` 遍历全部根 |
| 最长前缀 diff 标注 | ✅ `matchGitRootByLongestPrefix` |
| 多仓 commit + 父仓 pin | ✅ `.nested-repo-heads`（无 gitlink） |
| Push 过滤 | ✅ `workdirNeedsPush`，避免全量嵌套索 token |
| 行数门禁 | ✅ `layerFs.mjs` ≤500（拆出 FlatFiles / GitRemote / NestedGit） |
| 日志 | ✅ 沿用 oauth / git push req log |
| 新公网 API | ✅ 无（符合 Go-first；容器内既有路由行为扩展） |

## Log / Intent 审计

- 无新 SaaS 业务事件（容器 git 例外已书面记录于 plan）。
- 错误路径仍返回既有 `detail`；前端 `data-traceId` 路径未改。

## 残留风险

1. 任务容器需滚动更新 onlineServiceJS 镜像后，线上任务详情页才生效。
2. `.nested-repo-heads` 首次写入会进入父仓 PR；属有意关联产物。
