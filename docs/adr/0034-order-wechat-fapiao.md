# ADR-0034: 订单电子发票手动开具登记，退款冲红走微信支付普通商户 API

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** goal-mode 自动采用

---

## Context

租户支付成功后需要合规电子发票；退款后须全额红冲原蓝字发票并按剩余成交金额重开蓝票，推送到微信卡包。仓内无发票表；微信支付 SDK 无 `services/fapiao` 生成包。资金路径必须走仓内 `sdk/wechatpay-go`（ADR-0032）。

## Decision

We will：

1. 发票数据所有权在 **taskBill**（`billing_invoice_application` / `billing_invoice`），一律挂 `order_id`。
2. 用户申请、平台员工在商户平台**手动开具**后点击「已开具」登记（不调用开票 API，发票直落 `issued`，`issue_mode=manual`）；退款冲红仍用同前缀 `.../reverse`。
3. 申请须选择**专票/普票**（`invoice_type=special|general`）；专票仅限企业且须完整单位信息（税号/地址/电话/开户行/账号），缺失拒绝。
4. 手动开具的**发票文件**（PDF/JPEG/PNG ≤ 10MiB）由管理员上传到 pending 申请（`invoice_file_path`，重复上传覆盖），登记时复制到发票行；文件落 `<repoRoot>/data/taskbill_invoice_files/`，下载鉴权为平台员工或租户 `billing:view`。
5. 无生成 service 时用 `core.Client`，注释 `WechatPay-SDK-OK: no services/fapiao for <path>`。
6. 退款获批后若存在已开蓝票：全额红冲，再按剩余成交金额重开（剩余为 0 则只冲红不重开）。

## Alternatives Considered

### Alternative 1: 支付账单页 support_fapiao 官方抬头

- **Pros:** 少自建表单
- **Cons:** 无法管理员审批；与本站订单页入口要求不符
- **Why rejected:** 产品要求审批后下发

### Alternative 2: 自建 PDF + insert-cards

- **Pros:** 不依赖微信开具
- **Cons:** 须自备税控；用户明确要求调微信开票接口
- **Why rejected:** 与需求冲突

### Alternative 3: 服务商 issue-general（乐企数电）

- **Pros:** 数电能力更全
- **Cons:** 本仓是直连普通商户 Native，须 `sub_mchid`/乐企开票人
- **Why rejected:** 与现网商户模式不一致；后续可加配置分支

## Consequences

### Positive

- 发票与订单同库，租户/管理员同屏可见
- 退款与票面金额可对齐
- 手动开具文件随申请/发票行落库：租户侧可下载票面文件，管理员登记前可预览已上传文件

### Negative / Trade-offs

- 商户平台须开通电子发票并配置卡券模板，否则微信返回 `NO_AUTH` / `RULE_LIMIT`
- 审批到开具完成是异步（202 Accepted + 回调）
- 冲红同样异步：购方须在 72 小时内于微信卡包确认红字，逾期失效；本系统用读路径计算截止时间，不扫表过期

### Mitigations

- conf 可关 `fapiao.enabled`；单测注入 `issueWechatFapiaoFn` / `reverseWechatFapiaoFn`
- 回调验签走既有 `notify.Handler`
- 订单页与 `invoice_reverse_confirm` 提醒 72h 确认；真正完成仍以 `FAPIAO.REVERSED` 为准

## References

- https://pay.weixin.qq.com/doc/v3/merchant/4012538301
- https://pay.weixin.qq.com/doc/v3/merchant/4012538327
- [ADR-0032](0032-wechatpay-go-sdk.md)
