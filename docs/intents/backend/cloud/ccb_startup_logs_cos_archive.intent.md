# 意图：评论启动日志 COS 归档

## 背景与目标

工作台评论「启动日志」只在 MySQL 分片。目标：每条日志在分片写入后合并进 COS `startup_logs.json`；**Put 成功后删除分片行**；list 从 COS（指针 + 对象）读取，仅对 Put 失败残留走分片；前端契约不变。

## 范围与边界

- 范围内：`insertCCBLogRow` 后 best-effort COS；Put 成功后驱逐分片；指针表；list COS 优先；`startupLogsPathRule`；事件 `CommentStartupLogArchived`
- 范围外：心跳行；克隆日志；Python API；替换 ADR-0023 分片键

## 约束与风险

- 表前缀 `cloud_`、utf8mb4、Snowflake 主键
- 查询带 workspace_id + task_id
- 禁止业务 ticker；禁止环境 Proxy；密钥不进 git/GET

## 验收标准

1. insert 后 memory/COS Get 含该 log id
2. 同 id 合并不重复
3. COS 失败不影响分片 insert，且不删分片
4. Put 成功后分片无该 id，list 只从 COS 还原
5. 非员工 403；GET 不回显密钥

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|-----------|--------|--------------|---------|
| 启动日志归档 COS | CommentStartupLogArchived | Kafka；键 ws:task:comment:log_id | persist COS 成功 | publish-only | — |
| 管理员改 COS 路径 | StepFullCOSConfigUpdated | 既有 | admin PATCH | publish-only | — |
| 冷打开还原 | — | — | list bindings | hydrate | 纯查询 |

## 实施计划

见 `docs/superpowers/plans/2026-08-27-startup-logs-cos-archive-plan.md`。
