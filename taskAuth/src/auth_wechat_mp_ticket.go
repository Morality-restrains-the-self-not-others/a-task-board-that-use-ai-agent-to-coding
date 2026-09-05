package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"taskAuth/domain"
	"tracelog"
)

const (
	wechatMPFollowQRExpireSeconds = 86400
	wechatMPConflictUnionOther    = "unionid_bound_other"
	wechatMPConflictOpenIDOther   = "mp_openid_bound_other"
	wechatMPConflictMessage       = "该微信已绑定其他账号。请用已绑定该微信的账号登录后再扫码。"
)

var wechatHTTPPostJSON = wechatHTTPPostJSONLive

type mpFollowTicket struct {
	ID                  string
	UserID              string
	Status              string
	ExpireAt            time.Time
	ConflictOwnerUserID string
	ConflictCode        string
}

func handleWeChatMPFollowQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "Idempotency-Key required")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	// 票据表 user_id 为 BIGINT；bootstrap-admin 等确定性非数字 ID 写入前拒绝，
	// 避免 MySQL 1366（OPT-20260826-007）。
	if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid user id")
		return
	}
	app := wechatMPApp()
	if app == nil || strings.TrimSpace(app.AppSecret) == "" {
		writeErrorDetail(w, r, http.StatusServiceUnavailable, "wechat mp not configured")
		return
	}
	ticket, err := issueOrReuseMPFollowTicket(r.Context(), app, userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "wechat_mp_follow_qr_failed",
			"level", "error",
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeErrorDetail(w, r, http.StatusServiceUnavailable, wechatMPFollowQRUserMessage(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"temp_id":    ticket.ID,
		"qr_src":     wechatMPShowQRSrc(ticket.WechatTicket),
		"expires_at": ticket.ExpireAt.UTC().Format(time.RFC3339),
	})
}

type issuedMPFollowQR struct {
	ID           string
	WechatTicket string
	ExpireAt     time.Time
}

