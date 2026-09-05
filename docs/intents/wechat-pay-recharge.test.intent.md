# 微信支付 Native 充值 — 测试意图

## 用例

1. **Go — live 未就绪不视为 mock**：`mode=live` 且 `wechatLiveOK=false` → `wechatIsMock()==false` 且 `wechatConfigured()==false`
2. **Go — 验签拒绝**：伪造 `Wechatpay-Signature` → notify 4xx，无 billing_transaction
3. **Go — 金额不一致**：回调金额与 pending 不符 → 拒绝入账
4. **Go — mock-complete 已移除**：POST `.../orders/{id}/mock-complete/` → 405
5. **前端 — 无模拟支付按钮**：PayOrderModal 不渲染「模拟支付成功」
6. **Go — 不足 1 元订单按分下单**：`total_yuan_cents=55` 的待支付订单微信预下单 `amount.total=55`，不得为 100（`TestHandlePayOrderWechatDoesNotRoundSubYuanToOneYuan`）
7. **Go — 整元订单仍为分**：`total_yuan_cents=800` → `amount.total=800`（`TestHandlePayOrderWechatIntegerYuanStillFen`）

## 命令

```bash
# Go
cd /tmp/ram-work/taskBill && go test ./src/ -count=1 -run 'Wechat'

# Django
cd /tmp/ram-work/task2app && ./activate_env.sh run -- bash -lc \
  'cd Saas_project && python -m pytest tests/test_billing_recharge_validation.py -k wechat -q'

# Playwright（mock）
cd /tmp/ram-work/task2app/playwright/front_project && \
  npx playwright test BillingRecharge.wechat-mock.playwright.test.js
```

## 环境

- `conf/billing/wechatPay/conf.yaml` → `mode: live`
- 商户序列号 / APIv3 密钥 / 私钥：`config.local.yaml` 或 `WECHAT_PAY_*`（禁止提交仓库）
- live 联调：禁止 CI 使用生产密钥 fixture

## 变更日期

2026-08-19（补：资源订单 Native 金额按分、禁止取整到元）
