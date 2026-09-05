package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// disableBillEligibility 默认 no-op，避免单测打到本机 :8004。
var disableBillEligibility = func(string) error { return nil }

func disableBillEligibilityLive(userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BillServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8004" // 同节点调用，拆分时改 ${subdomains.xxx}
	}
	payload, err := json.Marshal(map[string]string{
		"referrer_user_id": userID,
		"reason":           "qualification_revoked",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/taskbill/referral/disable-eligibility/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.BillInternalSecret); sec != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("bill_status_%d", resp.StatusCode)
	}
	return nil
}
