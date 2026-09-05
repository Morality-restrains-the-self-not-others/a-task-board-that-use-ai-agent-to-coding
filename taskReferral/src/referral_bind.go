package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

type referralBindDeps struct {
	lookupOwner      func(code string) (string, error)
	lookupTenant     func(referrerID string) (int64, error)
	hasQualification func(referrerID string) bool
	syncEdge         func(referrer, referred string, tenantID int64, boundAt string, commissionEligible bool, channelCode string) error
}

func (d referralBindDeps) Bind(referredUserID, accessCode string) (status, reason string, err error) {
	referredUserID = strings.TrimSpace(referredUserID)
	accessCode = strings.TrimSpace(accessCode)
	if referredUserID == "" {
		return "skipped", "empty_referred", nil
	}
	if accessCode == "" {
		return "skipped", "empty_code", nil
	}
	owner, err := d.lookupOwner(accessCode)
	if err != nil {
		return "error", "lookup_owner", err
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return "skipped", "unknown_code", nil
	}
	if owner == referredUserID {
		return "skipped", "self_referral", nil
	}
	tenantID, err := d.lookupTenant(owner)
	if err != nil {
		return "error", "lookup_tenant", err
	}
	if tenantID <= 0 {
		return "skipped", "no_tenant", nil
	}
	eligible := false
	if d.hasQualification != nil {
		eligible = d.hasQualification(owner)
	}
	if err := d.syncEdge(owner, referredUserID, tenantID, timeNowUTC(), eligible, accessCode); err != nil {
		return "error", "sync_edge", err
	}
	log.Printf("[taskReferral] referral_bind_edge referrer=%s referred=%s commission_eligible=%t", owner, referredUserID, eligible)
	return "ok", "", nil
}

func productionBindDeps() referralBindDeps {
	return referralBindDeps{
		lookupOwner:      lookupShareCodeOwner,
		lookupTenant:     lookupReferrerTenantHTTP,
		hasQualification: referrerHasActiveQualification,
		syncEdge:         syncReferralEdgeHTTP,
	}
}

func handleInternalBindFromCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"status": "method not allowed"})
		return
	}
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "forbidden"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid json"})
		return
	}
	referred := strField(body, "referred_user_id")
	code := strField(body, "access_code")
	if code == "" {
		code = strField(body, "accessCode")
	}
	st, reason, err := productionBindDeps().Bind(referred, code)
	if err != nil {
		slog.Warn("referral_bind_failed", "referred_user_id", referred, "reason", reason, "error", err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"status": st, "reason": reason})
		return
	}
	log.Printf("[taskReferral] referral_bind status=%s reason=%s referred=%s", st, reason, referred)
	out := map[string]string{"status": st}
	if reason != "" {
		out["reason"] = reason
	}
	writeJSON(w, http.StatusOK, out)
}

func lookupReferrerTenantHTTP(referrerUserID string) (int64, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TenantServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8020"
	}
	url := base + "/api/internal/tenant/companies/by-creator?creator_id=" + referrerUserID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if sec := strings.TrimSpace(cfg.TenantInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("tenant by-creator status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, nil
	}
	return parseJSONInt64(list[0]["id"])
}

func syncReferralEdgeHTTP(referrer, referred string, tenantID int64, boundAt string, commissionEligible bool, channelCode string) error {
	base := strings.TrimRight(strings.TrimSpace(cfg.BillServiceURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8004"
	}
	payload, err := json.Marshal(map[string]interface{}{
		"referrer_user_id":    referrer,
		"referred_user_id":    referred,
		"referrer_tenant_id":  strconv.FormatInt(tenantID, 10),
		"bound_at":            boundAt,
		"commission_eligible": commissionEligible,
		"channel_code":        strings.TrimSpace(channelCode),
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/taskbill/referral/sync-edge/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.BillInternalSecret); sec != "" {
		req.Header.Set("X-TaskBill-Internal-Secret", sec)
	} else {
		slog.Warn("sync_edge_missing_bill_secret")
	}
	resp, err := tracelog.DirectClient(8 * time.Second).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sync-edge status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func parseJSONInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case json.Number:
		return t.Int64()
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, nil
		}
		return strconv.ParseInt(s, 10, 64)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return 0, nil
		}
		return strconv.ParseInt(s, 10, 64)
	}
}
