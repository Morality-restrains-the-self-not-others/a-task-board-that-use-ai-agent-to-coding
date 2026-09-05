package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

type saasApplicant struct {
	UID   int64
	Email string
}

func hasMainSiteSessionCookie(r *http.Request) bool {
	if r == nil {
		return false
	}
	if c, err := r.Cookie("token"); err == nil && strings.TrimSpace(c.Value) != "" {
		return true
	}
	if c, err := r.Cookie("userId"); err == nil && strings.TrimSpace(c.Value) != "" {
		return true
	}
	return false
}

// resolveSaasApplicant 优先用网关注入的 X-User-Id；provider 同源无网关头时，
// 仅在存在主站 session Cookie 时调用 taskAuth forward-auth（不转发 vendor JWT）。
func (a *App) resolveSaasApplicant(r *http.Request) *saasApplicant {
	if r == nil {
		return nil
	}
	uidStr := strings.TrimSpace(r.Header.Get("X-User-Id"))
	email := strings.ToLower(strings.TrimSpace(r.Header.Get("X-User-Email")))
	if uid, err := strconv.ParseInt(uidStr, 10, 64); err == nil && uid > 0 {
		return &saasApplicant{UID: uid, Email: email}
	}
	if !hasMainSiteSessionCookie(r) {
		return nil
	}
	return a.resolveSaasApplicantViaForwardAuth(r)
}

func (a *App) resolveSaasApplicantViaForwardAuth(r *http.Request) *saasApplicant {
	if a == nil || a.Cfg == nil {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return nil
	}
	reqURL := base + "/api/internal/gateway/forward-auth/"
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		logWarn(r.Context(), "event=VendorApplicantForwardAuthBuildFailed err=%v", err)
		return nil
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cookie := strings.TrimSpace(r.Header.Get("Cookie")); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if secret := strings.TrimSpace(a.Cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Do(req)
	if err != nil {
		logWarn(r.Context(), "event=VendorApplicantForwardAuthFailed err=%v", err)
		return nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		logWarn(r.Context(), "event=VendorApplicantForwardAuthDenied status=%d", resp.StatusCode)
		return nil
	}
	uidStr := strings.TrimSpace(resp.Header.Get("X-User-Id"))
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || uid <= 0 {
		return nil
	}
	email := strings.ToLower(strings.TrimSpace(resp.Header.Get("X-User-Email")))
	logInfo("event=VendorApplicantResolved via=cookie user_id=%d", uid)
	return &saasApplicant{UID: uid, Email: email}
}

func (a *App) requireVendorApplicant(w http.ResponseWriter, r *http.Request) (uid int64, email string, ok bool) {
	applicant := a.resolveSaasApplicant(r)
	if applicant == nil || applicant.UID <= 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return 0, "", false
	}
	email = applicant.Email
	if email == "" || isSyntheticEmail(email) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "厂商门户需先绑定邮箱账号，请前往个人资料页绑定邮箱"})
		return 0, "", false
	}
	return applicant.UID, email, true
}
