# 测试意图：ztree 提交按钮可见性

| ID | 场景 | 期望 |
|----|------|------|
| T1 | unit：`git_worktree_dirty=false` 且有 git | `canSubmit=true`，`submitDisabled=true` |
| T2 | unit：`git_worktree_dirty=true` | `canSubmit=true`，`submitDisabled=false` |
| T3 | unit：仅 `git_layer_diff_only` 的 layer_changes | `layerChangesPayloadImpliesWorktreeDirty===false` |
| T4 | unit：存在 `git_unstaged` | impliesDirty===true |
| T5 | unit：多仓次仓 dirty | `gitWorktreeDirty===true` |
| T6 | unit：无 @{u}，HEAD 超前 origin/master | `ahead>0`，`no_upstream=false` |
| T7 | playwright：任务详情选中可写层 | 可见 `layer-ztree-submit-btn` |
| T8 | unit：仅 `git_layer_diff_only`×3 | 文案「3 个相对父层差异」；submitTitle 含「相对父层差异」 |
| T9 | unit：含 `git_unstaged` | 文案仍为「N 个文件变化」 |
