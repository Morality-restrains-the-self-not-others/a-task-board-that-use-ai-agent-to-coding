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

// fetchReferrersBatch 批量查 taskReferral 内部接口，返回 referred_user_id →
// {referrer_user_id, channel_code}。best-effort：服务不可达/异常时返回空 map，
// 用户列表仍正常返回（OPT-20260821-031）。
func fetchReferrersBatch(ctx context.Context, userIDs []string) map[string]map[string]string {
	result := make(map[string]map[string]string)
	if len(userIDs) == 0 {
		return result
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.ReferralServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8025"
	}
	payload, err := json.Marshal(map[string]interface{}{"user_ids": userIDs})
	if err != nil {
		slog.WarnContext(ctx, "referrer_lookup_marshal_failed", "error", err.Error())
		return result
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/referral/referrers/lookup/", bytes.NewReader(payload))
	if err != nil {
		slog.WarnContext(ctx, "referrer_lookup_request_failed", "error", err.Error())
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.ReferralInternalSecret); sec != "" {
		req.Header.Set("X-TaskReferral-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(5 * time.Second).Do(req)
	if err != nil {
		slog.WarnContext(ctx, "referrer_lookup_http_failed", "error", err.Error())
		return result
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "referrer_lookup_rejected", "status", resp.StatusCode, "body", strings.TrimSpace(string(raw)))
		return result
	}
	var parsed struct {
		Referrers map[string]map[string]string `json:"referrers"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.WarnContext(ctx, "referrer_lookup_unmarshal_failed", "error", err.Error())
		return result
	}
	return parsed.Referrers
}

// referrerDisplayName 把推荐人 user_id 解析成管理员列表可读的展示名：
// profile username → email/phone 登录方式 identifier → 原始 user_id。
func referrerDisplayName(referrerID string, profiles map[string]string, logins map[string][]map[string]interface{}) string {
	if uname := strings.TrimSpace(profiles[referrerID]); uname != "" {
		return uname
	}
	for _, lm := range logins[referrerID] {
		mt, _ := lm["method_type"].(string)
		identifier, _ := lm["identifier"].(string)
		if identifier != "" && (mt == "email" || mt == "phone") {
			return identifier
		}
	}
	return referrerID
}
