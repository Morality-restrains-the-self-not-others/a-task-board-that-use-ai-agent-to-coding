# 意图：gitService GitLab 数据持久化

## 目标

系统内建 GitLab（gitService）的仓库与元数据在容器重启、容器重建、工作区重建后仍可用。

## 验收

- [x] 默认数据目录不在 tmpfs
- [x] compose 使用 `$GITLAB_HOME` 挂载
- [x] legacy `./gitlab_home` 有数据时自动迁移
- [x] tmpfs 上的 `GITLAB_HOME` 默认拒绝启动
- [x] `stop --clean` 不删除持久数据

## 业务意图 → 事件对照

**无对应事件**：纯基础设施运维（数据目录持久化），不新增业务领域事件或 MQ。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 持久卷就绪 / 迁移 | — | — | — | 纯基础设施运维 |

## 关联设计

`docs/superpowers/specs/2026-07-22-gitservice-durable-gitlab-home-design.md`
