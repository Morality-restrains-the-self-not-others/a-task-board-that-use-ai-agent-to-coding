package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"authz"
	"tracelog"
)

const (
	// maxInvoiceFileBytes 手动开具发票文件大小上限（10 MiB，PDF/图片）。
	maxInvoiceFileBytes = 10 << 20
	// invoiceFileMediaDir 数据目录下子目录（data/ 已 gitignore）。
	invoiceFileMediaDir = "taskbill_invoice_files"
)

var invoiceFileMediaRoot string

func initInvoiceFileMediaRoot(repoRoot string) {
	root := filepath.Join(repoRoot, "data", invoiceFileMediaDir)
	_ = os.MkdirAll(root, 0o755)
	invoiceFileMediaRoot = root
}

// invoiceFileAbsPath 与 taskTenantService member_avatar 同款防目录穿越守卫。
func invoiceFileAbsPath(stored string) (string, error) {
	stored = strings.TrimSpace(stored)
	if stored == "" || invoiceFileMediaRoot == "" {
		return "", fmt.Errorf("empty invoice file path")
	}
	clean := filepath.Clean("/" + stored)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid invoice file path")
	}
	abs := filepath.Join(invoiceFileMediaRoot, clean)
	rel, err := filepath.Rel(invoiceFileMediaRoot, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invoice file path escapes media root")
	}
	return abs, nil
}

func extForInvoiceContentType(ct string) string {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "application/pdf":
		return ".pdf"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ""
	}
}

// detectInvoiceContentType 仅放行 PDF/JPEG/PNG；内容探测优先，header 兜底。
func detectInvoiceContentType(data []byte, fallback string) string {
	ct := http.DetectContentType(data)
	if extForInvoiceContentType(ct) != "" {
		return ct
	}
	fb := strings.ToLower(strings.TrimSpace(fallback))
	if extForInvoiceContentType(fb) != "" {
		return fb
	}
	return ""
}

func saveInvoiceFile(appID int64, data []byte, contentType string) (string, error) {
	if appID <= 0 {
		return "", fmt.Errorf("application_id required")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty invoice file")
	}
	if len(data) > maxInvoiceFileBytes {
		return "", fmt.Errorf("invoice file too large")
	}
	ct := detectInvoiceContentType(data, contentType)
	if ct == "" {
		return "", fmt.Errorf("unsupported invoice file type (pdf/jpg/png only)")
	}
	var rnd [8]byte
	_, _ = rand.Read(rnd[:])
	rel := filepath.ToSlash(filepath.Join(
		"invoice_files",
		formatID(appID)+"_"+hex.EncodeToString(rnd[:])+extForInvoiceContentType(ct),
	))
	abs, err := invoiceFileAbsPath(rel)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

func deleteInvoiceFile(stored string) {
	abs, err := invoiceFileAbsPath(stored)
	if err != nil {
		return
	}
	_ = os.Remove(abs)
}

// parseInvoiceFileAppID 解析 .../invoice-applications/{id}/invoice-file/ 的申请 ID。
func parseInvoiceFileAppID(path string) (int64, bool) {
	for _, prefix := range []string{
		"/api/system-admin/invoice-applications/",
		"/api/internal/taskbill/invoice-applications/",
	} {
		rest, cut := strings.CutPrefix(path, prefix)
		if !cut {
			continue
		}
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) == 2 && parts[1] == "invoice-file" {
			id, err := strconv.ParseInt(parts[0], 10, 64)
			return id, err == nil && id > 0
		}
	}
	return 0, false
}

