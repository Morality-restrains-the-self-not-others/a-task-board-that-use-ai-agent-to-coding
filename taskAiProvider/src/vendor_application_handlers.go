package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
	"tracelog"
)

// vendorStatusOf 由 ai_provider_vendor 行推导申请状态（OPT-20260806-065 审核流）：
// qualified（is_active=1）/ rejected（review_note 非空）/ pending（其余）/ none（无记录）。
func vendorStatusOf(v *domain.Vendor) string {
	if v == nil {
		return "none"
	}
	if v.IsActive {
		return "qualified"
	}
	if strings.TrimSpace(v.ReviewNote) != "" {
		return "rejected"
	}
	return "pending"
}

func isSyntheticEmail(email string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(email)), "@sso.invalid")
}

func applicantHasBindableEmail(email string) bool {
	e := strings.TrimSpace(email)
	return e != "" && !isSyntheticEmail(e)
}

// vendorContactVerifiedFn 可被测试覆盖：默认查 taskAuth SMS gate。
var vendorContactVerifiedFn = defaultVendorContactVerified

func defaultVendorContactVerified(a *App, ctx context.Context, userID string) (bool, error) {
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	reqURL := base + "/api/internal/recharge-sms-gate/?user_id=" + url.QueryEscape(userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(a.Cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return false, nil
	}
	var gate struct {
		SmsVerified bool `json:"sms_verified"`
	}
	if err := json.Unmarshal(raw, &gate); err != nil {
		return false, err
	}
	return gate.SmsVerified, nil
}

// handleVendorApplicationUpload POST multipart 上传身份证/营业执照。
func (a *App) handleVendorApplicationUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, _, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	if a.vendorDocsBackend() == domain.VendorDocBackendCOS {
		writeJSON(w, http.StatusGone, map[string]any{"detail": "请改用 upload-url 预签名直传"})
		return
	}
	if err := r.ParseMultipartForm(infrastructure.MaxVendorDocBytes + (1 << 20)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效上传表单"})
		return
	}
	kind := strings.TrimSpace(r.FormValue("kind"))
	if !infrastructure.IsVendorKYCKind(kind) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "kind 必须为 id_card 或 business_license"})
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "缺少 file"})
		return
	}
	defer file.Close()
	key, _, err := infrastructure.SaveVendorDocument(a.Cfg.VendorDocsDir, uid, kind, hdr.Filename, file, hdr.Size)
	if err != nil {
		logWarn(r.Context(), "event=VendorDocUploadFailed user_id=%d kind=%s err=%v", uid, kind, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "上传失败：" + err.Error()})
		return
	}
	logInfo("event=VendorDocUploaded user_id=%d kind=%s", uid, kind)
	writeJSON(w, http.StatusOK, map[string]any{"kind": kind, "file_key": key})
}

