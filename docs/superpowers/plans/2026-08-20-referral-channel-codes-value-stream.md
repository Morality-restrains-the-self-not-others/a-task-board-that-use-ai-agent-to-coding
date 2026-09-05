# 价值流 — 多渠道推荐码

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-referral-channel-codes-design.md`

## 增量切片（按用户价值）

1. **发不同渠道链接**：创建渠道码 → 复制 `/auth/register/?accessCode=`。
2. **注册归因到渠道**：accessCode 绑边快照 `channel_code` + `commission_eligible`。
3. **分渠道看人数与分账**：筛选渠道+时间段；表展示人数与佣金。

## 步骤顺序

先 schema + 绑边快照（否则统计无维度），再渠道 CRUD，再 stats 过滤，最后前端。资格门控与渠道同发，避免无资格消费误计提。

## YAML 增量

见 `conf/value-stream.yaml` 流 `referral-channel-codes`。
