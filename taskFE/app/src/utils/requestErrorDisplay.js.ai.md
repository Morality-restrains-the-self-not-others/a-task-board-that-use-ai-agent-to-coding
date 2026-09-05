# requestErrorDisplay.js

请求失败展示统一出口。`formatApiError` / `humanizeRequestErrorMessage` 须经 `humanizePaymentProviderError`，避免把微信 SDK dump 和签名头塞进 modal/toast。
