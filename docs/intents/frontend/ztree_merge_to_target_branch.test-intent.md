# zTree「合并到目标分支」测试意图

| # | 场景 | 期望 |
|---|------|------|
| T1 | 层 `git_worktree_dirty===false` 且有 git，且当前分支 ≠ 合并目标 | 节点 `canMerge===true`，按钮可见 |
| T2 | 层 dirty===true（且当前分支 ≠ 合并目标） | `canMerge` 可见但 disabled，title 含「请先提交」 |
| T3 | 无 git（dirty===null） | 不展示合并按钮 |
| T4 | 点击合并且已配置 merge_target | 请求含 `target_branch`；成功后 refresh |
| T5 | 缺少 merge_target | alert 提示配置合并目标分支 |
| T6 | `git_remote.current_branch` 等于 `merge_target_branch_name` | `canMerge===false`，不展示合并按钮 |
