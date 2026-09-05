package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"tracelog"
)

// 推荐资格只存在于 taskReferral；微信商户平台「交易中心 > 管理分账接收方」
// 仅在 POST /v3/profitsharing/receivers/add 成功后出现。
// 文档：https://pay.weixin.qq.com/doc/v3/merchant/4012528995
// PERSONAL_OPENID 的 name 为选填且须公钥加密；无 KYC 实名则省略。
// custom_relation 仅当 relation_type=CUSTOM 时填写。

const (
	psReceiverRegistered     = "registered"
	psReceiverPendingOpenid  = "pending_openid"
	psReceiverSkippedNotLive = "skipped_not_live"
	psReceiverFailed         = "failed"
	psReceiverDeleted        = "deleted"
)

// deleteWechatReceiver 调用微信「删除分账接收方」；单测可替换避免打到微信。
var deleteWechatReceiver = deleteWechatReceiverLive

func deleteWechatReceiverLive(ctx context.Context, appid, openid string) error {
	return deleteProfitSharingReceiver(ctx, appid, openid)
}

type wechatPayIdentity struct {
	AppID     string
	AppKey    string
	OpenID    string
	LegalName string
}

type profitSharingReceiverEnsureResult struct {
	Status string
	Reason string
}

var lookupWechatPayIdentity = lookupWechatPayIdentityLive
var postProfitSharingReceiver = postProfitSharingReceiverLive

// —— 分账接收方 SDK 可注入接缝 ——

var profitSharingAddReceiverCall = func(ctx context.Context, svc *profitsharing.ReceiversApiService, req profitsharing.AddReceiverRequest) (*profitsharing.AddReceiverResponse, *core.APIResult, error) {
	return svc.AddReceiver(ctx, req)
}

var profitSharingDeleteReceiverCall = func(ctx context.Context, svc *profitsharing.ReceiversApiService, req profitsharing.DeleteReceiverRequest) (*profitsharing.DeleteReceiverResponse, *core.APIResult, error) {
	return svc.DeleteReceiver(ctx, req)
}

// profitSharingReceiverAddPayload 构建添加分账接收方请求。
// PERSONAL_OPENID 的 name 选填且须公钥加密；传入则与微信实名校验，不匹配会拒绝。
func profitSharingReceiverAddPayload(ident wechatPayIdentity) profitsharing.AddReceiverRequest {
	req := profitsharing.AddReceiverRequest{
		Appid:        core.String(ident.AppID),
		Type:         profitsharing.RECEIVERTYPE_PERSONAL_OPENID.Ptr(),
		Account:      core.String(ident.OpenID),
		RelationType: profitsharing.RECEIVERRELATIONTYPE_DISTRIBUTOR.Ptr(),
	}
	if name := strings.TrimSpace(ident.LegalName); name != "" {
		req.Name = core.String(name)
	}
	return req
}

func ensureProfitSharingReceiver(ctx context.Context, userID string) (profitSharingReceiverEnsureResult, error) {
	return ensureProfitSharingReceiverWithLegalName(ctx, userID, "")
}

