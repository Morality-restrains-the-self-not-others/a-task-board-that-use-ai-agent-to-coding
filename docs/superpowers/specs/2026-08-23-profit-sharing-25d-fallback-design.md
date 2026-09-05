# 设计：完成超过 25 天的订单自动分账申请（兜底）

- **日期**: 2026-08-23
- **入口**: 现有 `POST /api/internal/taskbill/profit-sharing/process-pending/`（`taskEvents` timer `billing_profit_sharing_scan` / `1_process_pending` 每小时触发）
- **架构变更**: 无（不新增服务 / 聚合 / 对外 HTTP / MQ；不更新 `docs/architecture/`）

## 背景

微信支付分账冻结窗口约 30 天。主路径在支付成功时写入 `billing_profit_sharing`，`settle_after = paid_at + delay`（默认 **8 天**）。小时扫描只处理 `status=pending AND settle_after<=now`。失败会标 `failed` 且不再重试；`settle_after` 若被配到 >25 天则可能错过微信窗口。

「完成」对资源订单 = `status=paid` 且 `paid_at` 已落（无独立 `completed_at`）。

## 方案（已采纳）

**不把主延迟从 8 天改成 25 天。** 25 天是微信 30 天窗口前的最后一次补申请。

在现有 `processPendingProfitSharings` 同一调用内增加兜底：

1. 订单 `paid` 且 `paid_at <= now-25d` **且** `paid_at > now-30d`（仍在窗口内）。
2. 分账行 `pending`（即使 `settle_after` 仍在未来）或 `failed`（同一 `out_profit_sharing_no` 重试，微信幂等）。
3. 已 `finished` / `processing`，或已有 `wechat_profit_sharing_id` → 跳过。
4. 无分账行但支付时有合格推荐关系 → `markOrderForProfitSharing` 后**立即执行**（不可再 `settle_after=now+8`，否则会超出 30 天窗口）。
5. 无 openid → 与现逻辑一样 skip，不标 failed。
6. 幂等键：`order_id` / `out_profit_sharing_no`；timer 重放不得重复成功打款。

常量：`profitSharingFallbackAfterDays = 25`，`profitSharingWechatWindowDays = 30`。

新代码放独立文件（`wechat_profit_sharing.go` 已超过 500 行门禁）。

## 非目标

- 不改默认 8 天主延迟、不改推荐配置下限
- 不新增对外 API、不新增 Kafka 事件、不在 `taskBill` 内加 ticker
- 不处理 `paid_at` 已超过 30 天的订单（窗口已过；告警见 OPT）

## 领域

无新聚合。沿用 `ProfitSharingRecord`。无新领域事件：同一 timer 扫表写出站微信 CreateOrder；成功仍只更新本表 status。

## 权限

沿用内部 `X-Internal-Secret`；无新角色。见权限分析文档。
