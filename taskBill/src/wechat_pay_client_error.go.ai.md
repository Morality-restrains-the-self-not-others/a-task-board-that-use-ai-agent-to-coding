# wechat_pay_client_error.go

`wechatPayClientError` 把 `*core.APIError` 和 HTTP dump 收成浏览器安全错误。`refundActionClientError` 给批准/拒绝 handler 用。完整 dump 只进 slog，不进 JSON。