// handleVendorApplication 提交/重提厂商申请（OPT-20260806-065 + 证照/联系方式）。
func (a *App) handleVendorApplication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, email, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	var body struct {
		CompanyName            string `json:"company_name"`
		ContactName            string `json:"contact_name"`
		IDCardFileKey          string `json:"id_card_file_key"`
		BusinessLicenseFileKey string `json:"business_license_file_key"`
		ContactPhone           string `json:"contact_phone"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	company := strings.TrimSpace(body.CompanyName)
	contact := strings.TrimSpace(body.ContactName)
	idKey := strings.TrimSpace(body.IDCardFileKey)
	licKey := strings.TrimSpace(body.BusinessLicenseFileKey)
	phone := strings.TrimSpace(body.ContactPhone)
	if company == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "公司名称必填"})
		return
	}
	if contact == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "联系人必填"})
		return
	}
	if len(company) > 255 || len(contact) > 255 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "公司名称/联系人长度超限（≤255）"})
		return
	}
	if idKey == "" || licKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "请上传身份证与营业执照"})
		return
	}
	if phone == "" || len(phone) > 32 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "请提供已验证的联系手机号"})
		return
	}
	if err := a.docStore().Head(r.Context(), uid, idKey); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "身份证文件无效，请重新上传"})
		return
	}
	if err := a.docStore().Head(r.Context(), uid, licKey); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "营业执照文件无效，请重新上传"})
		return
	}
	verified, err := vendorContactVerifiedFn(a, r.Context(), strconv.FormatInt(uid, 10))
	if err != nil {
		logWarn(r.Context(), "event=VendorContactVerifyCheckFailed user_id=%d err=%v", uid, err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "联系方式验证服务暂不可用，请稍后重试"})
		return
	}
	if !verified {
		writeJSON(w, http.StatusForbidden, map[string]any{"detail": "请先完成手机短信验证"})
		return
	}
	in := infrastructure.VendorApplicationInput{
		CompanyName:            company,
		ContactName:            contact,
		IDCardFileKey:          idKey,
		BusinessLicenseFileKey: licKey,
		ContactPhone:           phone,
	}
	vendor, conflict, err := a.DB.ApplyVendorApplication(uid, email, in)
	if err != nil {
		logWarn(r.Context(), "vendor-application apply failed user_id=%d err=%v", uid, err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "提交申请失败，请稍后重试"})
		return
	}
	if conflict {
		if vendor != nil && vendor.IsActive {
			writeJSON(w, http.StatusConflict, map[string]any{"detail": "已是厂商门户成员"})
		} else {
			writeJSON(w, http.StatusConflict, map[string]any{"detail": "申请审核中，请耐心等待"})
		}
		return
	}
	logInfo("event=VendorApplicationSubmitted user_id=%d vendor_id=%d", uid, vendor.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": vendorStatusOf(vendor), "vendor": vendorJSON(vendor)})
}

// handleAdminVendorDocument GET staff 下载证照。
func (a *App) handleAdminVendorDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	path := r.URL.Path
	// .../admin-vendors/{id}/documents/{kind}/ 或 legacy /api/admin/vendors/...
	kind := ""
	vendorID := int64(0)
	for _, prefix := range []string{"/api/ai-provider/admin-vendors/", "/api/admin/vendors/"} {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if len(parts) >= 3 && parts[1] == "documents" {
			id, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil && id > 0 && infrastructure.IsVendorKYCKind(parts[2]) {
				vendorID = id
				kind = parts[2]
			}
		}
	}
	if vendorID <= 0 || kind == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "路径无效"})
		return
	}
	v, err := a.DB.GetVendorByID(vendorID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "厂商不存在"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "查询失败"})
		return
	}
	key := v.IDCardFileKey
	if kind == infrastructure.VendorDocKindBusinessLicense {
		key = v.BusinessLicenseFileKey
	}
	if strings.TrimSpace(key) == "" || v.SaasUserID == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "证照不存在"})
		return
	}
	rc, ct, err := a.docStore().Open(r.Context(), *v.SaasUserID, key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "证照文件缺失"})
		return
	}
	defer rc.Close()
	if ct == "" {
		ct = infrastructure.VendorDocContentType(key)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "inline; filename=\""+kind+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

// handleAdminVendorReview 运营审核（approve/reject）。
func (a *App) handleAdminVendorReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	s, ok := a.requireStaff(w, r)
	if !ok {
		return
	}
	vendorID, ok := parsePathID(r.URL.Path, "/api/ai-provider/admin-vendors/")
	if !ok {
		vendorID, ok = parsePathID(r.URL.Path, "/api/admin/vendors/")
	}
	if !ok || vendorID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "vendor id 无效"})
		return
	}
	var body struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	action := strings.TrimSpace(body.Action)
	if action != "approve" && action != "reject" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "action 必须为 approve 或 reject"})
		return
	}
	vendor, err := a.DB.ReviewVendor(vendorID, action, strings.TrimSpace(body.Note), s.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]any{"detail": "厂商不存在"})
			return
		}
		logWarn(r.Context(), "admin-vendor review failed vendor_id=%d err=%v", vendorID, err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "审核失败，请稍后重试"})
		return
	}
	logInfo("event=VendorReviewed vendor_id=%d action=%s staff_id=%d", vendorID, action, s.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": vendorStatusOf(vendor), "vendor": vendorJSON(vendor)})
}

// vendorJSON 序列化厂商（vendor-status / 申请 / 审核共用）。
func vendorJSON(v *domain.Vendor) map[string]any {
	m := map[string]any{
		"id": infrastructure.IDStr(v.ID), "email": v.Email, "company_name": v.CompanyName,
		"contact_name": v.ContactName, "is_active": v.IsActive, "review_note": v.ReviewNote,
		"is_synthetic_email":   isSyntheticEmail(v.Email),
		"has_id_card":          strings.TrimSpace(v.IDCardFileKey) != "",
		"has_business_license": strings.TrimSpace(v.BusinessLicenseFileKey) != "",
		"contact_phone_masked": maskContactPhone(v.ContactPhone),
	}
	if v.SaasUserID != nil {
		m["saas_user_id"] = infrastructure.IDStr(*v.SaasUserID)
	} else {
		m["saas_user_id"] = nil
	}
	return m
}

func maskContactPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if len(p) < 7 {
		if p == "" {
			return ""
		}
		return "****"
	}
	return p[:3] + "****" + p[len(p)-4:]
}
