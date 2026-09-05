# 测试意图：gitService GitLab 数据持久化

| # | 场景 | 类型 | 期望 |
|---|------|------|------|
| T1 | 未设置 env/conf | 单元 | 默认 `$HOME/.local/share/daydaymoney/gitService`（或 XDG_DATA_HOME） |
| T2 | `GITLAB_HOME` 在 tmpfs 且未允许 | 单元 | 非零退出 |
| T3 | legacy 有实质数据、目标空 | 单元 | migrate predicate 为真 |
| T4 | 容器 rm 后 restart | 集成/手工 | 仓库仍存在；挂载 Source=GITLAB_HOME |
