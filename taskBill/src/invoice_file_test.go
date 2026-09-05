package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func setupInvoiceFileMedia(t *testing.T) {
	t.Helper()
	initInvoiceFileMediaRoot(t.TempDir())
}

func specialOrgBuyer() invoiceBuyerInput {
	return invoiceBuyerInput{
		InvoiceType: invoiceTypeSpecial,
		Type:        buyerTypeOrganization,
		Name:        "测试科技有限公司",
		TaxpayerID:  "91310000MA1FL4XH6A",
		Address:     "上海市浦东新区测试路1号",
		Telephone:   "021-12345678",
		BankName:    "招商银行上海分行",
		BankAccount: "6225880212345678",
	}
}

func uploadInvoiceFile(t *testing.T, mux *http.ServeMux, appID int64, content []byte, filename string, wantStatus int) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write form: %v", err)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/system-admin/invoice-applications/%s/invoice-file/", formatID(appID)), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("upload status=%d want %d body=%s", rec.Code, wantStatus, rec.Body.String())
	}
}

func TestParseInvoiceBuyerSpecialRequiresFullOrgInfo(t *testing.T) {
	// 专票 + 个人 → 拒绝
	if _, err := parseInvoiceBuyer(map[string]interface{}{
		"invoice_type": "special", "type": "INDIVIDUAL", "name": "张三",
	}); !errors.Is(err, ErrInvoiceSpecialInfoRequired) {
		t.Fatalf("special+individual err=%v want ErrInvoiceSpecialInfoRequired", err)
	}
	// 专票 + 企业缺任一单位信息 → 拒绝
	base := map[string]interface{}{
		"invoice_type": "special", "type": "ORGANIZATION", "name": "测试科技",
		"taxpayer_id": "91310000MA1FL4XH6A",
	}
	full := map[string]interface{}{
		"address": "上海市浦东新区测试路1号", "telephone": "021-12345678",
		"bank_name": "招商银行上海分行", "bank_account": "6225880212345678",
	}
	// 只保留 base + 单个单位字段 → 必然缺其余三个
	for k, v := range full {
		c := map[string]interface{}{}
		for kk, vv := range base {
			c[kk] = vv
		}
		c[k] = v
		if _, err := parseInvoiceBuyer(c); !errors.Is(err, ErrInvoiceSpecialInfoRequired) {
			t.Fatalf("special missing %s: err=%v want ErrInvoiceSpecialInfoRequired", k, err)
		}
	}
	// 完整专票 → 通过
	in, err := parseInvoiceBuyer(map[string]interface{}{
		"invoice_type": "special", "type": "ORGANIZATION", "name": "测试科技有限公司",
		"taxpayer_id": "91310000MA1FL4XH6A", "address": "上海市浦东新区测试路1号",
		"telephone": "021-12345678", "bank_name": "招商银行上海分行",
		"bank_account": "6225880212345678",
	})
	if err != nil {
		t.Fatalf("special full: %v", err)
	}
	if in.InvoiceType != invoiceTypeSpecial {
		t.Fatalf("type=%s want special", in.InvoiceType)
	}
	// 未传 invoice_type → 默认普票
	in2, err := parseInvoiceBuyer(map[string]interface{}{"type": "INDIVIDUAL", "name": "张三"})
	if err != nil {
		t.Fatal(err)
	}
	if in2.InvoiceType != invoiceTypeGeneral {
		t.Fatalf("default type=%s want general", in2.InvoiceType)
	}
	// 非法类型 → 拒绝
	if _, err := parseInvoiceBuyer(map[string]interface{}{
		"invoice_type": "e-invoice", "type": "INDIVIDUAL", "name": "张三",
	}); !errors.Is(err, ErrInvoiceBuyerInvalid) {
		t.Fatalf("err=%v want ErrInvoiceBuyerInvalid", err)
	}
	// 普票企业仍要求税号
	if _, err := parseInvoiceBuyer(map[string]interface{}{
		"type": "ORGANIZATION", "name": "测试科技",
	}); !errors.Is(err, ErrInvoiceOrgTaxIDRequired) {
		t.Fatalf("err=%v want ErrInvoiceOrgTaxIDRequired", err)
	}
}

