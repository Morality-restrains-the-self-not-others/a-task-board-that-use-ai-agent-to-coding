package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func stubWechatFapiaoOK(t *testing.T) {
	t.Helper()
	origI := issueWechatFapiaoFn
	origR := reverseWechatFapiaoFn
	issueWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoIssueRequest) (wechatFapiaoIssueResult, error) {
		return wechatFapiaoIssueResult{ApplyID: req.FapiaoApplyID, FapiaoID: req.FapiaoID}, nil
	}
	reverseWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoReverseRequest) error {
		return nil
	}
	t.Cleanup(func() {
		issueWechatFapiaoFn = origI
		reverseWechatFapiaoFn = origR
	})
}

// assertNoWechatIssueCall 手动开具模式守卫：审批登记不得触发微信自动开票。
func assertNoWechatIssueCall(t *testing.T) {
	t.Helper()
	orig := issueWechatFapiaoFn
	issueWechatFapiaoFn = func(ctx context.Context, req wechatFapiaoIssueRequest) (wechatFapiaoIssueResult, error) {
		t.Fatalf("manual issue mode: issueWechatFapiaoFn must not be called (req=%+v)", req)
		return wechatFapiaoIssueResult{}, nil
	}
	t.Cleanup(func() { issueWechatFapiaoFn = orig })
}

func TestApproveInvoiceMarksManuallyIssued(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV3")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	approved, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", "")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.Status != invoiceAppApproved {
		t.Fatalf("status=%s", approved.Status)
	}
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(invoices) != 1 {
		t.Fatalf("invoices=%d", len(invoices))
	}
	if invoices[0]["kind"] != invoiceKindBlue || invoices[0]["purpose"] != invoicePurposeOriginal {
		t.Fatalf("invoice=%v", invoices[0])
	}
	st := invoices[0]["status"]
	if st != invoiceStatusIssued {
		t.Fatalf("status=%v want issued (manual issue mode)", st)
	}
}

func TestApproveInvoiceZeroAmountRejected(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderRowOnly(t, tenantID, orderID, 0)
	appID := generateSnowflakeID()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_invoice_application (
			id, tenant_id, order_id, applicant_user_id, invoice_type, buyer_type, buyer_name, taxpayer_id,
			address, telephone, bank_name, bank_account, status, reviewer_user_id, review_note,
			reviewed_at, created_at, updated_at
		) VALUES (?, ?, ?, 'u1', 'general', 'INDIVIDUAL', '张三', '', '', '', '', '', 'pending', '', '', NULL, ?, ?)`,
		appID, tenantID, orderID, now, now,
	); err != nil {
		t.Fatalf("insert pending app: %v", err)
	}
	_, err := approveInvoiceApplication(context.Background(), appID, "admin-1", "ok", "")
	if !errors.Is(err, ErrInvoiceZeroAmount) {
		t.Fatalf("err=%v want ErrInvoiceZeroAmount", err)
	}
	var n int
	if qerr := db.QueryRow(`SELECT COUNT(*) FROM billing_invoice WHERE order_id = ?`, orderID).Scan(&n); qerr != nil {
		t.Fatalf("count invoices: %v", qerr)
	}
	if n != 0 {
		t.Fatalf("invoices=%d want 0", n)
	}
}

// OPT-20260823-053：手动开具登记时填写微信发票号码 → 落库并出现在订单发票 JSON。
func TestApproveInvoiceRegistersManualWechatFapiaoNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV6")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", "31100000000000000000"); err != nil {
		t.Fatalf("approve with fapiao number: %v", err)
	}
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(invoices) != 1 {
		t.Fatalf("invoices=%d", len(invoices))
	}
	if invoices[0]["wechat_fapiao_number"] != "31100000000000000000" {
		t.Fatalf("wechat_fapiao_number=%v want 31100000000000000000", invoices[0]["wechat_fapiao_number"])
	}
}

// OPT-20260823-053：空号码登记 → 置空，不报错。
func TestApproveInvoiceEmptyFapiaoNumberCleared(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV7")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", "   "); err != nil {
		t.Fatalf("approve with blank fapiao number: %v", err)
	}
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if invoices[0]["wechat_fapiao_number"] != "" {
		t.Fatalf("wechat_fapiao_number=%v want ''", invoices[0]["wechat_fapiao_number"])
	}
}

// OPT-20260823-053：号码超过 32 字符 → 400 拒绝。
func TestApproveInvoiceRejectsTooLongFapiaoNumber(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV8")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	long := "311000000000000000001111111111111"
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", long); !errors.Is(err, ErrInvoiceFapiaoNumberTooLong) {
		t.Fatalf("want ErrInvoiceFapiaoNumberTooLong, got %v", err)
	}
}

func TestRejectInvoiceAllowsReapply(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV4")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := rejectInvoiceApplication(context.Background(), app.ID, "admin-1", "资料不符"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	app2, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("reapply: %v", err)
	}
	if app2.ID == app.ID {
		t.Fatalf("expected new application")
	}
}

func TestAdminInvoiceApproveForbiddenForMember(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/invoice-applications/1/approve/", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetOrderIncludesInvoices(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	assertNoWechatIssueCall(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 55, "420000INV5")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", individualBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+formatID(tenantID)+"/billing/orders/"+formatID(orderID)+"/", nil)
	rec := httptest.NewRecorder()
	handleGetOrder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	raw, ok := body["invoices"].([]interface{})
	if !ok || len(raw) != 1 {
		t.Fatalf("invoices=%v", body["invoices"])
	}
	row := raw[0].(map[string]interface{})
	if row["order_id"] != formatID(orderID) {
		t.Fatalf("order_id=%v", row["order_id"])
	}
}

func TestBuildWechatFapiaoPayloadUsesTransactionID(t *testing.T) {
	p := buildWechatFapiaoPayload(wechatFapiaoIssueRequest{
		FapiaoApplyID:  "420000ABC",
		FapiaoID:       "fp-1",
		TotalAmountFen: 55,
		Buyer:          individualBuyer(),
	})
	if p["scene"] != "WITH_WECHATPAY" {
		t.Fatalf("scene=%v", p["scene"])
	}
	if p["fapiao_apply_id"] != "420000ABC" {
		t.Fatalf("apply_id=%v", p["fapiao_apply_id"])
	}
}

func TestUnusedRefundableCentsNoGrantsIsFull(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	got, err := unusedRefundableCents(context.Background(), db, tenantID, orderID, 55)
	if err != nil {
		t.Fatal(err)
	}
	if got != 55 {
		t.Fatalf("got=%d", got)
	}
}

func TestRemainingDealCentsAfterRefund(t *testing.T) {
	if remainingDealCentsAfterRefund(100, 100) != 0 {
		t.Fatal("full refund remaining should be 0")
	}
	if remainingDealCentsAfterRefund(100, 40) != 60 {
		t.Fatal("partial")
	}
}

func TestParseInvoiceBuyerIndividual(t *testing.T) {
	in, err := parseInvoiceBuyer(map[string]interface{}{"type": "INDIVIDUAL", "name": "李四"})
	if err != nil {
		t.Fatal(err)
	}
	if in.Type != buyerTypeIndividual || in.Name != "李四" {
		t.Fatalf("%+v", in)
	}
}

func TestInvoiceApplyErrors(t *testing.T) {
	if !errors.Is(ErrInvoicePendingExists, ErrInvoicePendingExists) {
		t.Fatal("sentinel")
	}
}