func issueOrReuseMPFollowTicket(ctx context.Context, app *WeChatAppConfig, userID string) (issuedMPFollowQR, error) {
	if existing, ok := loadReusableMPFollowTicket(userID); ok {
		return existing, nil
	}
	tempID := strconv.FormatInt(generateSnowflakeID(), 10)
	wxTicket, err := createWechatMPSceneQR(app, tempID)
	if err != nil {
		return issuedMPFollowQR{}, err
	}
	now := time.Now().UTC()
	expireAt := now.Add(time.Duration(wechatMPFollowQRExpireSeconds) * time.Second)
	if db == nil {
		return issuedMPFollowQR{}, fmt.Errorf("db not open")
	}
	_, err = db.Exec(`
		INSERT INTO auth_wechat_mp_follow_ticket
			(id, user_id, status, wechat_ticket, expire_at, conflict_code, mp_openid, unionid, created_at, updated_at)
		VALUES (?, ?, 'pending', ?, ?, '', '', '', ?, ?)`,
		tempID, userID, wxTicket, expireAt, timeNowUTC(), timeNowUTC())
	if err != nil {
		return issuedMPFollowQR{}, err
	}
	slog.InfoContext(ctx, "wechat_mp_follow_qr_issued",
		"level", "info",
		"ticket_status", "pending",
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
	return issuedMPFollowQR{ID: tempID, WechatTicket: wxTicket, ExpireAt: expireAt}, nil
}

func loadReusableMPFollowTicket(userID string) (issuedMPFollowQR, bool) {
	if db == nil || userID == "" {
		return issuedMPFollowQR{}, false
	}
	var out issuedMPFollowQR
	err := db.QueryRow(`
		SELECT id, wechat_ticket, expire_at
		FROM auth_wechat_mp_follow_ticket
		WHERE user_id = ? AND status = 'pending' AND expire_at > ? AND wechat_ticket != ''
		ORDER BY created_at DESC LIMIT 1`, userID, time.Now().UTC()).
		Scan(&out.ID, &out.WechatTicket, &out.ExpireAt)
	if err != nil {
		return issuedMPFollowQR{}, false
	}
	return out, true
}

func wechatMPFollowQRUserMessage(err error) string {
	var apiErr wechatMPAPIError
	if errors.As(err, &apiErr) && apiErr.Code == 40164 {
		if ip := wechatMPInvalidIPFromMsg(apiErr.Msg); ip != "" {
			return fmt.Sprintf("服务号接口未授权本机出口 IP（%s），无法生成关注二维码。请将该地址加入公众平台 IP 白名单后刷新本页。", ip)
		}
		return "服务号接口未授权本机出口 IP，无法生成关注二维码。请将服务器公网 IP 加入公众平台 IP 白名单后刷新本页。"
	}
	return "wechat qrcode create failed"
}

func createWechatMPSceneQR(app *WeChatAppConfig, sceneStr string) (string, error) {
	token, err := getWechatMPAccessToken(app)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", wechatMPAPIError{Code: -1, Msg: "empty access_token"}
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"expire_seconds": wechatMPFollowQRExpireSeconds,
		"action_name":    "QR_STR_SCENE",
		"action_info": map[string]interface{}{
			"scene": map[string]string{"scene_str": sceneStr},
		},
	})
	body, err := wechatHTTPPostJSON("https://api.weixin.qq.com/cgi-bin/qrcode/create?access_token="+url.QueryEscape(token), payload)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Ticket  string `json:"ticket"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.ErrCode != 0 || strings.TrimSpace(parsed.Ticket) == "" {
		return "", fmt.Errorf("wechat qrcode/create: %d %s", parsed.ErrCode, parsed.ErrMsg)
	}
	return strings.TrimSpace(parsed.Ticket), nil
}

func wechatMPShowQRSrc(wxTicket string) string {
	return "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=" + url.QueryEscape(wxTicket)
}

func loadMPFollowTicket(tempID string) (mpFollowTicket, bool) {
	if db == nil || tempID == "" {
		return mpFollowTicket{}, false
	}
	var row mpFollowTicket
	err := db.QueryRow(`
		SELECT id, user_id, status, expire_at, IFNULL(conflict_owner_user_id, ''), conflict_code
		FROM auth_wechat_mp_follow_ticket WHERE id = ?`, tempID).
		Scan(&row.ID, &row.UserID, &row.Status, &row.ExpireAt, &row.ConflictOwnerUserID, &row.ConflictCode)
	if err != nil {
		return mpFollowTicket{}, false
	}
	return row, true
}

// processWechatMPFollowTicket matches scene to a ticket. Returns true when the
// temp_id exists (caller must not fall through to the no-scene pending path).
func processWechatMPFollowTicket(ctx context.Context, app *WeChatAppConfig, tempID, openID, unionID string) bool {
	ticket, ok := loadMPFollowTicket(tempID)
	if !ok {
		return false
	}
	now := time.Now().UTC()
	if ticket.Status == "pending" && now.After(ticket.ExpireAt.UTC()) {
		updateMPFollowTicket(tempID, "expired", "", "", openID, unionID, "")
		slog.InfoContext(ctx, "wechat_mp_ticket_expired",
			"level", "info",
			"ticket_status", "expired",
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return true
	}
	if ticket.Status == "bound" || ticket.Status == "conflict" {
		return true
	}
	if openID == "" {
		slog.WarnContext(ctx, "wechat_mp_ticket_missing_openid",
			"level", "warn",
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return true
	}
	if owner, hit := findWechatUserByUnionID(unionID); hit && owner != ticket.UserID {
		markMPFollowTicketConflict(ctx, ticket, openID, unionID, owner, wechatMPConflictUnionOther)
		return true
	}
	if owner, hit := findWechatUserByAppOpenID(domain.WechatMPAppKey, openID); hit && owner != ticket.UserID {
		markMPFollowTicketConflict(ctx, ticket, openID, unionID, owner, wechatMPConflictOpenIDOther)
		return true
	}
	appID := ""
	if app != nil {
		appID = app.AppID
	}
	upsertWechatIdentity(ticket.UserID, domain.WechatMPAppKey, appID, openID, unionID, "", "", timeNowUTC())
	deleteMpSubscribePending(unionID)
	updateMPFollowTicket(tempID, "bound", "", "", openID, unionID, "")
	go publishWechatMpSubscribedTicket(context.Background(), ticket.UserID, openID, unionID, "bound", tempID)
	slog.InfoContext(ctx, "wechat_mp_subscribed",
		"level", "info",
		"outcome", "bound",
		"ticket_status", "bound",
		"user_id", ticket.UserID,
		"openid_fp", wechatIDFingerprint(openID),
		"unionid_fp", wechatIDFingerprint(unionID),
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
	return true
}

func markMPFollowTicketConflict(ctx context.Context, ticket mpFollowTicket, openID, unionID, ownerID, code string) {
	appKey := domain.WechatMPAppKey
	updateMPFollowTicket(ticket.ID, "conflict", ownerID, code, openID, unionID, "")
	go publishWechatIdentityConflict(context.Background(), ownerID, appKey, openID, unionID, "")
	go publishWechatMpSubscribedTicket(context.Background(), ticket.UserID, openID, unionID, "conflict", ticket.ID)
	slog.InfoContext(ctx, "wechat_mp_subscribed",
		"level", "info",
		"outcome", "conflict",
		"ticket_status", "conflict",
		"conflict_code", code,
		"openid_fp", wechatIDFingerprint(openID),
		"unionid_fp", wechatIDFingerprint(unionID),
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
}

func updateMPFollowTicket(tempID, status, conflictOwner, conflictCode, openID, unionID, _ string) {
	if db == nil || tempID == "" {
		return
	}
	var owner interface{}
	if strings.TrimSpace(conflictOwner) == "" {
		owner = nil
	} else {
		owner = conflictOwner
	}
	_, err := db.Exec(`
		UPDATE auth_wechat_mp_follow_ticket
		SET status = ?, conflict_owner_user_id = ?, conflict_code = ?, mp_openid = ?, unionid = ?, updated_at = ?
		WHERE id = ?`,
		status, owner, conflictCode, openID, unionID, timeNowUTC(), tempID)
	if err != nil {
		slog.Error("wechat_mp_ticket_update", "level", "error", "error", err.Error())
	}
}

func followTicketStatusForUser(userID string) (status, conflictCode, message string) {
	if db == nil || userID == "" {
		return "none", "", ""
	}
	var st, code string
	var expireAt time.Time
	err := db.QueryRow(`
		SELECT status, conflict_code, expire_at
		FROM auth_wechat_mp_follow_ticket
		WHERE user_id = ?
		ORDER BY created_at DESC LIMIT 1`, userID).Scan(&st, &code, &expireAt)
	if err != nil {
		return "none", "", ""
	}
	if st == "pending" && time.Now().UTC().After(expireAt.UTC()) {
		return "expired", "", "二维码已过期，请刷新页面重新获取。"
	}
	if st == "conflict" {
		msg := wechatMPConflictMessage
		return st, code, msg
	}
	return st, code, ""
}
