package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	invoiceAppPending   = "pending"
	invoiceAppApproved  = "approved"
	invoiceAppRejected  = "rejected"
	invoiceAppCancelled = "cancelled"

	invoiceKindBlue = "blue"
	invoiceKindRed  = "red"

	invoicePurposeOriginal = "original"
	invoicePurposeReverse  = "reverse"
	invoicePurposeReissue  = "reissue"

	invoiceStatusIssuing        = "issuing"
	invoiceStatusIssued         = "issued"
	invoiceStatusReversePending = "reverse_pending"
	invoiceStatusReversed       = "reversed"
	invoiceStatusReverseExpired = "reverse_expired"
	invoiceStatusFailed         = "failed"

	invoiceReverseConfirmHours = 72

	buyerTypeIndividual   = "INDIVIDUAL"
	buyerTypeOrganization = "ORGANIZATION"
)

var (
	ErrInvoiceOrderNotPaid        = errors.New("仅已支付的微信支付订单可申请开票")
	ErrInvoiceZeroAmount          = errors.New("订单金额为 0 元，无法申请开票")
	ErrInvoicePendingExists       = errors.New("该订单已有待审批的开票申请")
	ErrInvoiceAlreadyIssued       = errors.New("该订单已有有效蓝字发票")
	ErrInvoiceBuyerInvalid        = errors.New("发票抬头不完整")
	ErrInvoiceOrgTaxIDRequired    = errors.New("单位抬头须填写纳税人识别号")
	ErrInvoiceSpecialInfoRequired = errors.New("专票须填写单位全称、纳税人识别号、注册地址、电话、开户行与银行账号")
	ErrInvoiceApplicationNotFound = errors.New("开票申请不存在")
	ErrInvoiceApplicationNotPend  = errors.New("开票申请不是待审批状态")
)

const (
	invoiceTypeGeneral = "general" // 普票
	invoiceTypeSpecial = "special" // 专票
)

type InvoiceApplication struct {
	ID              int64
	TenantID        int64
	OrderID         int64
	ApplicantUserID string
	InvoiceType     string
	BuyerType       string
	BuyerName       string
	TaxpayerID      string
	Address         string
	Telephone       string
	BankName        string
	BankAccount     string
	InvoiceFilePath string
	Status          string
	ReviewerUserID  string
	ReviewNote      string
	ReviewedAt      sql.NullString
	CreatedAt       string
	UpdatedAt       string
}

type Invoice struct {
	ID                 int64
	TenantID           int64
	OrderID            int64
	ApplicationID      int64
	RelatedInvoiceID   sql.NullInt64
	Kind               string
	Purpose            string
	Status             string
	AmountYuanCents    int64
	FapiaoID           string
	WechatApplyID      string
	WechatFapiaoNumber string
	InvoiceFilePath    string
	BuyerSnapshot      string
	FailReason         string
	CreatedAt          string
	UpdatedAt          string
}

type invoiceBuyerInput struct {
	InvoiceType string
	Type        string
	Name        string
	TaxpayerID  string
	Address     string
	Telephone   string
	BankName    string
	BankAccount string
}

func (b invoiceBuyerInput) snapshotJSON() string {
	raw, _ := json.Marshal(map[string]string{
		"invoice_type": b.InvoiceType,
		"type":         b.Type,
		"name":         b.Name,
		"taxpayer_id":  b.TaxpayerID,
		"address":      b.Address,
		"telephone":    b.Telephone,
		"bank_name":    b.BankName,
		"bank_account": b.BankAccount,
	})
	return string(raw)
}

func parseInvoiceBuyer(body map[string]interface{}) (invoiceBuyerInput, error) {
	buyer, _ := body["buyer"].(map[string]interface{})
	if buyer == nil {
		buyer = body
	}
	in := invoiceBuyerInput{
		InvoiceType: strings.ToLower(strings.TrimSpace(stringField(buyer, "invoice_type"))),
		Type:        strings.ToUpper(strings.TrimSpace(stringField(buyer, "type"))),
		Name:        strings.TrimSpace(stringField(buyer, "name")),
		TaxpayerID:  strings.TrimSpace(stringField(buyer, "taxpayer_id")),
		Address:     strings.TrimSpace(stringField(buyer, "address")),
		Telephone:   strings.TrimSpace(stringField(buyer, "telephone")),
		BankName:    strings.TrimSpace(stringField(buyer, "bank_name")),
		BankAccount: strings.TrimSpace(stringField(buyer, "bank_account")),
	}
	if in.InvoiceType == "" {
		in.InvoiceType = invoiceTypeGeneral
	}
	if in.InvoiceType != invoiceTypeGeneral && in.InvoiceType != invoiceTypeSpecial {
		return in, ErrInvoiceBuyerInvalid
	}
	if in.Type != buyerTypeIndividual && in.Type != buyerTypeOrganization {
		return in, ErrInvoiceBuyerInvalid
	}
	if in.Name == "" || len(in.Name) > 256 {
		return in, ErrInvoiceBuyerInvalid
	}
	if in.Type == buyerTypeOrganization && in.TaxpayerID == "" {
		return in, ErrInvoiceOrgTaxIDRequired
	}
	// 专票仅对企业开具，且须完整单位开票信息（名称/税号/地址/电话/开户行/账号）。
	if in.InvoiceType == invoiceTypeSpecial &&
		(in.Type != buyerTypeOrganization || in.Address == "" || in.Telephone == "" || in.BankName == "" || in.BankAccount == "") {
		return in, ErrInvoiceSpecialInfoRequired
	}
	return in, nil
}

