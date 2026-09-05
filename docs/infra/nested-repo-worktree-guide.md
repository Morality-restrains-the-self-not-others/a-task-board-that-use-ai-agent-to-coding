# 嵌套独立仓 Worktree 指引

monorepo `ram-work` 通过 `.gitmodules` 注册嵌套独立 Git 仓（如 `taskFE`、`taskAuth`）。**不要在元仓根目录**对嵌套路径执行 `git worktree add`——worktree 不会自动带出嵌套仓内容，SPA build（`npm run build`）会缺文件。

## 正确做法

1. **进入目标嵌套仓**（该目录自身有 `.git`）：
   ```bash
   cd task2app
   git worktree add ../task2app-wt feat/my-branch
   ```
2. 在 worktree 内正常开发、测试、提交；嵌套仓的 remote/branch 与元仓无关。
3. **`npm run build`（纯 Vite）** 须在完整的 `taskFE/app` 工作树执行（`conf/frontend/vue/config.yaml` 在 monorepo 根 `conf/`；孤立 worktree 未挂载 conf 时构建期配置解析可能失败），或在主 checkout 跑 build 并指向 `dist/` 产物。
4. **功能交付后拆除 worktree**（`git worktree remove` + `cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>`）。禁止留下 `{repo}-wt/`。见 `.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`。

## 禁止

- ❌ 在 `ram-work` 根执行 `git worktree add ../ram-work-wt` 后指望 `task2app/` 可用
- ❌ 仅在孤立 worktree 内跑 `npm run build`（缺少 monorepo `conf/`，构建期配置解析可能失败）

## 相关

- `.gitmodules` — 嵌套仓 path ↔ url 注册表
- `taskFE/app/ai.md` — SPA build（Vite → dist/）约定
- OPT-20260718-014
