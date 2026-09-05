# 嵌套仓是否升级为 gitlink submodule — 评估结论（2026-07-18）

**结论：维持现状（`.gitmodules` 注册表 + 独立克隆 / `.gitignore` 工作树），暂不升级为 mode=160000 gitlink。**

## 理由

1. runAll / 任务容器 / 发现逻辑已按「注册表 + 相对 URL」工作；改为 `git submodule update` 会牵动克隆、CI、worktree 与容器 bootstrap。
2. 产品需要「父仓可浏览、子仓可独立推送」；完整 submodule 增加同步负担，收益有限。
3. 若未来强制 `git submodule` CLI 工作流，再开迁移专项（含兼容性矩阵）。

参见：`taskProjectService` nested git 解析、`.gitmodules`、`docs/dev/nested-repo-worktree.md`。
