# 已合入 feat 分支自动清理

- **版本**：1.2.0
- **日期**：2026-08-19
- **适用范围**：monorepo 根下所有独立 Git 仓库（含 `task2app`、`docs`、`taskEvents`、`taskCloudService` 等）及元仓额外 worktree

## 目标

功能合入 `main` 后，**立即**删除对应的本地与远端 `feat/*` 分支，并拆除为该功能创建的隔离 worktree，避免：

- Agent / 人再次从陈旧 feat tip 分叉；
- `main..feat` 误判为「还有未合入提交」；
- 远端堆积已合并幽灵分支；
- `{repo}-wt/`、`ram-work-meta2` 一类目录在功能已落地后继续占着分支，使清理脚本误判「worktree 占用、不可删」。

## 默认交付路径（与 `/10-ship` 对齐）

Agent / `/0-auto-flow` Step 10 **默认**：将变更**直接合入 `main` 并 `git push origin main`**（子仓优先，见规则 32）。

- **禁止**默认 `gh pr create` 或「只开 PR、不 merge」——这会造成开放 PR 堆积且功能已用另一路径进 main。
- 若历史上已为同主题开过 PR：合入并推送 `main` 后须 `gh pr close`，注明已直接合入。
- 用户**显式**要求开 PR 做人工审查时除外；审查通过后仍须合入 `main`、推送并走本节清理。

## 强制要求

1. **合入即删**：将 `feat/*` 合入（或 squash/merge 到）`main` 并推送远端 `main` 后，**必须**删除：
   - 本地同名分支；
   - 各 remote（通常 `origin`、`gitlab`）上的同名分支（若存在）。
2. **判定标准**（满足其一即可删分支）：
   - `git rev-list --count main..<branch>` 为 `0`（分支全部提交已在 `main` 可达）；
   - 或 `git branch --merged main` 列出该分支；
   - 或本轮交付已确认 **等价落地**（内容已在 `main`，仅 SHA 不同，例如另开 commit / cherry-pick / 重写）。此时仍须拆除 worktree 并 `--shipped-branch` 删除该 feat。
3. **禁止删除**：`main` / `master`；当前检出分支（须先 `checkout main`）；仍有**未落地**独有提交的分支；名称匹配保护列表（见脚本 `--protect`）。「SHA 不在 main」单独不足以保留——先核对该提交内容是否已在 main。
4. **Agent 义务**：goal-mode / ship / 「提交到 main 并推送」类任务在推送 `main` 成功后，**自动**执行清理脚本，**不得**仅口头建议「择机删除」。
5. **Worktree 必须先拆**：本仓库嵌套仓隔离目录约定为 `{repo}-wt/`（gitignore）。Ship **先**拆 worktree，再删分支。占用中的 worktree 会挡住 `git branch -d`。脏 worktree（未提交变更）禁止删除，须先提交、拣选到 main、或明确丢弃。
6. **执行入口**：

```bash
# 1) 拆除本轮功能的隔离工作树，并删除该 feat 本地+远端（含等价落地）
python3 runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>

# 2) 再清其它已合入 main 的 feat/*（SHA 已在 main 可达）
python3 runAll/scripts/delete_merged_feat_branches.py --apply

# 预览
python3 runAll/scripts/cleanup_stale_worktrees.py --scan
python3 runAll/scripts/delete_merged_feat_branches.py --dry-run
```

可选：`--repo task2app` 限定单仓；`--prefix feat/`（默认）；`--protect main,master`。

会话结束 / Stop hook 会 `--scan` 写入 `.runall/stale_worktrees.txt`（gitignore）。非空则下一会话必须处理，禁止继续堆新功能。

## 推荐做法

- 周维护 cron 可挂 `delete_merged_feat_branches.py --apply --fetch` 与 `cleanup_stale_worktrees.py --scan`（见 `runAll/scripts/install-ram-work-maintenance-cron.sh`）。
- 若用户要求开 PR：合入 PR 并删分支后，本地亦跑同一脚本，避免本地残留；默认路径不依赖 PR。
- Squash merge 后若 tip 仍显示未合入：以「功能是否已在 main」为准；若 tip 已无独有提交则删；若 tip 仍有未拣选提交则保留并继续合入。
- 在嵌套仓创建 worktree：`cd <repo> && git worktree add ../<repo>-wt feat/<name>`（见 `docs/dev/nested-repo-worktree.md`）。**不要**在元仓根对嵌套路径 `git worktree add`。

## 与相关规则的关系

- 不改变「禁止 `--no-verify`」等 Git 提交规范（`00_project_constraints.md` 第 6 条）。
- `/10-ship` 与 `/3-worktrees` 必须引用本条：创建隔离工作树时就要计划拆除；交付不得只合代码不拆目录；**默认直接 push `origin/main`**。
- `list_git_repos` **不得**把 `*-wt/` 或 `.git` 为 `gitdir:` 文件的 checkout 当成独立子仓扫描。
- 见 ADR-0019（Agent 交付默认合入 main）。
