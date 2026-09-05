package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func handleWechatFapiaoNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"code": "FAIL", "message": "method"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "bad body"})
		return
	}
	if !verifyWechatNotification(r.Header, body) {
		slog.WarnContext(r.Context(), "wechat_fapiao_notify_bad_signature", "level", "warn")
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "FAIL", "message": "invalid signature"})
		return
	}
	decrypted, err := decryptWechatNotify(body)
	if err != nil {
		slog.ErrorContext(r.Context(), "wechat_fapiao_notify_decrypt_failed",
			"level", "error",
			"error", err.Error(),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "FAIL", "message": "decrypt failed"})
		return
	}
	if err := applyWechatFapiaoNotify(r.Context(), decrypted); err != nil {
		slog.ErrorContext(r.Context(), "wechat_fapiao_notify_apply_failed",
			"level", "error",
			"error", err.Error(),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"code": "FAIL", "message": "apply failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": "SUCCESS", "message": "成功"})
}

func applyWechatFapiaoNotify(ctx context.Context, decrypted []byte) error {
	eventType, applyID, fapiaoID, fapiaoNumber, lineStatus := parseFapiaoNotifyResource(decrypted)
	slog.InfoContext(ctx, "wechat_fapiao_notify",
		"level", "info",
		"event_type", eventType,
		"fapiao_id", fapiaoID,
		"line_status", lineStatus,
	)
	now := time.Now().UTC().Format(time.RFC3339)
	et := strings.ToUpper(eventType)
	switch {
	case strings.Contains(et, "ISSUED") && !strings.Contains(et, "FAIL") && !strings.Contains(et, "REVERSE"):
		return markFapiaoIssued(ctx, applyID, fapiaoID, fapiaoNumber, now)
	case strings.Contains(et, "REVERSE"):
		return markFapiaoReversed(ctx, applyID, fapiaoID, now)
	case strings.Contains(et, "FAIL"):
		return markFapiaoFailed(ctx, applyID, fapiaoID, lineStatus, now)
	default:
		if strings.EqualFold(lineStatus, "ISSUED") {
			return markFapiaoIssued(ctx, applyID, fapiaoID, fapiaoNumber, now)
		}
		return nil
	}
}

func markFapiaoIssued(ctx context.Context, applyID, fapiaoID, fapiaoNumber, now string) error {
	q := `UPDATE billing_invoice SET status = ?, wechat_fapiao_number = CASE WHEN ? = '' THEN wechat_fapiao_number ELSE ? END, updated_at = ? WHERE status IN (?, ?) `
	args := []interface{}{invoiceStatusIssued, fapiaoNumber, fapiaoNumber, now, invoiceStatusIssuing, invoiceStatusIssued}
	if fapiaoID != "" {
		q += ` AND fapiao_id = ?`
		args = append(args, fapiaoID)
	} else if applyID != "" {
		q += ` AND wechat_apply_id = ?`
		args = append(args, applyID)
	} else {
		return nil
	}
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		go djangoEmitEvent(ctx, "BILLING_INVOICE_ISSUED", map[string]interface{}{
			"fapiao_id":       fapiaoID,
			"wechat_apply_id": applyID,
		})
	}
	return nil
}

func markFapiaoReversed(ctx context.Context, applyID, fapiaoID, now string) error {
	if applyID == "" && fapiaoID == "" {
		return nil
	}
	where := ""
	var match interface{}
	if applyID != "" {
		where = `wechat_apply_id = ?`
		match = applyID
	} else {
		where = `fapiao_id = ?`
		match = fapiaoID
	}
	res, err := db.ExecContext(ctx, `
		UPDATE billing_invoice SET status = ?, updated_at = ?
		WHERE kind = ? AND status <> ? AND `+where,
		invoiceStatusReversed, now, invoiceKindBlue, invoiceStatusReversed, match)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE billing_invoice SET status = ?, updated_at = ?
		WHERE kind = ? AND status = ? AND `+where,
		invoiceStatusIssued, now, invoiceKindRed, invoiceStatusReversePending, match); err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		slog.InfoContext(ctx, "invoice_reversed",
			"level", "info",
			"wechat_apply_id", applyID,
			"fapiao_id", fapiaoID,
		)
		go djangoEmitEvent(ctx, "BILLING_INVOICE_REVERSED", map[string]interface{}{
			"wechat_apply_id": applyID,
			"fapiao_id":       fapiaoID,
		})
	}
	return nil
}

func markFapiaoFailed(ctx context.Context, applyID, fapiaoID, reason, now string) error {
	q := `UPDATE billing_invoice SET status = ?, fail_reason = ?, updated_at = ? WHERE status = ? `
	args := []interface{}{invoiceStatusFailed, truncateRunes(reason, 500), now, invoiceStatusIssuing}
	if fapiaoID != "" {
		q += ` AND fapiao_id = ?`
		args = append(args, fapiaoID)
	} else if applyID != "" {
		q += ` AND wechat_apply_id = ?`
		args = append(args, applyID)
	} else {
		return nil
	}
	_, err := db.ExecContext(ctx, q, args...)
	return err
}
