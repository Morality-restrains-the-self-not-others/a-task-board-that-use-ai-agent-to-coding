# [运行时] 推荐页分成比例显示 5%：直连商户误调服务商 merchant-configs

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：112
- 维护者：Trae AI 团队

## 现象

- `/profile/referral/`「收益分成比例」`data-testid="referral-rate-display"` 显示 `5%`。
- 内部接口 `GET /api/internal/taskbill/referral/commission-rate/` 返回 `source=local_fallback`。
- 微信支付商户平台「产品中心-分账-分账管理比例」与页面不一致。

## 根因

1. 代码调用合作伙伴接口 `GET /v3/profitsharing/merchant-configs/{sub_mchid}`（文档仅支持普通服务商）。
2. 直连商户把本 `mchid` 填进 path，微信返回 400 `INVALID_REQUEST`「子商户或品牌商户不能为当前请求的服务商商户」。
3. 失败路径曾回退本地佣金政策 5%，页面看起来像「微信就是 5%」。

Loki：`{job="task-bill"} |= "profit sharing ratio"`。

## 解决方案

1. 后续纠正见 114、115：**禁止**用 conf 30% 或本地 5% 伪装。比例只能来自微信支付查询接口。
2. 直连商户对该接口会 400，必须 503 / 前端「—」，不得回退。
3. 打款与展示同源：仅接口值；接口失败打款比例 0。

## 验证

```bash
cd taskBill && go test ./src -count=1 -run 'CommissionRate|WechatMaxRatio|QueryWechatMerchantMaxRatioLive'
curl -sS -H 'X-TaskBill-Internal-Secret: <secret>' http://127.0.0.1:8004/api/internal/taskbill/referral/commission-rate/
# 直连现网期望 503，body 不含 5% / 0.05 / 30%
# 仅当微信查询 200 时才返回 commission_rate_display 与 source=wechat_merchant_config
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/115_referral_rate_only_from_wechat_api.md`
- `docs/intents/backend/task-referral.intent.md`
