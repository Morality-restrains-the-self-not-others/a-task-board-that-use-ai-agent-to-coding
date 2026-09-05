package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// errReferrerOpenidMissing 分账接收方 openid 为空：禁止向微信提交空 account
// （微信 PARAM_ERROR：/body/receivers/0/account 字符数 0，小于最小值 1）。
var errReferrerOpenidMissing = errors.New("referrer_openid_missing")

const referrerOpenidMissingPublic = "推荐人未绑定微信收款账号，无法分账。请先让推荐人用微信扫码登录。"

// resolveProfitSharingReceiverOpenid 在发起微信分账前解析接收方 PERSONAL_OPENID。
// 优先已登记接收方（wechat_identity 支付/mp 投影）；过期 web 登录快照不得盖过它。
func resolveProfitSharingReceiverOpenid(ctx context.Context, r *profitSharingRecord) (string, error) {
	if r == nil {
		return "", errReferrerOpenidMissing
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if row, ok := loadProfitSharingReceiver(r.ReferrerUserID); ok {
		if oid := strings.TrimSpace(row.OpenID); oid != "" {
			stale := strings.TrimSpace(r.ReferrerOpenid) != "" && strings.TrimSpace(r.ReferrerOpenid) != oid
			persistProfitSharingOpenid(r.ID, oid)
			r.ReferrerOpenid = oid
			slog.InfoContext(ctx, "profit_sharing_openid_from_receiver",
				"level", "info",
				"profit_sharing_id", r.ID,
				"referrer_user_id", r.ReferrerUserID,
				"replaced_stale_snapshot", stale,
			)
			return oid, nil
		}
	}
	if oid := strings.TrimSpace(r.ReferrerOpenid); oid != "" {
		return oid, nil
	}
	ident, err := lookupWechatPayIdentity(ctx, r.ReferrerUserID, strings.TrimSpace(wechatCfg.Appid))
	if err != nil {
		slog.WarnContext(ctx, "profit_sharing_openid_lookup_failed",
			"level", "warn",
			"profit_sharing_id", r.ID,
			"referrer_user_id", r.ReferrerUserID,
			"error", err.Error(),
		)
		return "", err
	}
	if oid := strings.TrimSpace(ident.OpenID); oid != "" {
		persistProfitSharingOpenid(r.ID, oid)
		r.ReferrerOpenid = oid
		slog.InfoContext(ctx, "profit_sharing_openid_from_identity",
			"level", "info",
			"profit_sharing_id", r.ID,
			"referrer_user_id", r.ReferrerUserID,
		)
		return oid, nil
	}
	slog.WarnContext(ctx, "profit_sharing_openid_missing",
		"level", "warn",
		"profit_sharing_id", r.ID,
		"referrer_user_id", r.ReferrerUserID,
	)
	return "", errReferrerOpenidMissing
}

func persistProfitSharingOpenid(id int64, openid string) {
	openid = strings.TrimSpace(openid)
	if db == nil || id <= 0 || openid == "" {
		return
	}
	_, _ = db.Exec(`
		UPDATE billing_profit_sharing
		SET referrer_openid = ?, updated_at = ?
		WHERE id = ? AND (referrer_openid IS NULL OR referrer_openid = '' OR referrer_openid <> ?)`,
		openid, time.Now().UTC().Format(time.RFC3339), id, openid)
}

// profitSharingActionClientError 分账动作失败映射：缺 openid 为 400 业务前置条件，
// 微信 INVALID_REQUEST/RULE_LIMIT/NOT_ENOUGH 为 409（业务拒绝，不是网关挂了），
// 其余保持 502 且不把微信 PARAM_ERROR JSON dump 原样交给浏览器。
func profitSharingActionClientError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	if errors.Is(err, errReferrerOpenidMissing) {
		return http.StatusBadRequest, referrerOpenidMissingPublic
	}
	if errors.Is(err, errProfitSharingAmountZero) {
		return http.StatusConflict, profitSharingAmountZeroPublic
	}
	if code, message, _, ok := extractWechatProfitSharingReject(err); ok {
		if isEmptyReceiverAccountProviderError(err.Error()) || isEmptyReceiverAccountProviderError(message) {
			return http.StatusBadRequest, referrerOpenidMissingPublic
		}
		if isWechatProfitSharingBusinessReject(code) {
			if message == "" {
				message = "微信拒绝分账：" + code
			}
			return http.StatusConflict, message
		}
	}
	_, msg := paymentActionClientError(err)
	if isEmptyReceiverAccountProviderError(msg) {
		return http.StatusBadRequest, referrerOpenidMissingPublic
	}
	return http.StatusBadGateway, msg
}

func isEmptyReceiverAccountProviderError(text string) bool {
	s := strings.TrimSpace(text)
	if s == "" {
		return false
	}
	if strings.Contains(s, "/body/receivers/0/account") {
		return true
	}
	return strings.Contains(s, "PARAM_ERROR") && strings.Contains(s, "分账接收方帐号")
}
