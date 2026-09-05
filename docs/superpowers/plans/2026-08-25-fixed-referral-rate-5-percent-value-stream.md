# 价值流 — 推荐分成比例固定 5%

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-fixed-referral-rate-5-percent-design.md`
- **增量**: 单切片，政策常量化

## Related Value Streams

- 既有推荐资格管理（策略 + 入账天数）保持可写。
- 本增量只改「分成比例」：展示与计提/打款口径从配置改为固定 5%。

## 当前价值流（增量）

1. 超管打开 `/system-admin/referral-management`。
2. 分成比例卡只读展示 5%；不请求配置来填充比例；无保存。
3. 被推荐人消费 → taskBill 按 5% 计提点数。
4. 微信打款金额 = `min(5%, 商户 max_ratio)`；微信比例不可用则打款 0。
5. 用户推荐页 / 申请列表 / 绩效展示的比例文案为 5%，并说明固定而非可调。

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP-FR-1 | 打开管理页比例卡 | 文案含固定 5%；无 `referral-rate-input`；无 `referral-ratio-save` |
| TP-FR-2 | GET config 返回 12 | 管理页仍展示 5%，不绑配置 |
| TP-FR-3 | 库写入 12 后计提 10000 | 佣金点数 500 |
| TP-FR-4 | POST config `{referral_rate_percent:18}` | 响应与后续读均为 5 |
| TP-FR-5 | 微信 max 8% | 打款比例 5 |
| TP-FR-6 | 微信 max 3% | 打款比例 3 |
| TP-FR-7 | 申请列表 / stats / performance | `referral_rate_display=5%`，即使库为 12 |
| TP-FR-8 | 个人推荐页说明 | 不再写「将来可能调整」 |

无新 MQ。比例写入意图删除（无副作用可发 `REFERRAL_RATIO_UPDATED`）。
