# NFR：gitservice-durable-gitlab-home

默认 L2；耐久性与运维安全升至 L3。

| 类别 | 等级 | 要求 |
|------|------|------|
| Durability / RPO | L3 | 主机重启或容器重建后 GitLab 数据不丢（持久目录在非易失文件系统） |
| Operability | L3 | 启动日志明确打印 `GITLAB_HOME` 与 fstype；tmpfs 硬失败 |
| Security | L2 | 数据目录属启动用户；PAT 0600；不新增网络面 |
| Performance | L2 | 迁移一次性；日常启动无额外显著开销 |
| Compatibility | L2 | 可用 env/conf 覆盖路径；`--clean` 不删数据 |