func invoiceApplicationJSON(app *InvoiceApplication) map[string]interface{} {
	m := map[string]interface{}{
		"id":                formatID(app.ID),
		"tenant_id":         formatID(app.TenantID),
		"order_id":          formatID(app.OrderID),
		"applicant_user_id": app.ApplicantUserID,
		"invoice_type":      app.InvoiceType,
		"buyer_type":        app.BuyerType,
		"buyer_name":        app.BuyerName,
		"status":            app.Status,
		"reviewer_user_id":  app.ReviewerUserID,
		"review_note":       app.ReviewNote,
		"created_at":        app.CreatedAt,
		"updated_at":        app.UpdatedAt,
	}
	if app.TaxpayerID != "" {
		m["taxpayer_id"] = app.TaxpayerID
	}
	if app.Address != "" {
		m["address"] = app.Address
	}
	if app.Telephone != "" {
		m["telephone"] = app.Telephone
	}
	if app.BankName != "" {
		m["bank_name"] = app.BankName
	}
	if app.BankAccount != "" {
		m["bank_account"] = app.BankAccount
	}
	if app.InvoiceFilePath != "" {
		m["invoice_file_path"] = app.InvoiceFilePath
	}
	if app.ReviewedAt.Valid {
		m["reviewed_at"] = app.ReviewedAt.String
	}
	return m
}

func invoiceJSON(inv *Invoice) map[string]interface{} {
	m := map[string]interface{}{
		"id":                   formatID(inv.ID),
		"tenant_id":            formatID(inv.TenantID),
		"order_id":             formatID(inv.OrderID),
		"application_id":       formatID(inv.ApplicationID),
		"kind":                 inv.Kind,
		"purpose":              inv.Purpose,
		"status":               inv.Status,
		"amount_yuan":          centsToYuanStr(inv.AmountYuanCents),
		"amount_yuan_cents":    inv.AmountYuanCents,
		"fapiao_id":            inv.FapiaoID,
		"wechat_apply_id":      inv.WechatApplyID,
		"wechat_fapiao_number": inv.WechatFapiaoNumber,
		"created_at":           inv.CreatedAt,
		"updated_at":           inv.UpdatedAt,
	}
	if inv.InvoiceFilePath != "" {
		m["invoice_file_url"] = fmt.Sprintf(
			"/api/tenant/%s/billing/orders/%s/invoices/%s/file/",
			formatID(inv.TenantID), formatID(inv.OrderID), formatID(inv.ID))
	}
	if t := snapshotInvoiceType(inv.BuyerSnapshot); t != "" {
		m["invoice_type"] = t
	}
	if inv.RelatedInvoiceID.Valid {
		m["related_invoice_id"] = formatID(inv.RelatedInvoiceID.Int64)
	}
	if inv.FailReason != "" {
		m["fail_reason"] = inv.FailReason
	}
	decorateInvoiceReverseConfirm(m, inv)
	return m
}

