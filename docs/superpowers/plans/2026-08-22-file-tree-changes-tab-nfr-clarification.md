# NFR 澄清 — 项目文件树与变动列表 Tab

- **日期**: 2026-08-22
- **价值流**: `docs/superpowers/plans/2026-08-22-file-tree-changes-tab-value-stream.md`
- **默认等级**: L2（展示）；无新写路径

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| FE `/tenant/{tenantId}/workspace/{workspaceId}/task-detail/{taskId}/` | `tenantId` + `workspaceId` | 合适（租户隔离、与任务详情既有分片一致） | L2 | 本增量不改路由 |
| 既有 `GET .../container-layer-children/` | tenant/workspace/task + `layer_id` | 合适 | L2 | 不改 |
| 既有层 diff / 执行日志转发 | 同上 | 合适 | L2 | 不改 |
| Tab 本地 state | 无网络路径 | — | L0 | 无伸缩需求 |

无「缺分片键却声称无可伸缩性」的新 API。升级触发：单层变动条目超扫描上限时走既有 load-more，不在本增量。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| Tab 点击 | 无 | 连点 | — | — | **L0** Anti-Replay-OK: 只读 Tab，不生成 Idempotency-Key |
| 展示文件树 / 变动列表 | 无（沿用既有 GET） | 刷新 | — | — | 既有 stale-while-revalidate |
| 变动 Tab 内提交/暂存 | 有（既有） | — | 既有 layer+files | 既有 | **本增量不改写路径** |

无新 HTTP 写、无 Kafka、无 Webhook、无 timer。禁止用 `tenant_id`/`user_id` 作消费键 — 本增量无消费者。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 沿用任务详情路径上的租户/工作区键 |
| 数据一致性 | L0 | 展示已有快照；切 Tab 不重新取数 |
| 安全 | L2 | 无新攻击面 |
| 可用性 | L2 | `v-show` 保持状态 |
| 性能 | L2 | 两面板都挂载；层切换时本就会拉树 |
| 可观测性 | L2 | 既有错误 `data-traceId` 不变 |

## 质量场景

1. 刺激：选中层。响应：默认见文件变动 Tab 与变动列表（无载荷时空态）。
2. 刺激：点「项目文件树」。响应：见文件树；变动列表 DOM 仍在。
3. 刺激：切到另一层。响应：回到文件变动 Tab。

## 领域模型影响

无新聚合。UI 状态 `LayerFilesActiveTab` 为呈现层枚举，不进入计费/层工作区写模型。
