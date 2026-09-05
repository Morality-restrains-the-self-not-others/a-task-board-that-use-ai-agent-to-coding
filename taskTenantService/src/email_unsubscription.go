package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

const emailUnsubscribedHint = "该邮箱已退订邮件邀请，请手动复制邀请链接给对方"

type emailUnsubLookup struct {
	Unsubscribed   bool   `json:"unsubscribed"`
	UnsubscribeURL string `json:"unsubscribe_url"`
}

var checkEmailUnsubscribedFn = lookupEmailUnsubscriptionHTTP

func lookupEmailUnsubscriptionHTTP(email string) (unsubscribed bool, unsubscribeURL string) {
	email = strings.TrimSpace(email)
	if email == "" || cfg.TaskAuthURL == "" {
		return false, ""
	}
	u := strings.TrimRight(cfg.TaskAuthURL, "/") + "/api/internal/email-unsubscription/?email=" + url.QueryEscape(email)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		slog.Warn("email_unsubscription_lookup_build", "level", "warn", "err", err.Error())
		return false, ""
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Warn("email_unsubscription_lookup_failed", "level", "warn", "err", err.Error())
		return false, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Warn("email_unsubscription_lookup_status", "level", "warn", "status", resp.StatusCode)
		return false, ""
	}
	var out emailUnsubLookup
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		slog.Warn("email_unsubscription_lookup_decode", "level", "warn", "err", err.Error())
		return false, ""
	}
	return out.Unsubscribed, out.UnsubscribeURL
}
