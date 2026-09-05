# 功能意图：完成超过 25 天的订单自动分账申请（兜底）

## 意图

系统对**已支付超过 25 天、仍在微信约 30 天窗口内、尚未成功分账**的资源订单，由现有小时扫描自动发起微信分账申请，作为推荐人手动分账（15 天解冻）之外的兜底。不把正常冻结改成 25 天。

## 角色

- 系统（`taskEvents` timer → `taskBill` 内部 API）：唯一执行者
- 推荐人 / 付款人 / 超管：不操作本路径

## 行为

1. 扫描入口仍是 `POST /api/internal/taskbill/profit-sharing/process-pending/`（内部密钥）。
2. 主路径改为推荐人在 15–30 天窗口手动 POST 分账；扫描**不再**对 due pending（`settle_after<=now`）自动执行。
3. 兜底集合：订单 `paid` 且 `now-30d < paid_at <= now-25d`，且分账行为 `pending`（含 `settle_after` 未到）或 `failed`，且无微信分账单号。
4. `finished` / `processing` / 已有微信单号 → 不调用 CreateOrder。
5. 无分账行但支付时推荐资格合格 → 补行并**立即**执行（settle 不得再 +8 天）。
6. 无 openid → skip，不标 failed。
7. 重放：同一 `out_profit_sharing_no`；已成功则不再打款。

## 非目标

- 不把冻结窗口从 15 天改成 25 天；25 天只做未点按钮时的兜底
- 不处理超过 30 天窗口的订单
- 不新增对外 API / 领域事件 / 进程内 ticker

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 25 天兜底发起分账 | — | 出站微信 APIv3 CreateOrder | `processPendingProfitSharings` 兜底分支 | 微信冻结资金划转 | 无新领域事件：与手动分账同一写路径，只扩查询谓词；成功仍只更新 `billing_profit_sharing.status` |

## 变更记录

- 2026-08-23：新增 25–30 天窗口兜底扫描（失败重试、漏行补标、未来 settle_after 仍执行）
- 2026-08-23：due pending 停止自动执行；15–30 天主路径改为推荐人手动分账，本兜底保留
