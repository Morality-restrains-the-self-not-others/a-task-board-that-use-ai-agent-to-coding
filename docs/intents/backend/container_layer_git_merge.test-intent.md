# 容器层 git/merge 测试意图

| # | 场景 | 期望 |
|---|------|------|
| T1 | 无 target_branch | 400 |
| T2 | 无 git workdir | 400 `no git` |
| T3 | dirty worktree | 400 |
| T4 | 干净，源有独有提交，目标可检出 | 200 ok，留在目标分支 |
| T5 | 冲突 | 409，merge abort |
| T6 | gateway L0 注册 `container-layer-git-merge` | 可解析并转发 `/git/merge` |
