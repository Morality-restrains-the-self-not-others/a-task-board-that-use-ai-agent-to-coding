# 评论启动日志 workspace 分表 — NFR 澄清

- **Date:** 2026-08-20
- **Default level:** L2（Standard）；存储隔离 L3

## 路径分片键审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| FE `/tenant/{tid}/workspace/{ws}/task-detail/{task}/` | workspaceId | ✅ 与表分片键一致 | 保持 |
| GET/POST `.../workspace/{ws}/cloud/compute/comment-container-bindings/` | workspaceId + X-Workspace-Id | ✅ | list/create 必须用该键选表 |
| GET `.../task/{task}/comment-container-bindings/`（测试直连） | 无 ws | ⚠️ 不适配 | 测试夹具补 workspace；生产走 workspace 路由 |
| Kafka/SSE `publishTaskSSE(taskID, commentID, statusData)` | 原无 workspace | ⚠️ | 补 statusData 或 CSC 解析；失败不写 |
| 表名 `cloud_comment_container_binding_logs_{NN}` | HASH(workspace_id) | ✅ 预分 16 | CRC32 稳定 |

Hard Gate：通过。可伸缩性 **L2**：16 片覆盖中期增长；单 workspace 年增量 ≪ 100 万时不做时间分区。升级触发：单分片 > 500 万行 → 加 RANGE(created_at) 或扩到 64 片。

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 键 | 重放 |
|------|--------|----------|----------|-----|------|
| GET list + logs | 无 | — | — | L0 只读 | — |
| POST create binding | 写 pending 日志 | 双击 ensure | 同一 binding 再 ensure | 不新增日志行（ensure 已存在则 skip） | 保持 |
| append stage / SSE 调度行 | 插入日志行 | SSE 重放 | 允许同文案多行（时间线） | 无唯一约束；L1 可接受重复行 | 与改前一致 |
| migrate 回填 | INSERT SELECT | 重跑 migrate | 行 id | INSERT IGNORE / 追踪表 L1 | 幂等 |

资金路径不适用。

## 质量场景

1. 刺激：两 workspace 同时写启动日志。响应：落入不同或相同分片均可，但 SELECT 按本 ws 过滤，互不可见。
2. 刺激：空 workspace 写入。响应：error/skip，不写 shard 00 垃圾。
3. 刺激：冷打开同一评论。响应：logs 文案与顺序与单表时代一致（同秒顺序靠 Snowflake/id）。
