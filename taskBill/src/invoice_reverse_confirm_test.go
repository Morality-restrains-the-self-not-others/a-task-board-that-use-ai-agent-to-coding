package main

import (
	"testing"
	"time"
)

func TestInvoiceJSONReversePendingHas72hDeadline(t *testing.T) {
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	created := createdAt.Format("2006-01-02 15:04:05.000000")
	wantDeadline := createdAt.Add(time.Duration(invoiceReverseConfirmHours) * time.Hour).Format(time.RFC3339)
	inv := Invoice{
		ID: 1, TenantID: 2, OrderID: 3, ApplicationID: 4,
		Kind: invoiceKindRed, Purpose: invoicePurposeReverse,
		Status: invoiceStatusReversePending, CreatedAt: created, UpdatedAt: created,
	}
	m := invoiceJSON(&inv)
	if m["status"] != invoiceStatusReversePending {
		t.Fatalf("status=%v", m["status"])
	}
	if m["reverse_confirm_hours"] != invoiceReverseConfirmHours {
		t.Fatalf("hours=%v", m["reverse_confirm_hours"])
	}
	if m["reverse_confirm_deadline"] != wantDeadline {
		t.Fatalf("deadline=%v want %s", m["reverse_confirm_deadline"], wantDeadline)
	}
	if m["reverse_confirm_expired"] != false {
		t.Fatalf("expired=%v", m["reverse_confirm_expired"])
	}
}

func TestInvoiceJSONReversePendingExpiredOverlaysStatus(t *testing.T) {
	createdAt := time.Now().UTC().Add(-time.Duration(invoiceReverseConfirmHours+24) * time.Hour).Truncate(time.Second)
	created := createdAt.Format("2006-01-02 15:04:05.000000")
	wantDeadline := createdAt.Add(time.Duration(invoiceReverseConfirmHours) * time.Hour).Format(time.RFC3339)
	inv := Invoice{
		ID: 1, TenantID: 2, OrderID: 3, ApplicationID: 4,
		Kind: invoiceKindRed, Purpose: invoicePurposeReverse,
		Status: invoiceStatusReversePending, CreatedAt: created, UpdatedAt: created,
	}
	m := invoiceJSON(&inv)
	if m["status"] != invoiceStatusReverseExpired {
		t.Fatalf("status=%v want %s", m["status"], invoiceStatusReverseExpired)
	}
	if m["reverse_confirm_expired"] != true {
		t.Fatalf("expired=%v", m["reverse_confirm_expired"])
	}
	if m["reverse_confirm_deadline"] != wantDeadline {
		t.Fatalf("deadline=%v want %s", m["reverse_confirm_deadline"], wantDeadline)
	}
}

func TestInvoiceJSONIssuedHasNoReverseConfirm(t *testing.T) {
	inv := Invoice{
		ID: 1, TenantID: 2, OrderID: 3, ApplicationID: 4,
		Kind: invoiceKindBlue, Purpose: invoicePurposeOriginal,
		Status: invoiceStatusIssued, CreatedAt: utcNow(), UpdatedAt: utcNow(),
	}
	m := invoiceJSON(&inv)
	if _, ok := m["reverse_confirm_deadline"]; ok {
		t.Fatalf("unexpected deadline on issued: %v", m)
	}
}

func TestOrderReverseConfirmFromInvoicesPrefersRed(t *testing.T) {
	createdAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	created := createdAt.Format("2006-01-02 15:04:05.000000")
	wantDeadline := createdAt.Add(time.Duration(invoiceReverseConfirmHours) * time.Hour).Format(time.RFC3339)
	red := invoiceJSON(&Invoice{
		ID: 9, TenantID: 2, OrderID: 3, ApplicationID: 4,
		Kind: invoiceKindRed, Purpose: invoicePurposeReverse,
		Status: invoiceStatusReversePending, CreatedAt: created, UpdatedAt: created,
	})
	conf := orderReverseConfirmFromInvoices([]map[string]interface{}{red})
	if conf == nil {
		t.Fatal("expected confirm object")
	}
	if conf["hours"] != invoiceReverseConfirmHours {
		t.Fatalf("hours=%v", conf["hours"])
	}
	if conf["required"] != true {
		t.Fatalf("required=%v", conf["required"])
	}
	if conf["message"] != invoiceReverseConfirmHint {
		t.Fatalf("message=%v", conf["message"])
	}
	if conf["deadline"] != wantDeadline {
		t.Fatalf("deadline=%v want %s", conf["deadline"], wantDeadline)
	}
}
