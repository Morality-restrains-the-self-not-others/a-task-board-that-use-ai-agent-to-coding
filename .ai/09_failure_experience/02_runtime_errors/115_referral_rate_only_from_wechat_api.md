# [运行时] 推荐分成比例只能来自微信支付接口，禁止 min(上限, 5%) / conf 30%

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：115
- 维护者：Trae AI 团队

## 现象

- `/profile/referral/`「收益分成比例」曾依次显示 `5%`、`30%`，均与微信支付商户后台「产品中心-分账-分账管理比例」不一致。
- 用户明确：后台既不是 5%，也不是 30%；`min(微信/conf 上限, 本地政策 5%)` 是错误口径。

## 根因

1. 普通商户 APIv3 **没有**「查询分账管理比例」接口。官方 FAQ 写明只能登录商户平台查看：https://pay.weixin.qq.com/doc/v3/merchant/4014547102
2. 唯一查询接口是合作伙伴 `GET /v3/profitsharing/merchant-configs/{sub_mchid}`（https://pay.weixin.qq.com/doc/v3/partner/4012466864）。现网探测：
   - `GET /v3/profitsharing/merchant-configs` → 404
   - `GET /v3/profitsharing/merchant-configs/{本 mchid}` → 400 `INVALID_REQUEST`「子商户或品牌商户不能为当前请求的服务商商户」
3. 失败后曾用 conf 默认 30%（产品介绍上限）或本地政策 5% 或两者取小，把故障值显示成真实比例。

请求分账 `POST /v3/profitsharing/orders` 只有 `receivers.amount`（分），没有 ratio。

## 解决方案

1. 展示与打款 **只** 使用微信支付查询接口返回的 `max_ratio`（万分比；申请提高后可超过文档默认 30%）。
2. 禁止 conf `profit_sharing_max_ratio_percent`、禁止本地 5%、禁止 `min(上限, 5%)`。
3. 接口非 200 或无法解析 → 内部 API 503、`source=unavailable`、前端「—」、打款比例 0。
4. 直连商户现网会走 503/「—」，直到微信提供普通商户查询接口或商户以服务商身份拿到 200。

## 后续纠正（2026-08-22）

用户推荐页「当前的分账比例」改为读取超管 `billing_referral_config.referral_rate_percent`（`configuredReferralRateDisplay`）。微信 `merchant-configs` 失败不再把用户页抹成「—」。内部 `commission-rate` 与打款封顶仍禁止用 conf 30% / 本地 5% 伪装微信 `max_ratio`。

## 验证

```bash
cd taskBill && go test ./src -count=1 -run 'ParseWechatMaxRatioPercent|HandleInternalReferralCommissionRate|GetCommissionRate|QueryWechatMerchantMaxRatioLive|ResolveCommissionRate'
WECHAT_RATIO_PROBE=1 go test ./src -count=1 -timeout 60s -run TestProbeWechatProfitSharingRatioAPIs
curl -sS -H 'X-TaskBill-Internal-Secret: <secret>' http://127.0.0.1:8004/api/internal/taskbill/referral/commission-rate/
# 直连现网期望 503，body 不含 5% / 30%
# 若微信返回 200 max_ratio=1500，期望 commission_rate_display=15% source=wechat_merchant_config
```

Loki：`{job="task-bill"} |= "not using conf or local percent"`

## 关联

- `.ai/09_failure_experience/02_runtime_errors/112_referral_rate_wechat_partner_api_400.md`
- `.ai/09_failure_experience/02_runtime_errors/114_referral_rate_wechat_cap_not_payout.md`
- `docs/intents/backend/task-referral.intent.md`
