package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

var taskReferralHTTP = tracelog.DirectClient(8 * time.Second)

// lookupReferrerPaytimeQualification 是支付时刻资格接缝：默认 HTTP 问 taskReferral。
// 单测替换为确定性 true/false，避免绑边 commission_eligible 快照冒充现查。
var lookupReferrerPaytimeQualification = lookupReferrerPaytimeQualificationLive

func lookupReferrerPaytimeQualificationLive(ctx context.Context, referrerUserID string) bool {
	referrerUserID = strings.TrimSpace(referrerUserID)
	if referrerUserID == "" {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskReferralBaseURL), "/")
	if base == "" {
		return false
	}
	u := base + "/api/internal/referral/qualification/active/?user_id=" + url.QueryEscape(referrerUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false
	}
	if sec := strings.TrimSpace(cfg.TaskReferralInternalSecret); sec != "" {
		req.Header.Set("X-TaskReferral-Internal-Secret", sec)
	}
	resp, err := taskReferralHTTP.Do(req)
	if err != nil {
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "paytime referral qualification lookup failed", "wechat_profit_sharing", map[string]string{
			"err": truncateBytes([]byte(err.Error()), 300),
		})
		return false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		tracelog.EmitWithTrace(tracelog.TraceIDFromContext(ctx), "warn", "paytime referral qualification lookup status", "wechat_profit_sharing", map[string]string{
			"http_status": fmt.Sprintf("%d", resp.StatusCode),
		})
		return false
	}
	var out struct {
		Active bool `json:"active"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return false
	}
	return out.Active
}
