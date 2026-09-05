# 嵌套独立仓与 Git Worktree

本 monorepo（`ram-work`）将 `task2app`、`taskAuth` 等作为**嵌套独立 Git 仓**（目录在父仓 `.gitignore`，并在 `.gitmodules` 注册 path↔url，但工作树通常不是 gitlink submodule）。

## 正确做法

1. **在目标子仓开 worktree**，不要在元仓对嵌套路径执行 `git worktree add`：
   ```bash
   cd /path/to/ram-work/task2app
   git worktree add ../task2app-wt feat/my-feature
   ```
2. 元仓 `git worktree add` 只会带出父仓跟踪文件，**不会**包含嵌套仓完整内容，易导致缺源码/缺 `conf`。
3. **SPA build（`npm run build`，纯 Vite）**：须在完整的 `taskFE/app` 工作树执行（`conf/frontend/vue/config.yaml` 在 monorepo 根 `conf/`，孤立 worktree 未挂载时构建期配置解析可能失败）。Django 已退役，无 collectstatic 步骤。
4. **完成后拆除**：功能合入 `main` 或等价落地后必须 `git worktree remove`，不得长期残留 `{repo}-wt/`。Ship 入口：

   ```bash
   python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>
   python3 runAll/scripts/delete_merged_feat_branches.py --apply
   ```

   细则：`.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`。

## 相关

- `.gitmodules`：嵌套仓注册表
- `taskFE/app/ai.md`：SPA build（Vite → dist/）约定
