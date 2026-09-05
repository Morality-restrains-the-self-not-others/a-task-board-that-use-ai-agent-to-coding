package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"tracelog"
)

// ── License Agreement Admin CRUD ────────────────────────────────────────────

func handleAdminLicenseAgreements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListLicenseAgreements(w, r)
	case http.MethodPost:
		handleCreateLicenseAgreement(w, r)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleAdminLicenseAgreementDetail(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		handleGetLicenseAgreement(w, r, id)
	case http.MethodPut:
		handleUpdateLicenseAgreement(w, r, id)
	case http.MethodDelete:
		handleDeleteLicenseAgreement(w, r, id)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleListLicenseAgreements(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	var rows *sql.Rows
	var err error
	if kind != "" {
		rows, err = db.Query(`SELECT id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at FROM billing_license_agreements WHERE document_kind=? ORDER BY created_at DESC`, kind)
	} else {
		rows, err = db.Query(`SELECT id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at FROM billing_license_agreements ORDER BY created_at DESC`)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var a licenseAgreementRow
		if err := rows.Scan(&a.ID, &a.Title, &a.Content, &a.Version, &a.DocumentKind, &a.IsActive, &a.IsMaterialChange, &a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		out = append(out, a.toMap())
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func handleCreateLicenseAgreement(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		Version          string `json:"version"`
		DocumentKind     string `json:"document_kind"`
		IsActive         bool   `json:"is_active"`
		IsMaterialChange bool   `json:"is_material_change"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Content) == "" || strings.TrimSpace(body.Version) == "" {
		writeErrorJSON(w, http.StatusBadRequest, "title, content, version required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	kind := strings.TrimSpace(body.DocumentKind)
	if kind == "" {
		kind = "service"
	}
	isActive := 0
	if body.IsActive {
		isActive = 1
	}
	isMaterial := 0
	if body.IsMaterialChange {
		isMaterial = 1
	}

	// Deactivate other active agreements of the same kind if this one is active
	if isActive == 1 {
		_, _ = db.Exec(`UPDATE billing_license_agreements SET is_active=0, updated_at=NOW() WHERE document_kind=? AND is_active=1`, kind)
	}

	snowID := generateSnowflakeID()
	id := formatID(snowID)
	now := utcNow()
	if _, err := db.Exec(`INSERT INTO billing_license_agreements(id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		id, body.Title, body.Content, body.Version, kind, isActive, isMaterial, now, now); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusCreated, licenseAgreementRow{
		ID: id, Title: body.Title, Content: body.Content, Version: body.Version,
		DocumentKind: kind, IsActive: isActive, IsMaterialChange: isMaterial,
		CreatedAt: now, UpdatedAt: now,
	}.toMap())
}

func handleGetLicenseAgreement(w http.ResponseWriter, r *http.Request, id string) {
	var a licenseAgreementRow
	err := db.QueryRow(`SELECT id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at FROM billing_license_agreements WHERE id=?`, id).
		Scan(&a.ID, &a.Title, &a.Content, &a.Version, &a.DocumentKind, &a.IsActive, &a.IsMaterialChange, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "服务协议不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, a.toMap())
}

func handleUpdateLicenseAgreement(w http.ResponseWriter, r *http.Request, id string) {
	var exists string
	if err := db.QueryRow(`SELECT id FROM billing_license_agreements WHERE id=?`, id).Scan(&exists); err != nil {
		writeErrorJSON(w, http.StatusNotFound, "服务协议不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	var body struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		Version          string `json:"version"`
		DocumentKind     string `json:"document_kind"`
		IsActive         *bool  `json:"is_active"`
		IsMaterialChange *bool  `json:"is_material_change"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Content) == "" || strings.TrimSpace(body.Version) == "" {
		writeErrorJSON(w, http.StatusBadRequest, "title, content, version required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	kind := strings.TrimSpace(body.DocumentKind)
	isActive := 0
	if body.IsActive != nil && *body.IsActive {
		isActive = 1
	}
	isMaterial := 0
	if body.IsMaterialChange != nil && *body.IsMaterialChange {
		isMaterial = 1
	}

	if isActive == 1 {
		_, _ = db.Exec(`UPDATE billing_license_agreements SET is_active=0, updated_at=NOW() WHERE document_kind=? AND is_active=1 AND id!=?`, kind, id)
	}

	now := utcNow()
	if _, err := db.Exec(`UPDATE billing_license_agreements SET title=?,content=?,version=?,document_kind=?,is_active=?,is_material_change=?,updated_at=? WHERE id=?`,
		body.Title, body.Content, body.Version, kind, isActive, isMaterial, now, id); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, licenseAgreementRow{
		ID: id, Title: body.Title, Content: body.Content, Version: body.Version,
		DocumentKind: kind, IsActive: isActive, IsMaterialChange: isMaterial,
		UpdatedAt: now,
	}.toMap())
}

func handleDeleteLicenseAgreement(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := db.Exec(`DELETE FROM billing_license_agreements WHERE id=?`, id); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type licenseAgreementRow struct {
	ID               string
	Title            string
	Content          string
	Version          string
	DocumentKind     string
	IsActive         int
	IsMaterialChange int
	CreatedAt        string
	UpdatedAt        string
}

func (a licenseAgreementRow) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id":                 a.ID,
		"title":              a.Title,
		"content":            a.Content,
		"version":            a.Version,
		"document_kind":      a.DocumentKind,
		"is_active":          a.IsActive == 1,
		"is_material_change": a.IsMaterialChange == 1,
		"created_at":         a.CreatedAt,
		"updated_at":         a.UpdatedAt,
	}
}

// ── License Agreement Public Endpoints ───────────────────────────────────────

func handlePublicCurrentLicenseAgreement(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind == "" {
		kind = "service"
	}
	var a licenseAgreementRow
	err := db.QueryRow(`SELECT id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at FROM billing_license_agreements WHERE document_kind=? AND is_active=1 ORDER BY created_at DESC LIMIT 1`, kind).
		Scan(&a.ID, &a.Title, &a.Content, &a.Version, &a.DocumentKind, &a.IsActive, &a.IsMaterialChange, &a.CreatedAt, &a.UpdatedAt)
	if err != nil && isPaymentTermsKind(kind) {
		// Canonical/alias fallback: FE uses recharge_cents, OpenAPI historically listed recharge_points
		fallback := DocumentKindPaymentTermsAlias
		if kind == DocumentKindPaymentTermsAlias {
			fallback = DocumentKindPaymentTermsCanonical
		}
		err = db.QueryRow(`SELECT id,title,content,version,document_kind,is_active,is_material_change,created_at,updated_at FROM billing_license_agreements WHERE document_kind=? AND is_active=1 ORDER BY created_at DESC LIMIT 1`, fallback).
			Scan(&a.ID, &a.Title, &a.Content, &a.Version, &a.DocumentKind, &a.IsActive, &a.IsMaterialChange, &a.CreatedAt, &a.UpdatedAt)
	}
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "系统尚未发布服务协议，请联系管理员。", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, a.toMap())
}

func handleLicenseConsent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LicenseAgreementID string `json:"license_agreement_id"`
		AcceptedID         string `json:"accepted_license_agreement_id"`
		Context            string `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	agreementID := strings.TrimSpace(body.LicenseAgreementID)
	if agreementID == "" {
		agreementID = strings.TrimSpace(body.AcceptedID)
	}
	if agreementID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "缺少 license_agreement_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// Validate agreement exists and is active
	var isActive int
	var documentKind string
	if err := db.QueryRow(`SELECT is_active, document_kind FROM billing_license_agreements WHERE id=?`, agreementID).Scan(&isActive, &documentKind); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "服务协议不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "unauthorized", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	clientIP := resolveClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	consentContext := strings.TrimSpace(body.Context)
	if consentContext == "" {
		if isPaymentTermsKind(documentKind) {
			consentContext = "order_pay"
		} else {
			consentContext = "post_login_reconsent"
		}
	}

	consentID := formatID(generateSnowflakeID())
	now := utcNow()
	res, err := db.Exec(`INSERT IGNORE INTO billing_user_license_agreement_consents(id,user_id,license_agreement_id,consented_at,context,client_ip,user_agent) VALUES(?,?,?,?,?,?,?)`,
		consentID, userID, agreementID, now, consentContext, clientIP, userAgent)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Already consented — return existing row id
		_ = db.QueryRow(`SELECT id FROM billing_user_license_agreement_consents WHERE user_id=? AND license_agreement_id=? ORDER BY consented_at DESC LIMIT 1`,
			userID, agreementID).Scan(&consentID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "consent_id": consentID})
}

// ── Privacy Policy Admin CRUD ────────────────────────────────────────────────

func handleAdminPrivacyPolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListPrivacyPolicies(w, r)
	case http.MethodPost:
		handleCreatePrivacyPolicy(w, r)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleAdminPrivacyPolicyDetail(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		handleGetPrivacyPolicy(w, r, id)
	case http.MethodPut:
		handleUpdatePrivacyPolicy(w, r, id)
	case http.MethodDelete:
		handleDeletePrivacyPolicy(w, r, id)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleListPrivacyPolicies(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id,title,content,version,is_active,is_material_change,created_at,updated_at FROM billing_privacy_policies ORDER BY created_at DESC`)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var p privacyPolicyRow
		if err := rows.Scan(&p.ID, &p.Title, &p.Content, &p.Version, &p.IsActive, &p.IsMaterialChange, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		out = append(out, p.toMap())
	}
	if out == nil {
		out = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, out)
}

func handleCreatePrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		Version          string `json:"version"`
		IsActive         bool   `json:"is_active"`
		IsMaterialChange bool   `json:"is_material_change"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Content) == "" || strings.TrimSpace(body.Version) == "" {
		writeErrorJSON(w, http.StatusBadRequest, "title, content, version required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	isActive := 0
	if body.IsActive {
		isActive = 1
	}
	isMaterial := 0
	if body.IsMaterialChange {
		isMaterial = 1
	}

	if isActive == 1 {
		_, _ = db.Exec(`UPDATE billing_privacy_policies SET is_active=0, updated_at=NOW() WHERE is_active=1`)
	}

	snowID2 := generateSnowflakeID()
	id2 := formatID(snowID2)
	now := utcNow()
	if _, err := db.Exec(`INSERT INTO billing_privacy_policies(id,title,content,version,is_active,is_material_change,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		id2, body.Title, body.Content, body.Version, isActive, isMaterial, now, now); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusCreated, privacyPolicyRow{
		ID: id2, Title: body.Title, Content: body.Content, Version: body.Version,
		IsActive: isActive, IsMaterialChange: isMaterial, CreatedAt: now, UpdatedAt: now,
	}.toMap())
}

func handleGetPrivacyPolicy(w http.ResponseWriter, r *http.Request, id string) {
	var p privacyPolicyRow
	err := db.QueryRow(`SELECT id,title,content,version,is_active,is_material_change,created_at,updated_at FROM billing_privacy_policies WHERE id=?`, id).
		Scan(&p.ID, &p.Title, &p.Content, &p.Version, &p.IsActive, &p.IsMaterialChange, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "隐私条款不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, p.toMap())
}

func handleUpdatePrivacyPolicy(w http.ResponseWriter, r *http.Request, id string) {
	var exists string
	if err := db.QueryRow(`SELECT id FROM billing_privacy_policies WHERE id=?`, id).Scan(&exists); err != nil {
		writeErrorJSON(w, http.StatusNotFound, "隐私条款不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	var body struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		Version          string `json:"version"`
		IsActive         *bool  `json:"is_active"`
		IsMaterialChange *bool  `json:"is_material_change"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(body.Title) == "" || strings.TrimSpace(body.Content) == "" || strings.TrimSpace(body.Version) == "" {
		writeErrorJSON(w, http.StatusBadRequest, "title, content, version required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	isActive := 0
	if body.IsActive != nil && *body.IsActive {
		isActive = 1
	}
	isMaterial := 0
	if body.IsMaterialChange != nil && *body.IsMaterialChange {
		isMaterial = 1
	}

	if isActive == 1 {
		_, _ = db.Exec(`UPDATE billing_privacy_policies SET is_active=0, updated_at=NOW() WHERE is_active=1 AND id!=?`, id)
	}

	now := utcNow()
	if _, err := db.Exec(`UPDATE billing_privacy_policies SET title=?,content=?,version=?,is_active=?,is_material_change=?,updated_at=? WHERE id=?`,
		body.Title, body.Content, body.Version, isActive, isMaterial, now, id); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, privacyPolicyRow{
		ID: id, Title: body.Title, Content: body.Content, Version: body.Version,
		IsActive: isActive, IsMaterialChange: isMaterial, UpdatedAt: now,
	}.toMap())
}

func handleDeletePrivacyPolicy(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := db.Exec(`DELETE FROM billing_privacy_policies WHERE id=?`, id); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type privacyPolicyRow struct {
	ID               string
	Title            string
	Content          string
	Version          string
	IsActive         int
	IsMaterialChange int
	CreatedAt        string
	UpdatedAt        string
}

func (p privacyPolicyRow) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id":                 p.ID,
		"title":              p.Title,
		"content":            p.Content,
		"version":            p.Version,
		"is_active":          p.IsActive == 1,
		"is_material_change": p.IsMaterialChange == 1,
		"created_at":         p.CreatedAt,
		"updated_at":         p.UpdatedAt,
	}
}

// ── Privacy Policy Public Endpoints ──────────────────────────────────────────

func handlePublicCurrentPrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	var p privacyPolicyRow
	err := db.QueryRow(`SELECT id,title,content,version,is_active,is_material_change,created_at,updated_at FROM billing_privacy_policies WHERE is_active=1 ORDER BY created_at DESC LIMIT 1`).
		Scan(&p.ID, &p.Title, &p.Content, &p.Version, &p.IsActive, &p.IsMaterialChange, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "系统尚未发布隐私条款，请联系管理员。", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, p.toMap())
}

func handlePrivacyConsent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PrivacyPolicyID string `json:"privacy_policy_id"`
		AcceptedID      string `json:"accepted_privacy_policy_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	policyID := strings.TrimSpace(body.PrivacyPolicyID)
	if policyID == "" {
		policyID = strings.TrimSpace(body.AcceptedID)
	}
	if policyID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "缺少 privacy_policy_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	var isActive int
	if err := db.QueryRow(`SELECT is_active FROM billing_privacy_policies WHERE id=?`, policyID).Scan(&isActive); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "隐私条款不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "unauthorized", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	clientIP := resolveClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	consentID := formatID(generateSnowflakeID())
	now := utcNow()
	if _, err := db.Exec(`INSERT IGNORE INTO billing_user_privacy_policy_consents(id,user_id,privacy_policy_id,consented_at,context,client_ip,user_agent) VALUES(?,?,?,?,?,?,?)`,
		consentID, userID, policyID, now, "post_login_reconsent", clientIP, userAgent); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
