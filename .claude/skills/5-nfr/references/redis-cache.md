# Redis 职责与读路径缓存（NFR 可选附录）

> 被 `/5-nfr` 在性能或可用性 ≥ L2 的读路径、或多实例共享易失状态时引用。
> **不是硬门禁。** 缺本节不阻断进入 DDD。资金/配额/云资源 SoT 仍以 DB 唯一约束为准。

## 角色拆分（先选角色，再谈缓存）

| 角色 | 本仓库现状 | NFR 默认 |
|------|------------|----------|
| 读旁路缓存 | taskAuth PDP 进程缓存 + `membership_rev:` | 允许；须写 TTL/失效/降级 |
| pub/sub fan-out | taskEvents SSE → Redis → taskSSE | 不是缓存；延迟目标写在性能/可用性 |
| 短暂会话/工作流 | relay `relay:startup:*` | 允许 TTL 状态；写明丢失语义 |
| 分布式锁 | 非默认 | 仅多实例互斥；禁止当业务 SoT |
| 领域事件总线 | **禁止新增**；走 Kafka + 幂等消费 | 历史 Streams 不得作为新增量方案 |

## 何时引入 Redis 缓存

| 条件 | 建议 |
|------|------|
| 单实例、低频、强实时 | locmem 或直读 DB；写升级触发 |
| 多 worker / 跨进程要同一视图 | Redis 或版本戳；禁止各进程 locmem 各写各的 |
| 读多写少且可接受秒级陈旧 | cache-aside + TTL + 写路径 delete/INCR rev |
| 资金/配额/支付 | 禁止 Redis 为唯一正确性来源 |

## 决策树（触发后按序回答）

```
读路径性能/可用性 ≥ L2 或跨进程共享易失状态？
├─ 否 → 不写缓存策略表，本节可留「不适用」
└─ 是 → 选 Redis 角色（角色表）：
   ├─ 缓存 → 资金/配额/支付？
   │   ├─ 是 → 禁止 Redis 为唯一 SoT；回源 DB，Redis 仅降级读
   │   └─ 否 → 缓存策略表必填：键 / TTL / 失效 / 未命中 / Redis 宕机
   ├─ pub-sub / 短暂会话 → 写明丢失语义与 TTL；不是缓存设计
   ├─ 分布式锁 → 仅多实例互斥；禁止当业务 SoT
   └─ 领域事件总线 → 禁止；走 Kafka + 幂等消费
```

## 缓存策略表（写入 NFR 文档，可选）

| 路径 | Redis 角色 | 键 | TTL | 失效 | 未命中 | Redis 宕机 | 判定 |
|------|------------|-----|-----|------|--------|------------|------|
| GET /api/... | 缓存 | tenant:{id}:... | 30s | 写后 DEL 或 rev++ | 回源 owner | fail-open 陈旧 ≤ TTL | 合适 |

键必须带合适分片前缀（见路径分片键附录）。禁止仅 `user_id`/`tenant_id` 而无业务后缀。

## 降级语义（Redis 宕机必答）

| 语义 | 行为 | 适用 |
|------|------|------|
| fail-open 陈旧 | 进程内缓存继续服务，陈旧窗口 ≤ TTL | 非资金读路径（对齐 PDP `membership_rev` 降级） |
| fail-closed | 直读 DB 或报错 | 资金/配额/支付；写路径校验 |
| 直读 DB | 旁路 Redis 回源 | 缓存命中率低或短窗口 |

仓内已验证模式：**进程内热数据 + Redis 版本戳失效 + Redis 宕机降级 TTL**（taskAuth `rbac_pdp.go` / taskTenant `membership_rev_redis.go`）。不要默认上 Redis 当唯一热数据。

## 禁止条款

- Redis 作为资金/配额/云资源唯一 SoT
- Redis Streams 作为新领域事件总线（现网已迁 Kafka + `IdempotentDispatchService`）
- 用裸 `user_id`/`tenant_id` 当唯一缓存键且无业务后缀
- 多 worker 各持进程内 locmem 而互相不知情（读模型漂移）
