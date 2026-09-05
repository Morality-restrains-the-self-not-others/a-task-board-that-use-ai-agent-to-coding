package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"authz"
	"tracelog"
)

const (
	adminShareReasonMinRunes = 8
	adminShareReasonMaxRunes = 500
	psStatusVoided           = "voided"
	psStatusReturned         = "returned"
)

// handleSystemAdminShareProfitSharing POST /api/system-admin/profit-sharing/{id}/share/
// 平台员工带审计缘由向微信发起分账；可绕过推荐人 15 天冻结。
func handleSystemAdminShareProfitSharing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		slog.WarnContext(r.Context(), "admin_profit_sharing_share_unauthorized",
			"level", "warn",
			"reason", "missing_gateway_verify",
		)
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		slog.WarnContext(r.Context(), "admin_profit_sharing_share_forbidden",
			"level", "warn",
			"reason", "not_platform_staff",
		)
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	id, err := parseProfitSharingSharePathID(r.URL.Path)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid profit sharing id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	actorID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if actorID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing user identity", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey == "" {
		writeErrorJSON(w, http.StatusBadRequest, "Idempotency-Key required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	reason := strings.TrimSpace(stringField(body, "reason"))
	n := utf8.RuneCountInString(reason)
	if n < adminShareReasonMinRunes || n > adminShareReasonMaxRunes {
		writeErrorJSON(w, http.StatusBadRequest, "reason must be 8-500 characters", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	impersonator := strings.TrimSpace(r.Header.Get("X-Impersonator-Id"))
	sessionID := strings.TrimSpace(r.Header.Get("X-Impersonation-Session-Id"))
	impersonating := impersonator != ""

	code, state, idempotent, errMsg := shareAdminProfitSharingRecord(r, id, actorID, impersonator, sessionID, impersonating, reason, idemKey)
	if code != http.StatusOK {
		writeErrorJSON(w, code, errMsg, tracelog.TraceIDFromContext(r.Context()))
		return
	}
	resp := map[string]interface{}{"status": "ok", "state": state}
	if idempotent {
		resp["idempotent"] = true
	}
	writeJSON(w, http.StatusOK, resp)
}

func parseProfitSharingSharePathID(path string) (int64, error) {
	p := strings.TrimSuffix(strings.TrimSpace(path), "/")
	if !strings.HasSuffix(p, "/share") {
		return 0, fmt.Errorf("not a share path")
	}
	p = strings.TrimSuffix(p, "/share")
	i := strings.LastIndex(p, "/")
	if i < 0 || i+1 >= len(p) {
		return 0, fmt.Errorf("missing id")
	}
	return parseIDField(p[i+1:])
}

func shareAdminProfitSharingRecord(r *http.Request, id int64, actorID, impersonator, sessionID string, impersonating bool, reason, idemKey string) (int, string, bool, string) {
	ctx := r.Context()
	if existing, ok := loadAdminShareByIdempotency(idemKey); ok {
		slog.InfoContext(ctx, "admin_profit_sharing_share_idempotent",
			"level", "info",
			"profit_sharing_id", id,
			"actor_user_id", actorID,
			"impersonating", impersonating,
			"impersonator_user_id", impersonator,
			"impersonated_user_id", actorID,
			"impersonation_session_id", sessionID,
			"outcome", existing.Outcome,
		)
		if existing.Outcome == "failed" {
			return retryAdminShareAfterFailedIdempotency(r, existing, actorID, impersonator, sessionID, impersonating)
		}
		state := "shared"
		if existing.Outcome == "processing" {
			state = "processing"
		}
		return http.StatusOK, state, true, ""
	}

	var rec profitSharingRecord
	err := db.QueryRow(`
		SELECT ps.id, ps.out_profit_sharing_no, ps.order_id, ps.order_number, ps.tenant_id,
		       ps.referrer_user_id, ps.referrer_openid,
		       ps.commission_yuan_cents, ps.total_yuan_cents,
		       ps.status, COALESCE(ps.fail_reason, '')
		FROM billing_profit_sharing ps
		WHERE ps.id = ?`, id).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid,
		&rec.CommissionYuanCents, &rec.TotalYuanCents,
		&rec.Status, &rec.FailReason,
	)
	if err == sql.ErrNoRows {
		return http.StatusNotFound, "", false, "profit sharing record not found"
	}
	if err != nil {
		slog.ErrorContext(ctx, "admin_profit_sharing_share_load_failed",
			"level", "error",
			"error", err.Error(),
			"profit_sharing_id", id,
		)
		return http.StatusInternalServerError, "", false, "加载分账记录失败"
	}
	switch rec.Status {
	case psStatusProcessing:
		return http.StatusOK, "processing", true, ""
	case psStatusFinished:
		return http.StatusOK, "shared", true, ""
	case psStatusVoided, psStatusReturned:
		return http.StatusConflict, "", false, "当前状态不可分账"
	case psStatusPending, psStatusFailed:
		// 超管可绕过推荐人冻结窗；微信业务拒绝（比例/金额）返回 409，渠道不可达才 502
	default:
		return http.StatusConflict, "", false, "当前状态不可分账"
	}

	actionID := generateSnowflakeID()
	now := time.Now().UTC()
	_, err = db.Exec(`
		INSERT INTO billing_profit_sharing_admin_action
		  (id, profit_sharing_id, order_id, actor_user_id, impersonator_user_id,
		   impersonation_session_id, reason, idempotency_key, outcome, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'accepted', ?, ?)`,
		actionID, rec.ID, rec.OrderID, actorID, impersonator, sessionID, reason, idemKey, now, now)
	if err != nil {
		if existing, ok := loadAdminShareByIdempotency(idemKey); ok {
			state := "shared"
			if existing.Outcome == "processing" {
				state = "processing"
			}
			return http.StatusOK, state, true, ""
		}
		slog.ErrorContext(ctx, "admin_profit_sharing_share_audit_failed",
			"level", "error",
			"error", err.Error(),
			"profit_sharing_id", rec.ID,
		)
		return http.StatusInternalServerError, "", false, "写入审计失败"
	}

	if err := executeProfitSharing(ctx, rec); err != nil {
		markProfitSharingStatus(ctx, rec.ID, psStatusFailed, err.Error())
		_, _ = db.Exec(`UPDATE billing_profit_sharing_admin_action SET outcome = 'failed', updated_at = ? WHERE id = ?`,
			time.Now().UTC(), actionID)
		slog.WarnContext(ctx, "admin_profit_sharing_share_failed",
			"level", "warn",
			"profit_sharing_id", rec.ID,
			"actor_user_id", actorID,
			"impersonating", impersonating,
			"impersonator_user_id", impersonator,
			"impersonated_user_id", actorID,
			"impersonation_session_id", sessionID,
			"trace_id", tracelog.TraceIDFromContext(ctx),
			"error", err.Error(),
		)
		code, msg := profitSharingActionClientError(err)
		return code, "", false, msg
	}
	_, _ = db.Exec(`UPDATE billing_profit_sharing_admin_action SET outcome = 'succeeded', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), actionID)
	slog.InfoContext(ctx, "admin_profit_sharing_share_ok",
		"level", "info",
		"profit_sharing_id", rec.ID,
		"actor_user_id", actorID,
		"impersonating", impersonating,
		"impersonator_user_id", impersonator,
		"impersonated_user_id", actorID,
		"impersonation_session_id", sessionID,
		"out_profit_sharing_no", rec.OutProfitSharingNo,
	)
	return http.StatusOK, "shared", false, ""
}

type adminShareActionRow struct {
	ID              int64
	ProfitSharingID int64
	Outcome         string
}

func loadAdminShareByIdempotency(key string) (*adminShareActionRow, bool) {
	var row adminShareActionRow
	err := db.QueryRow(`
		SELECT id, profit_sharing_id, outcome
		FROM billing_profit_sharing_admin_action WHERE idempotency_key = ?`, key).Scan(
		&row.ID, &row.ProfitSharingID, &row.Outcome)
	if err != nil {
		return nil, false
	}
	return &row, true
}

func retryAdminShareAfterFailedIdempotency(r *http.Request, existing *adminShareActionRow, actorID, impersonator, sessionID string, impersonating bool) (int, string, bool, string) {
	ctx := r.Context()
	var rec profitSharingRecord
	err := db.QueryRow(`
		SELECT ps.id, ps.out_profit_sharing_no, ps.order_id, ps.order_number, ps.tenant_id,
		       ps.referrer_user_id, ps.referrer_openid,
		       ps.commission_yuan_cents, ps.total_yuan_cents,
		       ps.status, COALESCE(ps.fail_reason, '')
		FROM billing_profit_sharing ps
		WHERE ps.id = ?`, existing.ProfitSharingID).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid,
		&rec.CommissionYuanCents, &rec.TotalYuanCents,
		&rec.Status, &rec.FailReason,
	)
	if err != nil {
		return http.StatusInternalServerError, "", false, "加载分账记录失败"
	}
	if rec.Status == psStatusFinished {
		return http.StatusOK, "shared", true, ""
	}
	if rec.Status == psStatusProcessing {
		return http.StatusOK, "processing", true, ""
	}
	if err := executeProfitSharing(ctx, rec); err != nil {
		markProfitSharingStatus(ctx, rec.ID, psStatusFailed, err.Error())
		code, msg := profitSharingActionClientError(err)
		return code, "", false, msg
	}
	_, _ = db.Exec(`UPDATE billing_profit_sharing_admin_action SET outcome = 'succeeded', updated_at = ? WHERE id = ?`,
		time.Now().UTC(), existing.ID)
	slog.InfoContext(ctx, "admin_profit_sharing_share_retry_ok",
		"level", "info",
		"profit_sharing_id", rec.ID,
		"actor_user_id", actorID,
		"impersonating", impersonating,
		"impersonator_user_id", impersonator,
		"impersonated_user_id", actorID,
		"impersonation_session_id", sessionID,
	)
	return http.StatusOK, "shared", false, ""
}
