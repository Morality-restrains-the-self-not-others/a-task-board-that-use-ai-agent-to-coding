package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

func extractAccessCodeFromRegisterBody(body map[string]interface{}) string {
	if body == nil {
		return ""
	}
	code := strings.TrimSpace(strField(body, "access_code"))
	if code == "" {
		code = strings.TrimSpace(strField(body, "accessCode"))
	}
	return code
}

func bindReferralAfterRegister(ctx context.Context, userID, accessCode string) {
	userID = strings.TrimSpace(userID)
	accessCode = strings.TrimSpace(accessCode)
	if userID == "" || accessCode == "" {
		return
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.ReferralServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8025"
	}
	payload, err := json.Marshal(map[string]string{
		"referred_user_id": userID,
		"access_code":      accessCode,
	})
	if err != nil {
		slog.WarnContext(ctx, "referral_bind_marshal_failed", "error", err.Error())
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/referral/bind-from-code/", bytes.NewReader(payload))
	if err != nil {
		slog.WarnContext(ctx, "referral_bind_request_failed", "error", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.ReferralInternalSecret); sec != "" {
		req.Header.Set("X-TaskReferral-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		slog.WarnContext(ctx, "referral_bind_http_failed", "error", err.Error(), "user_id", userID)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "referral_bind_rejected",
			"status", resp.StatusCode, "user_id", userID, "body", strings.TrimSpace(string(raw)))
		return
	}
	slog.InfoContext(ctx, "referral_bind_done", "user_id", userID, "http_status", resp.StatusCode, "body", strings.TrimSpace(string(raw)))
}

func bindReferralAfterRegisterAsync(userID, accessCode string) {
	go bindReferralAfterRegister(context.Background(), userID, accessCode)
}

// validateAccessCodeInvitation 同步校验分享码是否为有效 active 码
// （taskReferral 内部端点 /api/internal/referral/share-code/validate/）。
// 返回 (valid, reachable)：reachable=false 表示下游不可达/异常，
// 调用方必须按 fail-closed 处理（注册门禁保持关闭），不得静默放行。
func validateAccessCodeInvitation(ctx context.Context, accessCode string) (bool, bool) {
	accessCode = strings.TrimSpace(accessCode)
	if accessCode == "" {
		return false, true
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.ReferralServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8025"
	}
	payload, err := json.Marshal(map[string]string{"code": accessCode})
	if err != nil {
		slog.WarnContext(ctx, "share_code_validate_marshal_failed", "error", err.Error())
		return false, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/referral/share-code/validate/", bytes.NewReader(payload))
	if err != nil {
		slog.WarnContext(ctx, "share_code_validate_request_failed", "error", err.Error())
		return false, false
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.ReferralInternalSecret); sec != "" {
		req.Header.Set("X-TaskReferral-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(5 * time.Second).Do(req)
	if err != nil {
		slog.WarnContext(ctx, "share_code_validate_http_failed", "error", err.Error())
		return false, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "share_code_validate_rejected",
			"status", resp.StatusCode, "body", strings.TrimSpace(string(raw)))
		return false, false
	}
	var parsed struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.WarnContext(ctx, "share_code_validate_unmarshal_failed", "error", err.Error())
		return false, false
	}
	slog.InfoContext(ctx, "share_code_validate_done", "valid", parsed.Valid, "code_len", len(accessCode))
	return parsed.Valid, true
}
