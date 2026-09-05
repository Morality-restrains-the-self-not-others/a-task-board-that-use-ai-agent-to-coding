package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
)

// wechatPayUserError is the browser-safe wrap of a WeChat Pay SDK failure.
// Error() never includes HTTP dumps or signature headers.
type wechatPayUserError struct {
	Public string
	Status int
	Code   string
	err    error
}

func (e *wechatPayUserError) Error() string {
	if e == nil {
		return ""
	}
	return e.Public
}

func (e *wechatPayUserError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func classifyWechatAPIError(apiErr *core.APIError) (int, string) {
	if apiErr == nil {
		return http.StatusBadGateway, "微信退款失败，请稍后重试"
	}
	msg := strings.TrimSpace(apiErr.Message)
	notEnough := apiErr.Code == "NOT_ENOUGH" ||
		strings.Contains(msg, "余额不足")
	if notEnough {
		if msg == "" {
			msg = "基本账户余额不足，请充值后重新发起"
		}
		return http.StatusConflict, "微信退款失败：" + msg
	}
	if msg != "" {
		return http.StatusBadGateway, "微信退款失败：" + msg
	}
	if apiErr.Code != "" {
		return http.StatusBadGateway, "微信退款失败：" + apiErr.Code
	}
	return http.StatusBadGateway, "微信退款失败，请稍后重试"
}

func humanizeProviderErrorText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "操作失败"
	}
	lower := strings.ToLower(text)
	dump := strings.Contains(lower, "wechatpay-signature") ||
		strings.Contains(lower, "wechatpay-serial") ||
		(strings.Contains(lower, "statuscode:") && strings.Contains(lower, "code:"))
	detail := ""
	if idx := strings.Index(text, "Message:"); idx >= 0 {
		rest := strings.TrimSpace(text[idx+len("Message:"):])
		if nl := strings.IndexAny(rest, "\n]"); nl >= 0 {
			detail = strings.TrimSpace(rest[:nl])
		} else {
			detail = strings.TrimSpace(rest)
		}
	}
	notEnough := strings.Contains(text, "NOT_ENOUGH") ||
		strings.Contains(text, "余额不足")
	if notEnough {
		if detail == "" {
			detail = "基本账户余额不足，请充值后重新发起"
		}
		return "微信退款失败：" + detail
	}
	if dump {
		if detail != "" {
			return "微信退款失败：" + detail
		}
		return "微信退款失败，请稍后重试或联系财务为商户号充值"
	}
	return text
}

func wechatPayClientError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *core.APIError
	if errors.As(err, &apiErr) {
		status, public := classifyWechatAPIError(apiErr)
		return &wechatPayUserError{
			Public: public,
			Status: status,
			Code:   apiErr.Code,
			err:    err,
		}
	}
	public := humanizeProviderErrorText(err.Error())
	if public != err.Error() {
		return &wechatPayUserError{
			Public: public,
			Status: http.StatusBadGateway,
			err:    err,
		}
	}
	return err
}

// paymentActionClientError 把支付/分账动作失败映射为浏览器安全消息：剥离
// WeChat HTTP dump（含 Wechatpay-Signature 等签名头），并对非退款动作去掉
// 「微信退款失败」措辞避免误导（下单/分账失败前缀由调用方附带）。业务错误
// （非渠道 dump）原样返回，供前端展示。
func paymentActionClientError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	var userErr *wechatPayUserError
	if errors.As(err, &userErr) {
		status := userErr.Status
		if status == 0 {
			status = http.StatusBadGateway
		}
		return status, strings.Replace(userErr.Public, "微信退款失败", "微信支付失败", 1)
	}
	var apiErr *core.APIError
	if errors.As(err, &apiErr) {
		status, public := classifyWechatAPIError(apiErr)
		return status, strings.Replace(public, "微信退款失败", "微信支付失败", 1)
	}
	public := humanizeProviderErrorText(err.Error())
	if public != err.Error() {
		return http.StatusBadGateway, strings.Replace(public, "微信退款失败", "微信支付失败", 1)
	}
	return http.StatusInternalServerError, err.Error()
}

// refundActionClientError maps approve/reject failures to an HTTP status and
// a message safe to send to browsers (no WeChat signature dumps).
func refundActionClientError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	raw := err.Error()
	if strings.Contains(raw, "not pending") || strings.Contains(raw, "not found") {
		return http.StatusBadRequest, raw
	}
	var userErr *wechatPayUserError
	if errors.As(err, &userErr) {
		status := userErr.Status
		if status == 0 {
			status = http.StatusBadGateway
		}
		return status, userErr.Public
	}
	var apiErr *core.APIError
	if errors.As(err, &apiErr) {
		return classifyWechatAPIError(apiErr)
	}
	public := humanizeProviderErrorText(raw)
	if public != raw {
		return http.StatusBadGateway, public
	}
	return http.StatusInternalServerError, raw
}
