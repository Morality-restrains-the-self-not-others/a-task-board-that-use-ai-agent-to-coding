package main

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"authz"
	"tracelog"
)

func handleTenantInvoiceApplications(w http.ResponseWriter, r *http.Request) {
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.RequirePerm(w, r, authz.PermBillingManage, formatID(tid)) {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	buyer, err := parseInvoiceBuyer(body)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	app, err := applyInvoiceApplication(r.Context(), tid, orderID, userID, buyer, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		writeInvoiceApplyError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, invoiceApplicationJSON(app))
}

func handleTenantListInvoices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if _, _, err := loadOrder(tid, orderID); err != nil {
		writeOrderLookupMiss(w, r, tid, orderID)
		return
	}
	items, err := listInvoicesForOrder(r.Context(), orderID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
}

func writeInvoiceApplyError(w http.ResponseWriter, r *http.Request, err error) {
	traceID := tracelog.TraceIDFromContext(r.Context())
	switch {
	case errors.Is(err, ErrInvoiceBuyerInvalid), errors.Is(err, ErrInvoiceOrgTaxIDRequired),
		errors.Is(err, ErrInvoiceSpecialInfoRequired), errors.Is(err, ErrInvoiceOrderNotPaid),
		errors.Is(err, ErrInvoiceZeroAmount):
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), traceID)
	case errors.Is(err, ErrInvoicePendingExists), errors.Is(err, ErrInvoiceAlreadyIssued):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "INVOICE_CONFLICT"})
	case strings.Contains(err.Error(), "订单不存在"):
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", traceID)
	default:
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), traceID)
	}
}

func isInvoiceApplicationListPath(path string) bool {
	for _, p := range []string{
		"/api/system-admin/invoice-applications",
		"/api/internal/taskbill/invoice-applications",
	} {
		if path == p || path == p+"/" {
			return true
		}
	}
	return false
}

func parseInvoiceApplicationAction(path string) (id int64, action string, ok bool) {
	prefixes := []string{
		"/api/system-admin/invoice-applications/",
		"/api/internal/taskbill/invoice-applications/",
	}
	var rest string
	for _, prefix := range prefixes {
		if after, cut := strings.CutPrefix(path, prefix); cut {
			rest = strings.Trim(after, "/")
			break
		}
	}
	if rest == "" {
		return 0, "", false
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	action = strings.TrimSuffix(parts[1], "/")
	if action != "approve" && action != "reject" {
		return 0, "", false
	}
	return id, action, true
}

func requireInvoiceAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/api/system-admin/") {
		if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
			slog.WarnContext(r.Context(), "invoice_admin_unauthorized",
				"level", "warn",
				"reason", "missing_gateway_verify",
			)
			writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
			return false
		}
		if !authz.IsPlatformStaff(r) {
			slog.WarnContext(r.Context(), "invoice_admin_forbidden",
				"level", "warn",
				"reason", "not_platform_staff",
			)
			writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
			return false
		}
		return true
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	return true
}

func handleSystemAdminInvoiceApplicationsRouter(w http.ResponseWriter, r *http.Request) {
	if !requireInvoiceAdminAuth(w, r) {
		return
	}
	path := r.URL.Path
	if isInvoiceApplicationListPath(path) {
		if r.Method != http.MethodGet {
			writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		f := invoiceListFilter{Status: strings.TrimSpace(r.URL.Query().Get("status"))}
		// OPT-20260823-045：管理端表头列过滤 — id / tenant_id / order_id 等值 + buyer_name 模糊。
		if v := strings.TrimSpace(r.URL.Query().Get("id")); v != "" {
			f.ID, _ = strconv.ParseInt(v, 10, 64)
		}
		if v := strings.TrimSpace(r.URL.Query().Get("tenant_id")); v != "" {
			f.TenantID, _ = strconv.ParseInt(v, 10, 64)
		}
		if v := strings.TrimSpace(r.URL.Query().Get("order_id")); v != "" {
			f.OrderID, _ = strconv.ParseInt(v, 10, 64)
		}
		f.BuyerName = strings.TrimSpace(r.URL.Query().Get("buyer_name"))
		items, err := listInvoiceApplications(r.Context(), f)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
		return
	}
	if strings.HasSuffix(path, "/invoice-file/") {
		appID, ok := parseInvoiceFileAppID(path)
		if !ok {
			writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		switch r.Method {
		case http.MethodPost:
			handleSystemAdminInvoiceFileUpload(w, r, appID)
		case http.MethodGet:
			handleSystemAdminInvoiceFileGET(w, r, appID)
		default:
			writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		}
		return
	}
	appID, action, ok := parseInvoiceApplicationAction(path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	reviewer := strings.TrimSpace(stringField(body, "reviewer_user_id"))
	if reviewer == "" {
		reviewer = strings.TrimSpace(r.Header.Get("X-User-Id"))
	}
	note, _ := body["note"].(string)
	fapiaoNumber, _ := body["fapiao_number"].(string)
	switch action {
	case "approve":
		app, err := approveInvoiceApplication(r.Context(), appID, reviewer, note, fapiaoNumber)
		if err != nil {
			if errors.Is(err, ErrInvoiceApplicationNotFound) {
				writeErrorJSON(w, http.StatusNotFound, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			if errors.Is(err, ErrInvoiceApplicationNotPend) || errors.Is(err, ErrInvoiceFapiaoNumberTooLong) || errors.Is(err, ErrInvoiceZeroAmount) {
				writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, invoiceApplicationJSON(app))
	case "reject":
		app, err := rejectInvoiceApplication(r.Context(), appID, reviewer, note)
		if err != nil {
			if errors.Is(err, ErrInvoiceApplicationNotFound) || errors.Is(err, ErrInvoiceApplicationNotPend) {
				writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, invoiceApplicationJSON(app))
	default:
		writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
	}
}
