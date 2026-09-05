# NFR 澄清：金丝雀平滑重启

- **日期:** 2026-09-03
- **价值流:** `docs/superpowers/plans/2026-09-03-runall-smooth-canary-restart-value-stream.md`
- **默认等级:** L2（Standard）；可用性对本增量 L3

## 路径分片键审视

| 路径 | 是否携带分片 ID | 可伸缩性 | 判定 |
|------|-----------------|----------|------|
| POST /api/restart-all | 否（本机编排） | L0 | 单机 runAll，无租户分片；升级触发：多机编排器 |
| POST /api/precise-restart | 否 | L0 | 同上 |
| GET /api/restart-all/progress | 否 | L0 | SSE 本机 |
| 业务服务健康 URL | 否（进程探针） | L0 | 探针不是业务分片路径 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST /api/restart-all | 有（重启进程） | 双击 / SSE 重连后再 POST | 一次 bulk 滚动 | bulk mutex `restart-all` | 进行中 409；完成后可再点（新一轮 canary） |
| POST /api/precise-restart | 有 | 双击 | 登记文件 + bulk mutex | `precise-restart` 互斥 | 同现网 |
| SIGTERM drain | 有 | 编排器重试 | 旧 PGID | 旧 PID 集合 | 已退出则 no-op |
| 业务 HTTP | 无本增量新写路径 | — | — | — | L0 无副作用 |

资金/云资源路径：本增量不触及。前端确认弹窗 + bulkQueue 同步门闩（既有）。

## 质量场景

- **可用性 L3**：滚动中途，未轮到的服务与已 overlap 的新进程保持健康探针成功（允许单服务 drain 窗口内部分连接失败）。
- **容错 L2**：overlap EADDRINUSE → drain-then-start；编译失败不杀旧进程。
- **可观测 L2**：日志 `canary_overlap` / `canary_drain` / `canary_overlap_fallback`，带 service name。

## 领域模型影响

新增 `LifecycleOperationCanaryRestart` 与 `IsCanaryRestartableStatus`；RestartAll 消费 canary 计划而非 Stop+Start 计划。