func ensureProfitSharingReceiverWithLegalName(ctx context.Context, userID, legalName string) (profitSharingReceiverEnsureResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return profitSharingReceiverEnsureResult{}, fmt.Errorf("user_id required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	preferredAppID := strings.TrimSpace(wechatCfg.Appid)
	ident, err := lookupWechatPayIdentity(ctx, userID, preferredAppID)
	if err != nil {
		res := profitSharingReceiverEnsureResult{Status: psReceiverFailed, Reason: truncateBytes([]byte(err.Error()), 300)}
		_ = upsertProfitSharingReceiverRow(userID, "", "", res.Status, res.Reason, false)
		return res, nil
	}
	ident.LegalName = strings.TrimSpace(legalName)
	if ident.LegalName == "" {
		if existing, ok := loadProfitSharingReceiver(userID); ok {
			ident.LegalName = strings.TrimSpace(existing.LegalName)
		}
	}
	if strings.TrimSpace(ident.OpenID) == "" || strings.TrimSpace(ident.AppID) == "" {
		res := profitSharingReceiverEnsureResult{Status: psReceiverPendingOpenid, Reason: "no_wechat_openid"}
		_ = upsertProfitSharingReceiverRow(userID, "", "", res.Status, res.Reason, false)
		persistReceiverLegalName(userID, ident.LegalName)
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "profit sharing receiver pending openid", "wechat_profit_sharing", map[string]string{
			"user_id": userID,
			"reason":  res.Reason,
		})
		return res, nil
	}
	if existing, ok := loadProfitSharingReceiver(userID); ok &&
		existing.Status == psReceiverRegistered &&
		existing.OpenID == ident.OpenID &&
		existing.AppID == ident.AppID &&
		(ident.LegalName == "" || existing.LegalName == ident.LegalName) {
		persistReceiverLegalName(userID, ident.LegalName)
		persistReferralEdgeOpenid(userID, existing.OpenID)
		return profitSharingReceiverEnsureResult{Status: psReceiverRegistered}, nil
	}
	if !wechatLiveOK {
		res := profitSharingReceiverEnsureResult{Status: psReceiverSkippedNotLive, Reason: "wechat_not_live"}
		_ = upsertProfitSharingReceiverRow(userID, ident.AppID, ident.OpenID, res.Status, res.Reason, false)
		persistReceiverLegalName(userID, ident.LegalName)
		return res, nil
	}
	if err := postProfitSharingReceiver(ctx, ident); err != nil {
		res := profitSharingReceiverEnsureResult{Status: psReceiverFailed, Reason: truncateBytes([]byte(err.Error()), 300)}
		_ = upsertProfitSharingReceiverRow(userID, ident.AppID, ident.OpenID, res.Status, res.Reason, false)
		persistReceiverLegalName(userID, ident.LegalName)
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "error", "profit sharing receiver add failed", "wechat_profit_sharing", map[string]string{
			"user_id": userID,
			"app_id":  ident.AppID,
			"err":     res.Reason,
		})
		return res, nil
	}
	res := profitSharingReceiverEnsureResult{Status: psReceiverRegistered}
	if err := upsertProfitSharingReceiverRow(userID, ident.AppID, ident.OpenID, res.Status, "", true); err != nil {
		return res, err
	}
	persistReceiverLegalName(userID, ident.LegalName)
	persistReferralEdgeOpenid(userID, ident.OpenID)
	tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "info", "profit sharing receiver registered", "wechat_profit_sharing", map[string]string{
		"user_id": userID,
		"app_id":  ident.AppID,
	})
	return res, nil
}

type profitSharingReceiverRow struct {
	UserID    string
	AppID     string
	OpenID    string
	Status    string
	LegalName string
}

func loadProfitSharingReceiver(userID string) (profitSharingReceiverRow, bool) {
	var row profitSharingReceiverRow
	err := db.QueryRow(`
		SELECT referrer_user_id, appid, openid, status, legal_name
		FROM billing_profit_sharing_receiver WHERE referrer_user_id = ?`, userID,
	).Scan(&row.UserID, &row.AppID, &row.OpenID, &row.Status, &row.LegalName)
	if err != nil {
		return profitSharingReceiverRow{}, false
	}
	return row, true
}

func persistReceiverLegalName(userID, name string) {
	name = strings.TrimSpace(name)
	if db == nil || strings.TrimSpace(userID) == "" || name == "" {
		return
	}
	_, _ = db.Exec(`UPDATE billing_profit_sharing_receiver SET legal_name = ? WHERE referrer_user_id = ?`, name, userID)
}

// persistReferralEdgeOpenid 登记分账接收方成功后回写推荐关系边的 referrer_openid。
// 覆盖过期网站应用 openid，避免新订单快照与支付 AppID 不成对。
func persistReferralEdgeOpenid(referrerUserID, openid string) {
	referrerUserID = strings.TrimSpace(referrerUserID)
	openid = strings.TrimSpace(openid)
	if db == nil || referrerUserID == "" || openid == "" {
		return
	}
	_, _ = db.Exec(`
		UPDATE billing_referral_edge
		SET referrer_openid = ?, updated_at = ?
		WHERE referrer_user_id = ? AND (referrer_openid IS NULL OR referrer_openid = '' OR referrer_openid <> ?)`,
		openid, utcNow(), referrerUserID, openid)
}

func lookupReceiverLegalName(userID string) string {
	row, ok := loadProfitSharingReceiver(strings.TrimSpace(userID))
	if !ok {
		return ""
	}
	return strings.TrimSpace(row.LegalName)
}

func upsertProfitSharingReceiverRow(userID, appid, openid, status, reason string, registered bool) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var registeredAt interface{}
	if registered {
		registeredAt = now
	}
	_, err := db.Exec(`
		INSERT INTO billing_profit_sharing_receiver
			(referrer_user_id, appid, openid, status, fail_reason, registered_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			appid = VALUES(appid),
			openid = VALUES(openid),
			status = VALUES(status),
			fail_reason = VALUES(fail_reason),
			registered_at = IF(VALUES(status) = 'registered', VALUES(registered_at), registered_at),
			updated_at = VALUES(updated_at)`,
		userID, appid, openid, status, reason, registeredAt, now, now,
	)
	return err
}

