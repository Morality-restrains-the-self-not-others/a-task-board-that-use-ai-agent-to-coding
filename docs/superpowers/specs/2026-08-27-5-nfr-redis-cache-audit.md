# 审计: /5-nfr 是否合理利用 Redis 缓存

- **Date:** 2026-08-27
- **Status:** accepted (research; skill 正文未改)
- **Scope:** `.claude/skills/5-nfr/` 设计 vs 本仓库 Redis 真实职责 vs 存量 NFR 文档
- **非目标:** 不改业务缓存实现；不把 Redis 缓存升为 NFR 硬门禁

> 输入: `/5-nfr` SKILL + `references/idempotency.md` + `references/path-shard-id-scalability.md`；`docs/superpowers/plans/*nfr*`；taskAuth PDP / taskTenant membership_rev / taskEvents SSE / taskCloudService relay session。
>
> 输出使用者: `/5-nfr` 补丁（见 OPT-20260827-004）、`/6-ddd` 读模型、架构读者。

## 一句话结论

**/5-nfr 没有合理地把 Redis 当作「缓存」来设计。** 它对 Redis 的三处引用分别是幂等键持久化备选、分片键前缀、以及可用性 L3 的无名「缓存兜底」。这在「不要给每个增量硬塞缓存」这一点上是克制且正确的；但缺少角色拆分与 cache-aside / TTL / 失效 / 降级清单，导致存量 NFR 几乎只在 Redis 当 MQ/基础设施时才提到它，读路径缓存则多落成进程内短 TTL。

判定标准（本审计的「合理」）：

| # | 标准 | 结果 |
|---|------|------|
| 1 | 不为每个增量强制 Redis 缓存（避免过度设计） | 通过 |
| 2 | 资金/配额不以 Redis 为唯一 SoT | 通过（幂等附录已禁止 MemoryStore-only） |
| 3 | 区分 Redis 职责：缓存 / pub-sub / 锁 / 会话 /（禁止）领域事件总线 | **失败** |
| 4 | 读路径性能 ≥ L2 时，强制回答是否缓存、键、TTL、失效、Redis 宕机降级 | **失败** |
| 5 | 与仓内已落地模式对齐（PDP `membership_rev` + 进程缓存） | **失败**（技能未引用） |
| 6 | 禁止把 Redis Streams 当新领域事件总线（已迁 Kafka） | **失败**（技能未引用；旧 NFR 仍按 Streams 写） |

## 技能正文里 Redis 实际出现了什么

技能合计约 713 行（SKILL 495 + 两个附录）。Redis/缓存命中 **3 处**，无一处定义 cache-aside：

| 位置 | 原文角色 | 是不是缓存设计 |
|------|----------|----------------|
| `SKILL.md` 可用性 ≥ L3 → 「缓存兜底」+ `find_with_fallback()` | 降级读 | 泛称「缓存」，未点名 Redis，无 TTL/失效 |
| `idempotency.md` 键持久化：DB 唯一约束 / 幂等表 / **Redis** | 去重存储 | **不是缓存**；且资金路径仍要求 DB 唯一约束兜底 |
| `path-shard-id-scalability.md` 路径类型含 **Redis key** 前缀 | 水平扩展边界 | **不是缓存策略**；只问键能不能当分片 |

性能类别列了 P50/P95、吞吐、排队、长尾放大，**没有** 命中率、TTL、stampede、显式失效、locmem vs 集中缓存。

硬门禁只有两条（路径分片键、幂等性），CI 也只检查这两张表（`check_nfr_path_shard_table.py`、`check_nfr_idempotency_table.py`）。缓存不是硬门禁——这点应保持。

## 本仓库 Redis 的真实角色（第一方）

基础设施：`conf/infra/redis/config.yaml`（`${INFRA_HOST}`:6379）。`docs/architecture/README.md` 仍写「消息队列 (Redis Streams)、缓存、SSE pub/sub」——Streams 作领域事件总线已过时。

| 角色 | 代表实现 | 是否「缓存」 | /5-nfr 是否覆盖 |
|------|----------|--------------|-----------------|
| SSE fan-out | `taskEvents/internal/handlers/sse` → Redis pub/sub → taskSSE | 否，消息通道 | 无（NFR 偶发写成「Kafka+Redis 路径」） |
| PDP 权限集合 | taskAuth 进程内缓存 + Redis `membership_rev:<user_id>` 版本戳；不可达则降级纯 TTL | **是**（失效总线 + 本地缓存） | 无 |
| Relay 启动会话 | `taskCloudService/src/redis_relay_session.go`，`relay:startup:*` TTL | 短暂工作流状态，近似 SoT | 无 |
| 领域事件总线 | 历史 Redis Streams；现网 Kafka + `IdempotentDispatchService` | 否，且已弃用作总线 | 无禁止条款；2026-05 NFR 仍按 Streams 写 |
| Django 身份/SMS | `taskauth:identity:user:{uid}`、`billing_recharge_sms_ok:{uid}` | **是**（短 TTL + 显式 delete） | 无清单；仅该增量 NFR 自己写全 |
| 分布式锁 | 若干 NFR「未来用 Redis/etcd」 | 否 | 无 |

仓内已经验证过的缓存模式是：**进程内热数据 + Redis 版本戳失效 + Redis 宕机降级 TTL**（taskAuth `rbac_pdp.go` / taskTenant `membership_rev_redis.go`）。技能完全没把这个模式变成 NFR 问题单。

## 存量 NFR 文档在做什么

扫描 `docs/superpowers/plans/*nfr-clarification.md` 与 `*-nfr.md`（约 279 份）：