// handleSystemAdminInvoiceFileUpload 管理员上传手动开具的发票文件（multipart，字段名 file）。
// 仅 pending 申请可挂文件；重复上传覆盖旧文件。authz 由 router 的 requireInvoiceAdminAuth 完成。
func handleSystemAdminInvoiceFileUpload(w http.ResponseWriter, r *http.Request, appID int64) {
	traceID := tracelog.TraceIDFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, maxInvoiceFileBytes+(1<<20))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid multipart form", traceID)
		return
	}
	fh, _, err := r.FormFile("file")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "missing file field", traceID)
		return
	}
	defer fh.Close()
	data, err := io.ReadAll(fh)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "read file failed", traceID)
		return
	}
	rel, err := saveInvoiceFile(appID, data, r.Header.Get("Content-Type"))
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}

	app, err := loadInvoiceApplication(r.Context(), db, appID)
	if err != nil {
		deleteInvoiceFile(rel)
		if err == ErrInvoiceApplicationNotFound {
			writeErrorJSON(w, http.StatusNotFound, err.Error(), traceID)
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), traceID)
		return
	}
	if app.Status != invoiceAppPending {
		deleteInvoiceFile(rel)
		writeErrorJSON(w, http.StatusBadRequest, ErrInvoiceApplicationNotPend.Error(), traceID)
		return
	}
	old := app.InvoiceFilePath
	if _, err := db.ExecContext(r.Context(), `
		UPDATE billing_invoice_application SET invoice_file_path = ?, updated_at = ?
		WHERE id = ? AND status = ?`,
		rel, utcNow(), appID, invoiceAppPending); err != nil {
		deleteInvoiceFile(rel)
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), traceID)
		return
	}
	if old != "" && old != rel {
		deleteInvoiceFile(old)
	}
	slog.InfoContext(r.Context(), "invoice_file_uploaded",
		"level", "info",
		"application_id", formatID(appID),
		"tenant_id", formatID(app.TenantID),
		"order_id", formatID(app.OrderID),
		"file_path", rel,
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"invoice_file_path": rel,
		"invoice_file_url":  fmt.Sprintf("/api/system-admin/invoice-applications/%s/invoice-file/", formatID(appID)),
	})
}

// handleSystemAdminInvoiceFileGET 管理员查看 pending 申请的发票文件。
func handleSystemAdminInvoiceFileGET(w http.ResponseWriter, r *http.Request, appID int64) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	app, err := loadInvoiceApplication(r.Context(), db, appID)
	if err != nil {
		if err == ErrInvoiceApplicationNotFound {
			writeErrorJSON(w, http.StatusNotFound, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(app.InvoiceFilePath) == "" {
		writeErrorJSON(w, http.StatusNotFound, "invoice file not set", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	serveInvoiceFile(w, r, app.InvoiceFilePath)
}

// handleInvoiceFileGET 租户/管理员查看订单发票行挂载的手动开具发票文件。
// 鉴权：平台员工直通；租户侧需 billing:view 权限。
func handleInvoiceFileGET(w http.ResponseWriter, r *http.Request) {
	traceID := tracelog.TraceIDFromContext(r.Context())
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", traceID)
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "订单不存在", traceID)
		return
	}
	orderID, err := parseOrderIDFromPath(r.URL.Path, "/orders/")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}
	invID, ok := parseInvoiceFileInvID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusNotFound, "not found", traceID)
		return
	}
	if !authz.IsPlatformStaff(r) && !authz.RequirePerm(w, r, authz.PermBillingView, formatID(tid)) {
		return
	}
	var path string
	err = db.QueryRowContext(r.Context(), `
		SELECT invoice_file_path FROM billing_invoice
		WHERE id = ? AND tenant_id = ? AND order_id = ?`,
		invID, tid, orderID).Scan(&path)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "invoice not found", traceID)
		return
	}
	if strings.TrimSpace(path) == "" {
		writeErrorJSON(w, http.StatusNotFound, "invoice file not set", traceID)
		return
	}
	serveInvoiceFile(w, r, path)
}

func parseInvoiceFileInvID(path string) (int64, bool) {
	idx := strings.Index(path, "/invoices/")
	if idx < 0 {
		return 0, false
	}
	rest := strings.TrimSuffix(path[idx+len("/invoices/"):], "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "file" {
		return 0, false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	return id, err == nil && id > 0
}

func serveInvoiceFile(w http.ResponseWriter, r *http.Request, stored string) {
	abs, err := invoiceFileAbsPath(stored)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "invoice file missing", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "invoice file missing", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "read failed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ct := http.DetectContentType(buf[:n])
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "read failed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "inline; filename=\"invoice"+extForInvoiceContentType(ct)+"\"")
	http.ServeContent(w, r, filepath.Base(abs), stat.ModTime(), f)
}