func postProfitSharingReceiverLive(ctx context.Context, ident wechatPayIdentity) error {
	if wechatClient == nil {
		return fmt.Errorf("wechat client not initialized")
	}
	svc := profitsharing.ReceiversApiService{Client: wechatClient}
	_, _, err := profitSharingAddReceiverCall(ctx, &svc, profitSharingReceiverAddPayload(ident))
	if err != nil {
		return wechatSDKResultError("add profit sharing receiver", err)
	}
	return nil
}

func lookupWechatPayIdentityLive(ctx context.Context, userID, preferredAppID string) (wechatPayIdentity, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return wechatPayIdentity{}, fmt.Errorf("user_id required")
	}
	path := "/api/internal/users/id/" + url.PathEscape(userID) + "/wechat-pay-openid/"
	if strings.TrimSpace(preferredAppID) != "" {
		path += "?preferred_app_id=" + url.QueryEscape(preferredAppID)
	}
	status, raw, err := taskAuthRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return wechatPayIdentity{}, err
	}
	if status == http.StatusNotFound {
		return wechatPayIdentity{}, nil
	}
	if status != http.StatusOK {
		return wechatPayIdentity{}, fmt.Errorf("taskAuth wechat-pay-openid status %d", status)
	}
	var parsed struct {
		AppID  string `json:"app_id"`
		AppKey string `json:"app_key"`
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return wechatPayIdentity{}, err
	}
	return wechatPayIdentity{
		AppID:  strings.TrimSpace(parsed.AppID),
		AppKey: strings.TrimSpace(parsed.AppKey),
		OpenID: strings.TrimSpace(parsed.OpenID),
	}, nil
}

func handleInternalEnsureProfitSharingReceiver(w http.ResponseWriter, r *http.Request) {
	traceID := tracelog.TraceIDFromContext(r.Context())
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", traceID)
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", traceID)
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", traceID)
		return
	}
	userID := strings.TrimSpace(stringField(body, "user_id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id required", traceID)
		return
	}
	legalName := strings.TrimSpace(stringField(body, "legal_name"))
	res, err := ensureProfitSharingReceiverWithLegalName(r.Context(), userID, legalName)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "ensure failed", traceID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  res.Status,
		"reason":  res.Reason,
		"user_id": userID,
	})
}

// deleteWechatProfitSharingReceiver 删除分账接收方（best-effort）。
// 无已登记接收方 / 未 live 时视为已处理；微信删除失败返回 failed，由调用方决定是否阻断。
func deleteWechatProfitSharingReceiver(ctx context.Context, userID string) (profitSharingReceiverEnsureResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return profitSharingReceiverEnsureResult{}, fmt.Errorf("user_id required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	existing, ok := loadProfitSharingReceiver(userID)
	if !ok || existing.Status != psReceiverRegistered || strings.TrimSpace(existing.OpenID) == "" || strings.TrimSpace(existing.AppID) == "" {
		// 无已登记接收方，无需调用微信；幂等记为 deleted。
		_ = upsertProfitSharingReceiverRow(userID, "", "", psReceiverDeleted, "", false)
		return profitSharingReceiverEnsureResult{Status: psReceiverDeleted}, nil
	}
	if !wechatLiveOK {
		res := profitSharingReceiverEnsureResult{Status: psReceiverDeleted, Reason: "wechat_not_live"}
		_ = upsertProfitSharingReceiverRow(userID, existing.AppID, existing.OpenID, res.Status, res.Reason, false)
		return res, nil
	}
	if err := deleteWechatReceiver(ctx, existing.AppID, existing.OpenID); err != nil {
		res := profitSharingReceiverEnsureResult{Status: psReceiverFailed, Reason: truncateBytes([]byte(err.Error()), 300)}
		_ = upsertProfitSharingReceiverRow(userID, existing.AppID, existing.OpenID, res.Status, res.Reason, false)
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "error", "profit sharing receiver delete failed", "wechat_profit_sharing", map[string]string{
			"user_id": userID,
			"app_id":  existing.AppID,
			"err":     res.Reason,
		})
		return res, nil
	}
	_ = upsertProfitSharingReceiverRow(userID, existing.AppID, existing.OpenID, psReceiverDeleted, "", false)
	tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "info", "profit sharing receiver deleted", "wechat_profit_sharing", map[string]string{
		"user_id": userID,
		"app_id":  existing.AppID,
	})
	return profitSharingReceiverEnsureResult{Status: psReceiverDeleted}, nil
}

func handleInternalDeleteProfitSharingReceiver(w http.ResponseWriter, r *http.Request) {
	traceID := tracelog.TraceIDFromContext(r.Context())
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", traceID)
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", traceID)
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", traceID)
		return
	}
	userID := strings.TrimSpace(stringField(body, "user_id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id required", traceID)
		return
	}
	res, err := deleteWechatProfitSharingReceiver(r.Context(), userID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "delete failed", traceID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  res.Status,
		"reason":  res.Reason,
		"user_id": userID,
	})
}
