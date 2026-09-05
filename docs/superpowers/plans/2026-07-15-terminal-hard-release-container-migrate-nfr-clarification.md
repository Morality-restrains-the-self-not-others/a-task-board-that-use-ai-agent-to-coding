# NFR 澄清：任务终态硬释放 + 镜像容器迁移

- 日期：2026-07-15
- 默认等级：L2 Standard（auth 边界 L3）

| 类别 | 等级 | 场景 / 度量 | 领域影响 |
|------|------|-------------|----------|
| 正确性 / 安全 | L3 | 不可跨租户 migrate/stop；payload 边界校验 | internal API 强制 company+workspace |
| 可用性 | L2 | migrate 失败可重试；不误杀兄弟容器 | DispatchRetryable |
| 性能 | L2 | 单实例兄弟数通常 ≤5；同步 migrate 超时由 events 重试 | 短超时 + retry |
| 可观测性 | L2 | 结构化日志 event=container_migrated / terminal_hard_release | tracelog 字段齐全 |
| 幂等 | L2 | 重复终态事件不重复建机 | mark + 空 bindings short-circuit |
| 一致性 | L2 | 最终一致；发布失败不回滚任务状态 | 与 v17 一致 |

## 域模型影响笔记

- CSC 增 `terminal_released`
- SharedInstanceBinding 为过渡概念；reuse 解绑后新路径应 1 任务 1 实例
