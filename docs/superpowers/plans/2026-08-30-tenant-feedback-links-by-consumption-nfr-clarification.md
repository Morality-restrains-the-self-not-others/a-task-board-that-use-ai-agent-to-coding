# NFR Clarification: 意见与建议链接

> Value stream: `docs/superpowers/plans/2026-08-30-tenant-feedback-links-by-consumption-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 / 升级触发 |
|------|---------|--------|----------|-----------------|
| GET `/api/tenant/{tenantId}/billing/feedback-links/` | `tenantId` | 是：消耗与成员都按租户 | L1 路径已带键；表为全局配置、求值按 tenant 读用量 | 查询必须带 tenantId；禁止扫全租户 |
| FE `/tenant/:tenant/…` 侧栏 | `tenant` | 是 | L1 | 与 API 同键 |
| GET/POST/PUT/DELETE `/api/system-admin/feedback-link-groups/` | 无 | — | **L0** | 平台全局配置，年增量 ≪ 1 万行。升级触发：组数 > 1 万或超管 QPS 成瓶颈时再按运营分区 |
| GET `/api/system-admin/feedback-resource-kinds/` | 无 | — | **L0** | 目录个位数；同上 |
| FE `/system-admin/feedback-links/` | 无 | — | **L0** | 平台运营页 |
| Kafka `feedback-link-group-*` key=`group_id` | 非 tenant | 适配审计流 | **L0** | 配置变更低频；禁止用 tenant_id 作幂等/分区键（配置无租户所有权） |

## 幂等性强制审视

| 路径 | 副作用 | 级别 | 重复源 | 业务边界 | 幂等键 | 重放 | 持久化 |
|------|--------|------|--------|----------|--------|------|--------|
| 租户 GET | 无 | **L0** | — | — | — | 纯查询 | — |
| 超管 GET groups/kinds | 无 | **L0** | — | — | — | 纯查询 | — |
| POST 新建组 | 写库+事件 | **L2** | 双击、超时重试 | 同一次保存意图 | `Idempotency-Key` header | 返回首次 201 体 | `billing_feedback_idempotency` UNIQUE |
| PUT 整组保存 | 写库+事件 | **L2** | 同上 | 同一组 + 同一意图 | `Idempotency-Key` | 返回首次 200 体 | 同上 |
| DELETE 组 | 写库+事件 | **L2** | 同上 | 同一组删除意图 | `Idempotency-Key` | 首次 204；重放 204 | 同上 |
| FEEDBACK_LINK_GROUP_* 发布 | 出站 Kafka | **L2** | 发布重试 | 同一 `group_id`+事件类型+`updated_at` | payload 键，非 tenant_id | 下游审计可重复 | 无强制消费者 |
| 租户消耗投影 | 无写 | **L0** | — | — | — | 只读 grant/gitlab/transaction | — |

资金路径不在本增量（不改账本余额）。默认 L2 足够；不升 L3。

## 相关 NFR 类别

| 类别 | 等级 | 决策 |
|------|------|------|
| 可伸缩性 | L1 租户读 / L0 超管配置 | 见上表 |
| 数据一致性 | L2 配置写 | 单聚合整组替换；事件在提交后发布 |
| 安全 | L3 URL | 仅 https；阈值不对租户回传；region 门禁 |
| 容错 | L2 | Kafka 空则 skip publish 不回滚配置（与现网 `publishEvent` 一致） |
| 性能 | L1 | 租户 GET 一次；无轮询 |

## 质量场景

1. **刺激**: 租户连点侧栏。**响应**: 无写请求；仅首次进壳 GET 一次。
2. **刺激**: 超管保存超时重试同一 Idempotency-Key。**响应**: 不建第二组。
3. **刺激**: 无 region 调 GET。**响应**: 403，无链接体。

## 领域模型影响

- 聚合根 FeedbackLinkGroup；阈值/链接无独立事务边界。
- 消耗为读模型，非实体。
- 幂等表独立于 `billing_idempotency_key`（后者绑定 transaction_id）。
