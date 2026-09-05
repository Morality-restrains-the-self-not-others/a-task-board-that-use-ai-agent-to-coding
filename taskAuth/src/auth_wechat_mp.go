package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"taskAuth/domain"
	"tracelog"
)

type wechatMPAPIError struct {
	Code int
	Msg  string
}

func (e wechatMPAPIError) Error() string {
	return fmt.Sprintf("wechat mp api %d %s", e.Code, e.Msg)
}

var wechatInvalidIPRe = regexp.MustCompile(`invalid ip ([0-9.]+)`)

func wechatMPInvalidIPFromMsg(msg string) string {
	m := wechatInvalidIPRe.FindStringSubmatch(msg)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

const wechatMPCallbackSuccess = "success"

var fetchWechatMPUnionID = fetchWechatMPUnionIDLive

var (
	wechatMPTokenMu      sync.Mutex
	wechatMPAccessToken  string
	wechatMPTokenExpires time.Time
)

func handleWeChatMPCallback(w http.ResponseWriter, r *http.Request) {
	app := wechatMPApp()
	if app == nil {
		slog.ErrorContext(r.Context(), "wechat_mp_callback_unconfigured",
			"level", "error",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		http.Error(w, "wechat mp not configured", http.StatusServiceUnavailable)
		return
	}
	q := r.URL.Query()
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	signature := q.Get("signature")
	if !domain.WechatMPCheckSignature(app.Token, timestamp, nonce, signature) {
		slog.WarnContext(r.Context(), "wechat_mp_callback_bad_signature",
			"level", "warn",
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, q.Get("echostr"))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	msg, err := parseWechatMPXML(body)
	if err != nil {
		slog.WarnContext(r.Context(), "wechat_mp_callback_xml",
			"level", "warn",
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeWechatMPSuccess(w)
		return
	}
	if msg.Encrypt != "" {
		msgSig := q.Get("msg_signature")
		if !wechatMPMsgSignatureOK(app.Token, timestamp, nonce, msg.Encrypt, msgSig) {
			http.Error(w, "invalid msg_signature", http.StatusForbidden)
			return
		}
		plain, decErr := decryptWechatMPAES(app.EncodingAESKey, msg.Encrypt)
		if decErr != nil {
			slog.WarnContext(r.Context(), "wechat_mp_callback_decrypt",
				"level", "warn",
				"error", decErr.Error(),
				"has_aes_key", strings.TrimSpace(app.EncodingAESKey) != "",
				"trace_id", tracelog.TraceIDFromContext(r.Context()),
			)
			writeWechatMPSuccess(w)
			return
		}
		msg, err = parseWechatMPXML(plain)
		if err != nil {
			writeWechatMPSuccess(w)
			return
		}
	}
	processWechatMPEvent(r.Context(), app, msg)
	writeWechatMPSuccess(w)
}

func writeWechatMPSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, wechatMPCallbackSuccess)
}

func processWechatMPEvent(ctx context.Context, app *WeChatAppConfig, msg wechatMPMessage) {
	if domain.WechatMPSubscribeCreatesUser() {
		return
	}
	openID := strings.TrimSpace(msg.FromUserName)
	if domain.WechatMPIsUnsubscribeEvent(msg.MsgType, msg.Event) {
		deleteMpSubscribePendingByOpenID(openID)
		slog.InfoContext(ctx, "wechat_mp_unsubscribe",
			"level", "info",
			"openid_fp", wechatIDFingerprint(openID),
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	if !domain.WechatMPIsFollowScanEvent(msg.MsgType, msg.Event) {
		slog.InfoContext(ctx, "wechat_mp_event_ignored",
			"level", "info",
			"event", strings.TrimSpace(msg.Event),
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	unionID := strings.TrimSpace(msg.UnionID)
	if unionID == "" {
		fetched, ferr := fetchWechatMPUnionID(app, openID)
		if ferr != nil {
			slog.WarnContext(ctx, "wechat_mp_user_info",
				"level", "warn",
				"error", ferr.Error(),
				"openid_fp", wechatIDFingerprint(openID),
				"trace_id", tracelog.TraceIDFromContext(ctx),
			)
		}
		unionID = strings.TrimSpace(fetched)
	}
	tempID := domain.WechatMPSceneTempID(msg.EventKey)
	if tempID != "" && processWechatMPFollowTicket(ctx, app, tempID, openID, unionID) {
		return
	}
	if !domain.WechatMPIsSubscribeEvent(msg.MsgType, msg.Event) {
		slog.InfoContext(ctx, "wechat_mp_event_ignored",
			"level", "info",
			"event", strings.TrimSpace(msg.Event),
			"has_scene", tempID != "",
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	if openID == "" || unionID == "" {
		slog.WarnContext(ctx, "wechat_mp_subscribe_missing_ids",
			"level", "warn",
			"has_openid", openID != "",
			"has_unionid", unionID != "",
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	appID := ""
	if app != nil {
		appID = app.AppID
	}
	if userID, ok := findWechatUserByUnionID(unionID); ok {
		upsertWechatIdentity(userID, domain.WechatMPAppKey, appID, openID, unionID, "", "", timeNowUTC())
		deleteMpSubscribePending(unionID)
		go publishWechatMpSubscribed(context.Background(), userID, openID, unionID, "bound")
		slog.InfoContext(ctx, "wechat_mp_subscribed",
			"level", "info",
			"outcome", "bound",
			"user_id", userID,
			"openid_fp", wechatIDFingerprint(openID),
			"unionid_fp", wechatIDFingerprint(unionID),
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
		return
	}
	saveMpSubscribePending(unionID, openID, appID)
	go publishWechatMpSubscribed(context.Background(), "", openID, unionID, "pending")
	slog.InfoContext(ctx, "wechat_mp_subscribed",
		"level", "info",
		"outcome", "pending",
		"openid_fp", wechatIDFingerprint(openID),
		"unionid_fp", wechatIDFingerprint(unionID),
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
}

func handleWeChatMPFollowStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	unionID := wechatUnionIDOf(userID)
	bound := userHasMPWechatIdentity(userID)
	if !bound && unionID != "" {
		claimMpSubscribePending(userID, unionID)
		bound = userHasMPWechatIdentity(userID)
	}
	if !bound {
		claimMPFollowFromLiveQRScene(r.Context(), userID)
		bound = userHasMPWechatIdentity(userID)
	}
	ticketStatus, conflictCode, message := followTicketStatusForUser(userID)
	slog.InfoContext(r.Context(), "wechat_mp_follow_status",
		"level", "info",
		"bound", bound,
		"has_unionid", unionID != "",
		"ticket_status", ticketStatus,
		"conflict_code", conflictCode,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bound":         bound,
		"has_unionid":   unionID != "",
		"ticket_status": ticketStatus,
		"conflict_code": conflictCode,
		"message":       message,
	})
}

func userHasMPWechatIdentity(userID string) bool {
	if userID == "" || db == nil {
		return false
	}
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM wechat_identity
		WHERE user_id = ? AND app_key = ? AND openid != ''`,
		userID, domain.WechatMPAppKey).Scan(&n)
	return err == nil && n > 0
}

func saveMpSubscribePending(unionID, openID, appID string) {
	if db == nil || unionID == "" {
		return
	}
	_, err := db.Exec(`
		INSERT INTO auth_wechat_mp_subscribe_pending (unionid, openid, app_id, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE openid = VALUES(openid), app_id = VALUES(app_id), created_at = VALUES(created_at)`,
		unionID, openID, appID, timeNowUTC())
	if err != nil {
		slog.Error("wechat_mp_pending_save", "level", "error", "error", err.Error())
	}
}

func deleteMpSubscribePending(unionID string) {
	if db == nil || unionID == "" {
		return
	}
	_, _ = db.Exec(`DELETE FROM auth_wechat_mp_subscribe_pending WHERE unionid = ?`, unionID)
}

func deleteMpSubscribePendingByOpenID(openID string) {
	if db == nil || openID == "" {
		return
	}
	_, _ = db.Exec(`DELETE FROM auth_wechat_mp_subscribe_pending WHERE openid = ?`, openID)
}

func claimMpSubscribePending(userID, unionID string) {
	if db == nil || userID == "" || unionID == "" {
		return
	}
	var openID, appID string
	err := db.QueryRow(`
		SELECT openid, app_id FROM auth_wechat_mp_subscribe_pending WHERE unionid = ?`, unionID).
		Scan(&openID, &appID)
	if err != nil {
		return
	}
	_, _ = db.Exec(`DELETE FROM auth_wechat_mp_subscribe_pending WHERE unionid = ?`, unionID)
	if strings.TrimSpace(openID) == "" {
		return
	}
	upsertWechatIdentity(userID, domain.WechatMPAppKey, appID, openID, unionID, "", "", timeNowUTC())
	go publishWechatMpSubscribed(context.Background(), userID, openID, unionID, "bound")
}

func wechatIDFingerprint(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 8 {
		return "***"
	}
	return id[:4] + "…" + id[len(id)-4:]
}

func fetchWechatMPUnionIDLive(app *WeChatAppConfig, openID string) (string, error) {
	if app == nil || strings.TrimSpace(app.AppSecret) == "" || strings.TrimSpace(openID) == "" {
		return "", nil
	}
	token, err := getWechatMPAccessToken(app)
	if err != nil || token == "" {
		return "", err
	}
	urlStr := "https://api.weixin.qq.com/cgi-bin/user/info?access_token=" + token +
		"&openid=" + openID + "&lang=zh_CN"
	body, err := wechatHTTPGet(urlStr)
	if err != nil {
		return "", err
	}
	var parsed struct {
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.ErrCode != 0 {
		return "", wechatMPAPIError{Code: parsed.ErrCode, Msg: parsed.ErrMsg}
	}
	return strings.TrimSpace(parsed.UnionID), nil
}

func getWechatMPAccessToken(app *WeChatAppConfig) (string, error) {
	wechatMPTokenMu.Lock()
	defer wechatMPTokenMu.Unlock()
	if wechatMPAccessToken != "" && time.Now().Before(wechatMPTokenExpires) {
		return wechatMPAccessToken, nil
	}
	urlStr := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=" +
		app.AppID + "&secret=" + app.AppSecret
	body, err := wechatHTTPGet(urlStr)
	if err != nil {
		return "", err
	}
	token, exp, err := parseWechatMPAccessTokenBody(body)
	if err != nil {
		wechatMPAccessToken = ""
		wechatMPTokenExpires = time.Time{}
		return "", err
	}
	wechatMPAccessToken = token
	if exp <= 0 {
		exp = 7200
	}
	wechatMPTokenExpires = time.Now().Add(time.Duration(exp-60) * time.Second)
	return wechatMPAccessToken, nil
}

func parseWechatMPAccessTokenBody(body []byte) (token string, expiresIn int, err error) {
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, err
	}
	if parsed.ErrCode != 0 {
		return "", 0, wechatMPAPIError{Code: parsed.ErrCode, Msg: parsed.ErrMsg}
	}
	token = strings.TrimSpace(parsed.AccessToken)
	if token == "" {
		return "", 0, wechatMPAPIError{Code: -1, Msg: "empty access_token"}
	}
	return token, parsed.ExpiresIn, nil
}
