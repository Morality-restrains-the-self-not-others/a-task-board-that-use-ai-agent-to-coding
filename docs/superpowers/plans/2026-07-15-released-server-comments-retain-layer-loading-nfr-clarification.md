# NFR 澄清：released-server-comments-retain-layer-loading

- **日期**: 2026-07-15
- **默认级别**: L2 Standard（goal-mode / auto-flow 默认）

| 类别 | 级别 | 场景与度量 |
|------|------|------------|
| 可用性 / UX | L2 | 冷打开已释放任务：首屏层图区 ≤1 次渲染即空态，不出现无限 connecting |
| 正确性 | L2 | 非服务态下忽略晚到 heartbeat；评论 Feed 条目数与详情 API 一致 |
| 性能 | L1 | 不新增轮询；pause 后停止心跳驱动补拉 |
| 安全 | L1 | 无新 API；不改变鉴权 |
| 可观测性 | L1 | 沿用现有 console/SSE；无需新埋点 |
| 兼容性 | L2 | 运行中双向未达成仍显示 connecting（S6） |

## 领域模型影响

无持久化模型变更。引入前端概念 `ServerNotServingUiState`（非 DDD 聚合）。
