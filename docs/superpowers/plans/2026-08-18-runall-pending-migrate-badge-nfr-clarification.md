# NFR 澄清 — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **价值流**: `docs/superpowers/plans/2026-08-18-runall-pending-migrate-badge-value-stream.md`
- **默认等级**: L2；本增量路径已书面 L0 处不升级

## 路径分片键强制审视

| 路径 | 分片 ID | 可伸缩性 | 动作 |
|------|---------|----------|------|
| GET `/api/dev/migrate-status` | 无 | **L0** | 单机运维台、registry 十数库；升级触发：库数 >100 或 P95>2s 再考虑并行上限/按 key 过滤 |
| 9999 页 `/` 徽章渲染 | 无 | **L0** | 纯前端 |
| POST `/api/dev/init-databases` | 无（既有） | L0 | 本增量不改写路径 |

L0 理由：运维编排器不是多租户 SaaS 路径；无 tenantId 需求。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| GET `/api/dev/migrate-status` | 无（只读） | — | — | 只读 | L0 | 重复触发同一结果（15s 缓存除外，语义仍只读） |
| POST `/api/dev/init-databases` | 有（既有 migrate 写） | 重试/重复点击 | `data_migrate_log.step_key` | 合适 | L2 | 已应用 step 跳过，仅执行未应用迁移 |

无事件消费。禁止用 tenant/user 作键（本路径无该字段）。

## 其他 NFR

| 类别 | 等级 | 场景 |
|------|------|------|
| 性能 | L2 | 全库 SELECT step_key + 列目录；并发≤4；超时 8s；禁止 2s 轮询 |
| 安全 | L2 | 不返回密码；库名白名单=registry；固定 SQL |
| 可用性 | L1 | 单库失败标 unreachable，其余继续 |
| 可观测性 | L2 | info 汇总 pending/unreachable；错误路径 warn |

## 领域模型影响

- Report 是值对象，无聚合持久化。
- 端口：`AppliedStepReader` / `LocalSQLLister`，便于测试不连 MySQL。