func TestInvoiceFileUploadApproveAndDownload(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	setupInvoiceFileMedia(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 88, "420000INVF")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", specialOrgBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	pdf := []byte("%PDF-1.4 fake invoice content for test\n%%EOF\n")
	uploadInvoiceFile(t, mux, app.ID, pdf, "invoice.pdf", http.StatusOK)

	// 上传已持久化
	app2, err := loadInvoiceApplication(context.Background(), db, app.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if app2.InvoiceFilePath == "" {
		t.Fatal("invoice_file_path not persisted")
	}
	// 重复上传覆盖旧文件并清理磁盘旧文件
	first := app2.InvoiceFilePath
	uploadInvoiceFile(t, mux, app.ID, []byte("%PDF-1.4 second fake invoice\n%%EOF\n"), "invoice2.pdf", http.StatusOK)
	app3, _ := loadInvoiceApplication(context.Background(), db, app.ID)
	if app3.InvoiceFilePath == first {
		t.Fatal("expected replacement path")
	}
	oldAbs, err := invoiceFileAbsPath(first)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(oldAbs); !os.IsNotExist(err) {
		t.Fatalf("old file not cleaned: %v", err)
	}

	// 管理员下载 pending 发票文件
	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/system-admin/invoice-applications/%s/invoice-file/", formatID(app.ID)), nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin download status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("body=%q", rec.Body.String())
	}

	// 审批登记：invoice_file_path 复制到发票行
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "已手动开具", ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	var invID int64
	var gotPath string
	if err := db.QueryRow(`
		SELECT id, invoice_file_path FROM billing_invoice WHERE application_id = ?`, app.ID).Scan(&invID, &gotPath); err != nil {
		t.Fatalf("scan invoice: %v", err)
	}
	if gotPath != app3.InvoiceFilePath {
		t.Fatalf("invoice file=%q want %q", gotPath, app3.InvoiceFilePath)
	}

	fileURL := fmt.Sprintf("/api/tenant/%s/billing/orders/%s/invoices/%s/file/",
		formatID(tenantID), formatID(orderID), formatID(invID))
	// 无权限 → 403
	req2 := httptest.NewRequest(http.MethodGet, fileURL, nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("no-perm status=%d want 403 body=%s", rec2.Code, rec2.Body.String())
	}
	// 租户 billing:view → 200
	req3 := httptest.NewRequest(http.MethodGet, fileURL, nil)
	req3.Header.Set("X-Tenant-Perms", formatID(tenantID)+":billing:view")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("with-perm status=%d body=%s", rec3.Code, rec3.Body.String())
	}
	if !bytes.HasPrefix(rec3.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("with-perm body=%q", rec3.Body.String())
	}
	// 平台员工直通 → 200
	req4 := httptest.NewRequest(http.MethodGet, fileURL, nil)
	req4.Header.Set("X-User-Roles", "super_admin")
	rec4 := httptest.NewRecorder()
	mux.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK {
		t.Fatalf("staff status=%d body=%s", rec4.Code, rec4.Body.String())
	}

	// 发票 JSON 暴露 invoice_file_url
	invoices, err := listInvoicesForOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(invoices) != 1 || invoices[0]["invoice_file_url"] == nil {
		t.Fatalf("invoices=%v want invoice_file_url", invoices)
	}
}

func TestInvoiceFileUploadForbiddenForMember(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "a.pdf")
	_, _ = fw.Write([]byte("%PDF-1.4\n"))
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/invoice-applications/1/invoice-file/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

func TestInvoiceFileUploadRequiresPending(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	setupInvoiceFileMedia(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 66, "420000INVG")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", specialOrgBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := approveInvoiceApplication(context.Background(), app.ID, "admin-1", "ok", ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	uploadInvoiceFile(t, mux, app.ID, []byte("%PDF-1.4\n"), "x.pdf", http.StatusBadRequest)
}

func TestInvoiceFileRejectRejectsUnsupportedType(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	setupInvoiceFileMedia(t)
	tenantID := generateSnowflakeID()
	orderID := generateSnowflakeID()
	insertPaidWechatOrderForInvoice(t, tenantID, orderID, 66, "420000INVH")
	app, err := applyInvoiceApplication(context.Background(), tenantID, orderID, "u1", specialOrgBuyer(), "")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	uploadInvoiceFile(t, mux, app.ID, []byte("#!/bin/sh\necho pwned\n"), "evil.sh", http.StatusBadRequest)
}
