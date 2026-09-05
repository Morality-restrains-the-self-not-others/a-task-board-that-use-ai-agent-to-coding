# [运行时] 推荐页分成比例显示 30%：把微信分账上限当成佣金

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：114
- 维护者：Trae AI 团队

## 现象

- `/profile/referral/`「收益分成比例」`data-testid="referral-rate-display"` 显示 `30%`。
- 微信支付商户后台「产品中心-分账-分账管理比例」不是 30%。
- 实际请求分账金额按本地政策 5% 计算。

## 根因

1. 合作伙伴接口 `GET /v3/profitsharing/merchant-configs/{sub_mchid}` 的 `max_ratio` 是「特约商户允许父商户分账的最大比例」，单位万分比（文档 2000=20%），且仅服务商可用。直连商户调用会 400（见 112）。
2. 失败后把微信产品介绍里的默认上限 30% 写入 conf，并当作推荐页展示 SSOT。
3. 真正的分账接口是普通商户 `POST /v3/profitsharing/orders`，字段为 `receivers.amount`（分），没有 ratio。打款一直是 `min(上限, 5%)`。
4. 页面展示上限、打款用佣金，两个字段被混用。

官方文档：
- 查询最大分账比例（仅普通服务商）：https://pay.weixin.qq.com/doc/v3/partner/4012466864
- 请求分账（普通商户）：https://pay.weixin.qq.com/doc/v3/merchant/4012524936
- 产品介绍（默认最高分账比例 30% 是上限不是佣金）：https://pay.weixin.qq.com/doc/v3/merchant/4012067962

Loki：`{job="task-bill"} |= "payout amount not wechat cap"`。

## 解决方案

`min(上限, 5%)` **已被否定**（见 115）。正确口径：

1. `commission_rate_display` / 打款比例 **只** 使用微信支付查询接口的 `max_ratio`。
2. 禁止 conf 30%、禁止本地 5%、禁止两者取小。
3. 直连商户查询 400/404 → 503 + 前端「—」+ 打款 0。

## 验证

```bash
cd taskBill && go test ./src -count=1 -run 'ParseWechatMaxRatioPercent|HandleInternalReferralCommissionRate|GetCommissionRate|QueryWechatMerchantMaxRatioLive'
curl -sS -H 'X-TaskBill-Internal-Secret: <secret>' http://127.0.0.1:8004/api/internal/taskbill/referral/commission-rate/
# 直连现网期望 503；成功路径仅当微信 200 才返回真实百分比
# 禁止 body 在失败时含 5% 或 30%
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/112_referral_rate_wechat_partner_api_400.md`
- `.ai/09_failure_experience/02_runtime_errors/115_referral_rate_only_from_wechat_api.md`
