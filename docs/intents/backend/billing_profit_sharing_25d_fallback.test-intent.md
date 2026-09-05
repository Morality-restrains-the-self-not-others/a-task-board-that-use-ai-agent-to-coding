# 测试意图：25 天分账兜底

## 测试目标

满 25 天未成功分账会在微信窗口内补申请；未满不提前；已成功不重打；失败用同一商户单号；超 30 天不再发起；主路径 8 天 due 扫描仍有效。

## 测试分层

- taskBill Go 单测（注入 `createProfitSharingOrder`，`wechatLiveOK=true`）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `paid_at` 26 天前、`pending`、`settle_after` 仍在未来 | 调用 CreateOrder |
| T2 | `paid_at` 24 天前、`settle_after` 在未来 | 不调用 |
| T3 | `failed`、完成 26 天、无微信单号 | 重试且 `out_profit_sharing_no` 不变 |
| T4 | `finished` + 微信单号 | 不调用 |
| T5 | process-pending 重放 | CreateOrder 只成功一次 |
| T6 | `paid_at` > 30 天 | 不调用 |
| T7 | 无分账行、有合格推荐边、完成 26 天 | 补行并立即执行 |
| T8 | `settle_after` 已到期、`paid_at` 仅 10 天（主路径） | 仍执行（8 天 due 未破坏） |

## 通过标准

T1–T8 全绿。无新领域事件。
