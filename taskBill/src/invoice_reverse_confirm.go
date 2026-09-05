package main

import (
	"strings"
	"time"
)

const (
	invoiceReverseConfirmHint        = "请在微信卡包于72小时内确认冲红，逾期冲红将失效"
	invoiceReverseConfirmExpiredHint = "冲红确认已超过72小时，冲红可能已失效，请联系平台处理"
)

func parseInvoiceTimeUTC(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func reverseConfirmAnchorTime(inv *Invoice) string {
	if inv == nil {
		return ""
	}
	if inv.Kind == invoiceKindRed {
		return inv.CreatedAt
	}
	if inv.UpdatedAt != "" {
		return inv.UpdatedAt
	}
	return inv.CreatedAt
}

func decorateInvoiceReverseConfirm(m map[string]interface{}, inv *Invoice) {
	if inv == nil || inv.Status != invoiceStatusReversePending {
		return
	}
	anchor, ok := parseInvoiceTimeUTC(reverseConfirmAnchorTime(inv))
	if !ok {
		return
	}
	deadline := anchor.Add(time.Duration(invoiceReverseConfirmHours) * time.Hour)
	expired := time.Now().UTC().After(deadline)
	m["reverse_confirm_hours"] = invoiceReverseConfirmHours
	m["reverse_confirm_deadline"] = deadline.Format(time.RFC3339)
	m["reverse_confirm_expired"] = expired
	if expired {
		m["status"] = invoiceStatusReverseExpired
	}
}

func orderReverseConfirmFromInvoices(invoices []map[string]interface{}) map[string]interface{} {
	var picked map[string]interface{}
	for _, inv := range invoices {
		st, _ := inv["status"].(string)
		if st != invoiceStatusReversePending && st != invoiceStatusReverseExpired {
			continue
		}
		if _, ok := inv["reverse_confirm_deadline"]; !ok {
			continue
		}
		picked = inv
		if inv["kind"] == invoiceKindRed {
			break
		}
	}
	if picked == nil {
		return nil
	}
	expired, _ := picked["reverse_confirm_expired"].(bool)
	msg := invoiceReverseConfirmHint
	if expired || picked["status"] == invoiceStatusReverseExpired {
		msg = invoiceReverseConfirmExpiredHint
		expired = true
	}
	return map[string]interface{}{
		"required": true,
		"hours":    picked["reverse_confirm_hours"],
		"deadline": picked["reverse_confirm_deadline"],
		"expired":  expired,
		"message":  msg,
	}
}
