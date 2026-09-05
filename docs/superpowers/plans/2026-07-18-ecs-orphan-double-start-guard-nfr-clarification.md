# NFR：ECS 孤儿防护

| 类别 | 级别 | 说明 |
|------|------|------|
| 正确性 | L3 | 禁止覆盖绑定导致云侧残留计费实例 |
| 性能 | L2 | 对账与 Describe 复用 30s TTL；仅 workspace CSC 任务 |
| 可观测性 | L2 | start 前 supersede / orphan recycle 打 structured log |
| 安全 | L2 | 仅用任务授权 AK；internal API 不变 |
| 可用性 | L2 | 旧实例删除失败则重试 start，避免双活后继续扩大孤儿 |
