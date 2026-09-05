# wechat_pay.go Companion

微信支付配置：`conf/billing/wechatPay/conf.yaml` 仅非机密；商户序列号 / APIv3 密钥 / PEM 在 `conf-local/billing/wechatPay/`。`overlayWechatLocalConfig` 与 `wechatResolveDataFile` 必须先看 conf-local。禁止再读 `config.local.yaml`。
