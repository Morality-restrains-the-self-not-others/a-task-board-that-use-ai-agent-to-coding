# 评论启动日志 workspace 分表 — 实施计划

- **Date:** 2026-08-20

## Tasks

- [ ] T1 选片纯函数测试（空 ID 失败；格式 `logs_%02d`；同 ws 稳定；Go CRC32 与期望向量一致）
- [ ] T2 `ccbLogTable` / `ccbLogShardIndex` 实现
- [ ] T3 dataMigrate `030_ccb_logs_workspace_shards.sql` 16 表 + 回填 + utf8mb4
- [ ] T4 写入改走分片（append stage + message）；Snowflake id；解析 workspace
- [ ] T5 list 按 workspace 选片；handlers 把 workspaceID 传入 list/create
- [ ] T6 测试夹具 truncate 16 分片；既有 timeline / persist 测传入 ws
- [ ] T7 跨 workspace 隔离测 + 同 workspace 读写测
- [ ] T8 意图文档 + 无新事件例外

无新 MQ 发布任务（存储路由，非新业务意图）。
