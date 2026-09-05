# 切换镜像 × 硬件硬拦截 — NFR 澄清

- **Date:** 2026-08-21
- **Levels default:** L2；云资源启动路径一致性 L3 语义（拒绝错误规格开机）

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| PATCH `/api/projects/{projectId}/tenant_id/{tenantId}/` | tenantId（路径）+ projectId | 合适 | L2 | 租户为分片键；项目为资源键；查询/更新均带 tenant 行上的 company_id |
| 前端 `/tenant/:tenantId/project/:projectId` | tenantId | 合适 | L2 | 与 API 对齐 |
| GET `/api/internal/tenant-installed-images/lookup?tenant_id=&id=` | tenant_id query | 合适 | L2 | 已安装镜像按租户分表/过滤 |
| start-vm（既有） | tenant + workspace + task | 合适 | L2 | 不改路径 |
| Kafka | 无本增量新消息 | — | L0 | 无新事件；升级触发：若将来发 ProjectImageChanged 须带 tenant_id 作分区键 |

无「缺分片 ID 却声称无可伸缩性」的路径。可伸缩性类别：L2（配置写 QPS 低，键已正确）。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 等级 |
|------|--------|------------|--------------|--------|----------|------|
| PATCH 项目镜像/模版 | 写 `project_entries` | 双击保存、HTTP 重试 | 同一 projectId 上的目标镜像+模版快照 | 资源 PUT 语义：最终状态覆盖 | 相同 body 重放得相同终态；校验失败 400 不写 | L2（自然幂等 + 前端 busy/Idempotency-Key 元规则 52） |
| lookup 已安装镜像 | 无（只读） | — | — | — | L0：纯查询 | L0 |
| start-vm | 开云主机 | 双击启动 | comment/task 启动意图 | 既有 comment binding / start 幂等 | 不在本增量改键 | 既有 ≥ L3；本增量只让错误组合更早 400 |
| 无 Kafka 消费 | — | — | — | 禁止用 tenant_id 当消费键 | 无新消费者 | — |

禁止用 `tenant_id`/`user_id` 作本增量幂等键。前端镜像保存与「保存运行模版」须同步门闩。

## 其他类别

| 类别 | 等级 | 说明 |
|------|------|------|
| 性能 | L1 | 多次 PATCH 多一次 lookup（10s timeout）；P95 允许 +200ms |
| 可用性 | L2 | lookup 失败 → 502/400 拒绝写，不降级放行 |
| 安全 | L2 | 既有鉴权；错误文案不含内部栈 |
| 一致性 | L2 | 单行校验后更新；无跨服务事务 |
| 可观测性 | L2 | 400/502 打 trace_id + 结构化原因 |
| 容错 | L1 | lookup 不重试放大；用户可再保存 |

## 质量场景

- 刺激：跨架构只 PATCH `container_image_id`。响应：400，DB 镜像与模版均不变，body 含 message + trace_id。
- 刺激：同单提交匹配完整模版。响应：200，两者均更新。
- 刺激：cloud lookup 5xx。响应：不写库，502 或 400 可重试。

## 领域模型影响

Project 聚合不变量：非空 `container_image_id` 变更时，生效模版必须完整且实例 ISA ∈ 镜像 `target_architectures`。
