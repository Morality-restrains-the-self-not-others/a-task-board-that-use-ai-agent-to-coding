# 评论启动日志 workspace 分表 — 价值流

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-ccb-logs-workspace-shard-design.md`

## Related Value Streams

Greenfield for physical table-name sharding. Builds on OPT-20260809-011（启动日志持久化与冷打开还原），不改变用户可见文案。

## 用户可见价值

打开任务详情评论执行区，启动日志仍完整；不同工作空间的日志物理隔离。

## Increments

| # | 增量 | 验收 |
|---|------|------|
| V1 | 选片函数 + 16 表 DDL | 同 workspace 同表；异 workspace 可异表；空 ID 报错 |
| V2 | 写入走分片（create / stage / SSE） | INSERT 命中 `..._{NN}`；遗留表无新行 |
| V3 | 读取走分片（list bindings.logs） | 冷打开时间线与改前一致；跨 ws 不串数据 |
| V4 | 存量回填 | CSC 能解析的历史行出现在正确分片 |

V1 为最小可交付；V2–V3 同会话落地；V4 随 migrate SQL。