func scanInvoiceApplication(scanner interface {
	Scan(dest ...any) error
}) (*InvoiceApplication, error) {
	var app InvoiceApplication
	err := scanner.Scan(
		&app.ID, &app.TenantID, &app.OrderID, &app.ApplicantUserID,
		&app.InvoiceType, &app.BuyerType, &app.BuyerName, &app.TaxpayerID, &app.Address,
		&app.Telephone, &app.BankName, &app.BankAccount, &app.InvoiceFilePath, &app.Status,
		&app.ReviewerUserID, &app.ReviewNote, &app.ReviewedAt,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

const invoiceAppSelectCols = `id, tenant_id, order_id, applicant_user_id,
		invoice_type, buyer_type, buyer_name, taxpayer_id, address, telephone, bank_name, bank_account,
		invoice_file_path, status, reviewer_user_id, review_note, reviewed_at, created_at, updated_at`

func loadInvoiceApplication(ctx context.Context, exec sqlExecutor, appID int64) (*InvoiceApplication, error) {
	row := exec.QueryRowContext(ctx, `SELECT `+invoiceAppSelectCols+` FROM billing_invoice_application WHERE id = ?`, appID)
	app, err := scanInvoiceApplication(row)
	if err == sql.ErrNoRows {
		return nil, ErrInvoiceApplicationNotFound
	}
	return app, err
}

func loadPendingInvoiceApplicationByOrder(ctx context.Context, exec sqlExecutor, tenantID, orderID int64) (*InvoiceApplication, error) {
	row := exec.QueryRowContext(ctx, `
		SELECT `+invoiceAppSelectCols+`
		FROM billing_invoice_application
		WHERE tenant_id = ? AND order_id = ? AND status = ?
		ORDER BY id ASC LIMIT 1`, tenantID, orderID, invoiceAppPending)
	app, err := scanInvoiceApplication(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return app, err
}

func orderHasActiveBlueInvoice(ctx context.Context, exec sqlExecutor, orderID int64) (bool, error) {
	var exists int
	err := exec.QueryRowContext(ctx, `
		SELECT 1 FROM billing_invoice
		WHERE order_id = ? AND kind = ? AND status IN (?, ?)
		LIMIT 1`, orderID, invoiceKindBlue, invoiceStatusIssuing, invoiceStatusIssued).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func listInvoicesForOrder(ctx context.Context, orderID int64) ([]map[string]interface{}, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, order_id, application_id, related_invoice_id, kind, purpose, status,
		       amount_yuan_cents, fapiao_id, wechat_apply_id, wechat_fapiao_number, invoice_file_path,
		       buyer_snapshot, fail_reason, created_at, updated_at
		FROM billing_invoice WHERE order_id = ? ORDER BY id ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(
			&inv.ID, &inv.TenantID, &inv.OrderID, &inv.ApplicationID, &inv.RelatedInvoiceID,
			&inv.Kind, &inv.Purpose, &inv.Status, &inv.AmountYuanCents, &inv.FapiaoID,
			&inv.WechatApplyID, &inv.WechatFapiaoNumber, &inv.InvoiceFilePath, &inv.BuyerSnapshot,
			&inv.FailReason, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, invoiceJSON(&inv))
	}
	return out, rows.Err()
}

func latestInvoiceApplicationForOrder(ctx context.Context, orderID int64) (*InvoiceApplication, error) {
	row := db.QueryRowContext(ctx, `
		SELECT `+invoiceAppSelectCols+`
		FROM billing_invoice_application WHERE order_id = ?
		ORDER BY id DESC LIMIT 1`, orderID)
	app, err := scanInvoiceApplication(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return app, err
}

func attachOrderInvoices(m map[string]interface{}, orderID int64) {
	ctx := context.Background()
	invoices, err := listInvoicesForOrder(ctx, orderID)
	if err != nil {
		m["invoices"] = []map[string]interface{}{}
		return
	}
	m["invoices"] = invoices
	if conf := orderReverseConfirmFromInvoices(invoices); conf != nil {
		m["invoice_reverse_confirm"] = conf
	}
	app, err := latestInvoiceApplicationForOrder(ctx, orderID)
	if err == nil && app != nil {
		m["invoice_application"] = invoiceApplicationJSON(app)
	}
}

// invoiceListFilter 开票申请列表过滤条件（OPT-20260823-045：管理端表头列过滤）。
type invoiceListFilter struct {
	ID        int64
	TenantID  int64
	OrderID   int64
	BuyerName string
	Status    string
}

func listInvoiceApplications(ctx context.Context, f invoiceListFilter) ([]map[string]interface{}, error) {
	query := `SELECT ` + invoiceAppSelectCols + ` FROM billing_invoice_application`
	args := []interface{}{}
	where := []string{}
	if f.ID > 0 {
		where = append(where, "id = ?")
		args = append(args, f.ID)
	}
	if f.TenantID > 0 {
		where = append(where, "tenant_id = ?")
		args = append(args, f.TenantID)
	}
	if f.OrderID > 0 {
		where = append(where, "order_id = ?")
		args = append(args, f.OrderID)
	}
	if name := strings.TrimSpace(f.BuyerName); name != "" {
		where = append(where, "buyer_name LIKE ?")
		args = append(args, "%"+name+"%")
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		where = append(where, "status = ?")
		args = append(args, s)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		app, err := scanInvoiceApplication(rows)
		if err != nil {
			return nil, err
		}
		m := invoiceApplicationJSON(app)
		if app.InvoiceFilePath != "" {
			m["invoice_file_url"] = fmt.Sprintf("/api/system-admin/invoice-applications/%s/invoice-file/", formatID(app.ID))
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// snapshotInvoiceType 从买家快照 JSON 解析发票类型（billing_invoice 无独立列）。
func snapshotInvoiceType(snapshot string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(snapshot), &m); err != nil {
		return ""
	}
	t, _ := m["invoice_type"].(string)
	if t == "" {
		t = invoiceTypeGeneral
	}
	return t
}

func buyerFromApplication(app *InvoiceApplication) invoiceBuyerInput {
	return invoiceBuyerInput{
		InvoiceType: app.InvoiceType,
		Type:        app.BuyerType,
		Name:        app.BuyerName,
		TaxpayerID:  app.TaxpayerID,
		Address:     app.Address,
		Telephone:   app.Telephone,
		BankName:    app.BankName,
		BankAccount: app.BankAccount,
	}
}

func newMerchantFapiaoID() string {
	return fmt.Sprintf("fp-%d", generateSnowflakeID())
}
