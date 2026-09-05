# NFR 澄清 — 推荐绩效微信分账 Tab

- **日期**: 2026-08-23
- **价值流**: `docs/superpowers/plans/2026-08-23-system-admin-referral-wechat-profit-sharing-tab-value-stream.md`
- **默认等级**: L2；资金只读展示 L3 安全；可伸缩性 L0（书面）

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 等级 | 结论 / 动作 |
|------|---------|------|------|-------------|
| GET `/api/system-admin/profit-sharing/?referrer_user_id=` | 无租户 ID | 缺租户键 | **L0** | 超管按推荐人图查；JOIN 用 referrer/买家等值；分页 ≤50。升级触发：单推荐人打标订单 >1 万或该接口 QPS>20 |
| POST `/api/system-admin/profit-sharing/refresh-wechat/` | 无 | 缺键 | **L0** | ids ≤50；升级触发：微信 429 常态 |
| FE `/system-admin/users/` 抽屉 Tab | 无 | 缺键 | **L0** | 管理后台单操作者 |

无分片 ID 标 L0 理由：平台员工低频跨租户只读 + 已分页 + 写明升级触发。`referrer_user_id` 是过滤键而非租户分片键，不纠正路径。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| GET 列表 | 无 | 刷新/切 Tab | — | — | **L0** |
| POST refresh-wechat | 无本地写；微信 QueryOrder 只读 | 连点同步 | 同一批 ids | 无（不需要 Idempotency-Key） | L0 重复查询；前端 `createClickGuard` |
| 微信 CreateOrder / 回退 | 本增量**不调用** | — | — | — | — |

无 Kafka / Webhook / timer 新路径。禁止用 `tenant_id`/`user_id` 作消费键 — 无消费者。

资金路径默认 ≥ L3 适用于**写**分账；本增量明确无本地资金副作用，出站为官方查询接口。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L0 | 见上 |
| 数据一致性 | L1 | 本地台账与微信短窗口可不一致；同步只展示 |
| 安全 | L3 | staff + 禁 openid + ids 图内校验 |
| 可用性 | L2 | 单行微信失败不拖垮整页 |
| 性能 | L2 | QueryOrder 串行当前页，受 FREQUENCY_LIMITED |
| 可观测性 | L2 | info 列表 ok；warn 越权/微信错；禁 PII |
| 幂等 | L0 | 无写 |

## 质量场景

1. 刺激：超管打开微信 Tab 且图内有打标订单。响应：列表含被推荐人与订单号，无 openid。
2. 刺激：同步时微信 FREQUENCY_LIMITED。响应：该行 wechat_error，其它行继续，本地 status 不变。
3. 刺激：非 staff。响应：403。
4. 刺激：refresh ids 含图外记录。响应：400，零次微信调用。

## 领域模型影响

不新增聚合。查询服务编排既有 ReferralEdge + ResourceOrder + ProfitSharing 台账。无纠正分片键。
