# 评论级执行细节 — NFR

| 类别 | 级别 | 说明 |
|------|------|------|
| 正确性 | L2 | active 归属启发式须可测；layer 面板全局唯一实例 |
| 可用性 | L2 | 执行细节默认折叠；active 评论可 auto-expand |
| 性能 | L2 | 非 active 评论不挂载 LayerGraphZtree；避免 N 份 zTree |
| 可访问性 | L2 | 折叠按钮 `aria-expanded`；连接状态 `role="status"` 保留 |
| 可维护性 | L2 | ContainerConnectionStatus 单文件 ≤500 行；ServerStartStatusPanel 瘦身 |
| 安全 | L1 | 无新 API；沿用任务详情读权限 |
| 可观测 | L1 | 可选 debug log：`activeExecutionCommentId` 变更 |

**默认整体级别：L2**（前端 UX 重组；无 SLA/计费/数据一致性变更）

结构热点：`useCommentExecutionContext` 与 Feed 重渲染频率（随 heartbeat  tick 需 memo 容器连接子树）。
