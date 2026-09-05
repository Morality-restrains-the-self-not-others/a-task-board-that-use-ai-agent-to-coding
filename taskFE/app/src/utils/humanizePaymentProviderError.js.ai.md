# humanizePaymentProviderError.js

禁止把微信支付 SDK `APIError.Error()` dump（含 Wechatpay-Signature）展示给用户。识别 `error http response` / `StatusCode:` / 签名头；`NOT_ENOUGH` 抽成余额不足中文。
