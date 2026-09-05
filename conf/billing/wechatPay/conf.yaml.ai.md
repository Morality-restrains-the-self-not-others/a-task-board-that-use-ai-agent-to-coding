# wechatPay/conf.yaml

- 密钥与 `api_v3_key`、商户私钥**禁止**写入应用日志、本文件或 Playwright fixture。
- `pub_key.pem` + `pub_key_id` 用于应答/回调验签（微信支付公钥模式）。
- 生产 `mode: live`。仓库内没有 `config.local.yaml`（gitignore）。从 `config.local.yaml.example` 复制后填写序列号 / APIv3 密钥；`apiclient_key.pem` 放同目录。也可设 `WECHAT_PAY_MERCHANT_SERIAL_NO` / `WECHAT_PAY_API_V3_KEY` / `WECHAT_PAY_PRIVATE_KEY_PATH`。
- live 初始化失败**不得**回退 mock；支付接口返回未配置。已移除 `mock-complete` HTTP 接口。
- `notify_url` 使用 `${scheme}://${subdomains.www}/api/billing/wechat/notify/`。
- 电子发票：`fapiao.enabled` 默认 `false`。开通商户平台电子发票与卡券模板后再改为 `true`。`tax_code` / `tax_rate`（万分之一）须与实际经营范围一致。回调 `fapiao.notify_url` → `/api/billing/wechat/fapiao/notify/`。
- `profit_sharing_max_ratio_percent` **已删除**（OPT-20260822-017，死配置）。推荐页与打款比例只从微信支付 `GET /v3/profitsharing/merchant-configs/{sub_mchid}` 读取 `max_ratio`。直连商户对该接口会 400；失败则内部 API 503、前端「—」、打款 0。禁止再添加 conf 30%、禁止本地 5%、禁止 `min(上限, 5%)`。
- 支付 `appid` 必须是商户号已绑定的服务号/公众号、小程序、移动应用或企业微信之一，且须在商户平台完成绑定；**不可**使用微信登录开放平台 `type=qr` 网站应用作 Native `appid`（`conf/auth/task-auth` 的登录网站应用直接用于支付会稳定触发 `APPID_MCHID_NOT_MATCH`）。
