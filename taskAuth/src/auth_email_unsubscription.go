package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"taskAuth/domain"
	"tracelog"
)

func unsubscribeHMACSecret() string {
	if s := strings.TrimSpace(cfg.UnsubscribeHmacSecret); s != "" {
		return s
	}
	return strings.TrimSpace(cfg.SSOJwtSecret)
}

func publicUnsubscribeAPIURL(token string) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.GatewayPublicBase), "/")
	if base == "" {
		base = "http://127.0.0.1:8003"
	}
	return base + "/api/public/email-unsubscribe/?token=" + url.QueryEscape(token)
}

func frontendUnsubscribeConfirmURL(ok bool, token string) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.FrontendBase), "/")
	if base == "" {
		base = "http://localhost:4000"
	}
	flag := "0"
	if ok {
		flag = "1"
	}
	u := base + "/auth/unsubscribe/?ok=" + flag
	// OPT-20260829-002: 确认页携带同一 HMAC token，供「重新接收邀请邮件」按钮复用
	// （与退订同一公开 token，不需要换新）。
	if token != "" {
		u += "&token=" + url.QueryEscape(token)
	}
	return u
}

func attachUnsubscribeURL(contextData map[string]interface{}, email string) {
	if contextData == nil {
		return
	}
	tok, err := domain.SignUnsubscribeToken(unsubscribeHMACSecret(), email)
	if err != nil {
		slog.Warn("invite_unsubscribe_url_skipped", "level", "warn", "err", err.Error())
		return
	}
	contextData["unsubscribe_url"] = publicUnsubscribeAPIURL(tok)
}

func isEmailUnsubscribed(email string) bool {
	norm, err := domain.NormalizeInviteEmail(email)
	if err != nil {
		return false
	}
	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM auth_email_unsubscription WHERE email = ?`, norm).Scan(&n)
	return err == nil && n > 0
}

func lookupUnsubscription(email string) (unsubscribed bool, unsubscribeURL string) {
	norm, err := domain.NormalizeInviteEmail(email)
	if err != nil {
		return false, ""
	}
	tok, tokErr := domain.SignUnsubscribeToken(unsubscribeHMACSecret(), norm)
	if tokErr == nil {
		unsubscribeURL = publicUnsubscribeAPIURL(tok)
	}
	var n int
	qerr := db.QueryRow(`SELECT COUNT(*) FROM auth_email_unsubscription WHERE email = ?`, norm).Scan(&n)
	return qerr == nil && n > 0, unsubscribeURL
}

func upsertEmailUnsubscription(ctx context.Context, email, source string) (inserted bool, err error) {
	norm, err := domain.NormalizeInviteEmail(email)
	if err != nil {
		return false, err
	}
	if source == "" {
		source = domain.UnsubscribeSourceInvite
	}
	var existing string
	err = db.QueryRow(`SELECT email FROM auth_email_unsubscription WHERE email = ?`, norm).Scan(&existing)
	if err == nil {
		return false, nil
	}
	if err != sql.ErrNoRows {
		return false, err
	}
	id := generateSnowflakeID()
	_, err = db.Exec(`INSERT INTO auth_email_unsubscription (id, email, source) VALUES (?, ?, ?)`, id, norm, source)
	if err != nil {
		// unique race
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return false, nil
		}
		return false, err
	}
	publishEmailUnsubscribed(ctx, norm, source)
	slog.InfoContext(ctx, "email_unsubscribed",
		"level", "info",
		"source", source,
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)
	return true, nil
}

func handlePublicEmailUnsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	source := domain.UnsubscribeSourceInvite
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if token == "" {
			token = strings.TrimSpace(r.Form.Get("token"))
		}
		if strings.Contains(r.Form.Get("List-Unsubscribe"), "One-Click") ||
			strings.TrimSpace(r.Form.Get("List-Unsubscribe")) == "One-Click" ||
			strings.Contains(r.PostForm.Encode(), "List-Unsubscribe=One-Click") {
			source = domain.UnsubscribeSourceList
		}
	}
	wantsJSON := strings.Contains(r.Header.Get("Accept"), "application/json") || r.Method == http.MethodPost
	email, err := domain.ParseUnsubscribeToken(unsubscribeHMACSecret(), token)
	if err != nil {
		if wantsJSON {
			writeError(w, r, http.StatusBadRequest, "链接无效")
			return
		}
		http.Redirect(w, r, frontendUnsubscribeConfirmURL(false, ""), http.StatusFound)
		return
	}
	if _, err := upsertEmailUnsubscription(r.Context(), email, source); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if wantsJSON && r.Method == http.MethodPost {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	http.Redirect(w, r, frontendUnsubscribeConfirmURL(true, token), http.StatusFound)
}

// handlePublicEmailResubscribe POST /api/public/email-resubscribe/?token=<hmac>
// 幂等移除 auth_email_unsubscription 行，使该邮箱恢复接收邀请邮件（OPT-20260829-002）。
// 与退订共用同一 HMAC token；写操作需 Idempotency-Key（FE clickGuard 每击新 key）。
func handlePublicEmailResubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "Idempotency-Key required")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		_ = r.ParseForm()
		token = strings.TrimSpace(r.Form.Get("token"))
	}
	email, err := domain.ParseUnsubscribeToken(unsubscribeHMACSecret(), token)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "链接无效")
		return
	}
	if _, err := removeEmailUnsubscription(r.Context(), email); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "resubscribed": true})
}

func handleInternalEmailUnsubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	email := strings.TrimSpace(r.URL.Query().Get("email"))
	unsub, link := lookupUnsubscription(email)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"unsubscribed":    unsub,
		"unsubscribe_url": link,
	})
}

func publishEmailUnsubscribed(ctx context.Context, email, source string) {
	data := map[string]interface{}{
		"email":           email,
		"source":          source,
		"unsubscribed_at": timeNowUTC(),
	}
	if err := publishDomainEventKafka(ctx, "EMAIL_UNSUBSCRIBED", data, email); err != nil {
		if err != errKafkaNotConfigured {
			slog.WarnContext(ctx, "email_unsubscribed_publish_failed", "level", "warn", "err", err.Error())
		}
	}
}

// removeEmailUnsubscription 幂等删除 auth_email_unsubscription 行。
// 已不在退订名单（无行可删）时返回 removed=false 且无错误（幂等）。
func removeEmailUnsubscription(ctx context.Context, email string) (removed bool, err error) {
	norm, err := domain.NormalizeInviteEmail(email)
	if err != nil {
		return false, err
	}
	result, err := db.Exec(`DELETE FROM auth_email_unsubscription WHERE email = ?`, norm)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n > 0 {
		publishEmailResubscribed(ctx, norm)
		slog.InfoContext(ctx, "email_resubscribed",
			"level", "info",
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
	}
	return n > 0, nil
}

func publishEmailResubscribed(ctx context.Context, email string) {
	data := map[string]interface{}{
		"email":           email,
		"resubscribed_at": timeNowUTC(),
	}
	if err := publishDomainEventKafka(ctx, "EMAIL_RESUBSCRIBED", data, email); err != nil {
		if err != errKafkaNotConfigured {
			slog.WarnContext(ctx, "email_resubscribed_publish_failed", "level", "warn", "err", err.Error())
		}
	}
}