| 集合 | 份数 | 占比 |
|------|------|------|
| 全部 NFR 澄清 | 279 | 100% |
| 提到 Redis | 20 | 7% |
| 提到缓存/cache（含进程内） | 35 | 13% |
| Redis **且** 缓存 | 4 | 1.4% |

20 份 Redis NFR 的主导用途：

- **基础设施/启停/flush：** docker-redis、conf-sync、dev reset、Portainer 不阻塞 redis
- **消息总线：** Kafka↔Redis Streams 切换、consumer reconnect、transport 可切换
- **SSE pub/sub：** 延迟目标写成 Kafka+Redis
- **真·缓存设计（少数）：** 充值 SMS Identity 缓存刷新（含 Redis 瞬断 QS）；账单 GET 的「日活>1000 再加 Redis 缓存」升级触发；OIDC 未来 `jti` 黑名单
- **明确拒绝 Redis：** OAuth session 不引入集中 session；relay token 保护「不做分布式锁 / Redis」

读路径缓存在 NFR 里更常见的写法是「进程内短 TTL / 可选缓存 / 不做缓存以保证实时」，而不是 Redis cache-aside。技能没有问「多 worker 时 locmem 是否会漂」，但 SMS NFR 自己写了这条升级触发——说明问题真实存在，只是没有被技能标准化。

## 设计意图 vs 缺口

**合理之处（应保留）：**

1. NFR 硬门禁对准分片键与幂等，而不是缓存。缓存是性能/可用性手段，不是领域边界。
2. 资金路径禁止「仅 Redis / 仅 MemoryStore」作唯一幂等手段，与元规则 48/49 一致。
3. Redis key 可作为分片路径类型列出，避免全局热 key 无界增长。
4. 「不为未来增量预判」阻止了给每个 CRUD 预置 Redis。

**不合理之处（缓存维度）：**

1. **角色混淆。** Agent 看到「Redis」会联想到 Streams/MQ（2026-05 一批 NFR 的遗产），而不是 cache-aside。技能未写「领域事件走 Kafka；Redis 不作新总线」。
2. **性能类别缺缓存问题。** L2 读路径没有「是否缓存 / 键 / TTL / 谁失效 / 未命中回源 / stampede」。
3. **「缓存兜底」不可验证。** 可用性 L3 只说仓储 `find_with_fallback()`，没有 Redis 宕机是否 fail-open、是否返回陈旧数据、陈旧窗口多长。仓内 PDP 已有可抄的降级语义。
4. **locmem vs Redis 未门禁。** 单实例进程缓存在 NFR 里大量出现；多 worker 时会静默不一致（SMS NFR 已踩到）。
5. **与 DDD 衔接只有 CQRS。** 性能 ≥ L3 指向读写分离，不指向「先 Redis 读旁路、失效跟写模型」。对中等规模 SaaS，cache-aside 往往先于 CQRS。

## 建议补丁（不要做成硬门禁）

在 `/5-nfr` 增加可选附录 `references/redis-cache.md`，由性能或可用性 ≥ L2 的**读路径**或「多实例共享状态」触发。NFR 文档增加可选表「缓存策略审视」，**缺表不 fail CI**。

触发后必须回答：

1. Redis 在本增量的角色是哪一种？（缓存 / pub-sub / 锁 / 短暂会话 / 不用）
2. 若是缓存：键（须含租户等分片前缀）、TTL、显式失效信号、未命中谁回源、stampede（singleflight / 仅回源）。
3. Redis 不可达：fail-open（陈旧+TTL）还是 fail-closed？陈旧窗口？
4. 为何不是进程内 locmem？（单实例可 locmem；多 worker / 跨进程失效必须 Redis 或等价）
5. 禁止：Redis 作为资金/配额 SoT；Redis Streams 作为新领域事件总线；用 `user_id`/`tenant_id` 当唯一缓存键且无业务后缀。

推荐默认模式（对齐 PDP）：进程内缓存 + Redis 版本戳/delete 失效 + Redis 宕机降级 TTL。不要默认上 Redis 当唯一热数据。

全文草案见文末附录。落地跟踪：`OPT-20260827-004`。架构 README 过时描述：`OPT-20260827-005`。

## 权衡

- **不把缓存表做成 Hard Gate：** 多数增量是写路径或低频管理页；强制填表会制造空话。
- **不在本审计改 SKILL.md：** 研究结论先落盘；补丁走独立切片，避免把流水线技能与审计文档绑在一次提交。
- **不建议新 ADR：** 这是技能完备性，不是新基础设施选型。Kafka vs Redis 总线已有设计文档与存量 NFR。

## 自检

- [x] 技能全文检索 Redis/缓存
- [x] 对照第一方 Redis 调用（排除 gitService/gitlab-ce）
- [x] 统计存量 NFR 提及率
- [x] 判定标准预先写明并逐条打分
- [x] 建议可执行且明确「不做成硬门禁」

---

## 附录: 建议写入 `.claude/skills/5-nfr/references/redis-cache.md` 的正文

```markdown
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

## 缓存策略表（写入 NFR 文档，可选）

| 路径 | Redis 角色 | 键 | TTL | 失效 | 未命中 | Redis 宕机 | 判定 |
|------|------------|-----|-----|------|--------|------------|------|
| GET /api/... | 缓存 | tenant:{id}:... | 30s | 写后 DEL 或 rev++ | 回源 owner | fail-open 陈旧 ≤ TTL | 合适 |

键必须带合适分片前缀（见路径分片键附录）。禁止仅 `user_id`/`tenant_id` 而无业务后缀。
```
