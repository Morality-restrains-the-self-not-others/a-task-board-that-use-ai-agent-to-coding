# NFR 澄清 — 推荐分成比例固定 5%

- **日期**: 2026-08-25
- **价值流**: `docs/superpowers/plans/2026-08-25-fixed-referral-rate-5-percent-value-stream.md`
- **默认等级**: L2；资金计提/打款沿用既有 L3

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| `/system-admin/referral-management` | 无 | 缺键 | L0 | 平台超管单例页；QPS 极低。升级触发：该页 P95>500ms 或需按租户政策时再引入 tenantId |
| `GET /api/system-admin/referral/config/` | 无 | 缺键 | L0 | 全局单行配置（入账天数）；非租户账本。升级触发：出现按租户延迟策略 |
| `POST /api/system-admin/referral/config/` | 无 | 缺键 | L0 | 仅写 settle_delay 单行；同上 |
| 用户推荐页 `/profile/referral/` | 无（前端壳） | 缺键 | L0 | 只读文案；数据仍走既有 `userId` stats。升级触发：stats 单用户行数>100 万 |
| 计提 `billing_referral_commission_accrual` | `referrer_tenant_id` 在行内 | 合适（租户账本） | L3 | 本增量不改分片；仍按租户账户入账 |
| 微信分账出站 | 订单/商户 | 既有 | L3 | 打款比例改为 min(5, max_ratio)；路径不变 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| 管理页比例卡 | 无 | — | — | 只读 | L0 | 无按钮；Anti-Replay-OK：只读展示 |
| GET referral/config | 无 | — | — | 只读 | L0 | 纯查询 |
| POST referral/config（settle_delay） | 写单行 | 双击、超时重试 | 单例 `singleton_key=global` 覆盖写 | 自然幂等 PUT 语义 | L1 | 同延迟重放结果一致；比例字段忽略。入账天数按钮沿用既有保存（非本增量新增） |
| POST 比例（已删除） | 无 | — | — | 只读 | L0 | 前端不再发；服务端忽略比例体 |
| 计提 accrue | 插佣金行 | 消费事件重放 | `(referred, source_txn)` 唯一 | 合适 | L3 | ON DUPLICATE 空操作 |
| 微信打款 | 出站分账 | timer/重试 | 既有 out_no | 合适 | L3 | 沿用既有；金额按固定 5% 封顶 |

禁止用 `tenant_id`/`user_id` 作本增量新幂等键。资金路径不因常量化而降级。

## 类别等级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L0（管理配置）/ L3（计提） | 见路径表 |
| 数据一致性 | L3 | 计提与打款政策同源常量 5% |
| 安全 | L2 | 超管不能改资金比例 |
| 可观测性 | L1 | 打款仍打 wechat 比例日志；政策源改为 `fixed_5` |
| 容错 | L3 | 微信 max 不可用仍打款 0，禁止用配置 30% 伪装 |

## 质量场景

1. 刺激：超管打开比例卡且 config API 返回 12。响应：页面 5%，无保存。
2. 刺激：库 `referral_rate_percent=12`，消费 10000 点。响应：计提 500。
3. 刺激：微信 max_ratio=8%。响应：打款 5%。
4. 刺激：微信查询失败。响应：打款 0，展示政策仍 5%。

## 领域模型影响

`ReferralCommissionRate` 从可变配置值对象改为不可变常量 VO（唯一合法值 5）。`ReferralConfig` 聚合不再把比例当可变属性；仅 `settle_delay_days` 可写。
