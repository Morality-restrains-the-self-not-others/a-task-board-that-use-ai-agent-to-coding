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

// fetchQualificationsBatch 批量查 taskReferral 当前分账资格。
// ok=false 表示下游失败，调用方应将字段写成 JSON null。
func fetchQualificationsBatch(ctx context.Context, userIDs []string) (map[string]bool, bool) {
	if len(userIDs) == 0 {
		return map[string]bool{}, true
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.ReferralServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8025"
	}
	payload, err := json.Marshal(map[string]interface{}{"user_ids": userIDs})
	if err != nil {
		slog.WarnContext(ctx, "qualification_lookup_marshal_failed", "error", err.Error())
		return nil, false
	}
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/internal/referral/qualification/active/batch/", bytes.NewReader(payload))
	if err != nil {
		slog.WarnContext(ctx, "qualification_lookup_request_failed", "error", err.Error())
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.ReferralInternalSecret); sec != "" {
		req.Header.Set("X-TaskReferral-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(5 * time.Second).Do(req)
	if err != nil {
		slog.WarnContext(ctx, "qualification_lookup_http_failed", "error", err.Error(), "user_count", len(userIDs))
		return nil, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.WarnContext(ctx, "qualification_lookup_rejected", "status", resp.StatusCode, "user_count", len(userIDs), "duration_ms", time.Since(started).Milliseconds())
		return nil, false
	}
	var parsed struct {
		Qualifications map[string]bool `json:"qualifications"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		slog.WarnContext(ctx, "qualification_lookup_unmarshal_failed", "error", err.Error())
		return nil, false
	}
	if parsed.Qualifications == nil {
		parsed.Qualifications = map[string]bool{}
	}
	slog.InfoContext(ctx, "qualification_lookup_ok", "user_count", len(userIDs), "duration_ms", time.Since(started).Milliseconds())
	return parsed.Qualifications, true
}

func attachProfitSharingQualifications(users []map[string]interface{}, byUser map[string]bool, ok bool) {
	for _, u := range users {
		if !ok {
			u["has_profit_sharing_qualification"] = nil
			continue
		}
		uid, _ := u["id"].(string)
		u["has_profit_sharing_qualification"] = byUser[uid]
	}
}
