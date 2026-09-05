# NFR：父层 diff git-first

| 类别 | 级别 | 说明 |
|------|------|------|
| 性能 | L2 | 有 git 时避免 O(全树) walk；status/tree-diff 为 O(变动) |
| 正确性 | L2 | 共享 .git 须扫父+子 status；HEAD 分叉须 tree-diff |
| 可观测 | L2 | 既有请求日志；truncated 语义保留 |
| 安全 | L2 | 仍限定层内 workdir，无新出站 |
