# 测试意图：变动文件扫描上限治理

| ID | 场景 | 期望 |
| --- | --- | --- |
| T1 | unit：`collectIndex` 含大量 node_modules | 不计入 map；`truncated=false` |
| T2 | unit：非噪声文件超过 `MAX_DIFF_ENTRIES` | `truncated=true` 且 map.size=上限 |
| T3 | unit：Hints 传入 `truncatedTraceId` | 节点带 `data-traceId` |
| T4 | unit：Hints 空 traceId | 不挂 `data-traceId` |
| T5 | unit：分页 helper 回归 | `applyLayerParentDiffPagination` 行为不变 |
| T6 | unit：共享 `.git` + 大量 tracked + 子层改一文件 | `strategy=git`，`truncated=false`，含该文件 |
| T7 | unit：父脏子净（共享 `.git`） | 仍列出差异，`strategy=git` |
| T8 | unit：HEAD 分叉（worktree 共享对象） | tree-diff 出 added/removed |
| T9 | unit：非 git 目录 | `strategy=walk`，可 `truncated=true` |
| T10 | unit：独立 clone HEAD 分叉 + 大仓 | `strategy=git`（ls-tree），`truncated=false`，含 only-p/only-c |
