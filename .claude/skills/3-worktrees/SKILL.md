---
name: 3-worktrees
description: "Step 3 - Git Worktrees: isolate large changes in a clean branch or worktree."
---

# /3-worktrees — 工作空间隔离

Invoke the Superpowers `using-git-worktrees` skill: create an isolated workspace for the current change. Skip for small, single-file fixes.

嵌套仓须在**子仓内**创建，目录名为 `{repo}-wt/`：

```bash
cd task2app
git worktree add ../task2app-wt feat/my-feature
```

禁止在元仓根对嵌套路径 `git worktree add`。约定见 `docs/dev/nested-repo-worktree.md`。

**拆除（强制）**：功能合入 main 或等价落地后，必须在 `/10-ship` 拆除 worktree，不得留下 `{repo}-wt/`。入口：

```bash
python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>
```

细则：`.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`。

To proceed, invoke the **using-git-worktrees** skill via the Skill tool.

## 完成后 — 下一步选择

Worktree 创建完毕后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "工作空间已隔离。下一步做什么？"
multiSelect: false
options:
  1. label: "价值流映射 (推荐)"
     description: "将设计映射为端到端价值流增量"
  2. label: "重新头脑风暴"
     description: "设计尚未完成，先完成设计文档"
```

- 用户选 1 → 调用 `/4-value-stream`
- 用户选 2 → 调用 `/1-brainstorming-design-docs`
