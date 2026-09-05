package main

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func (a *App) handleVendorApplicationUploadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, _, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind        string `json:"kind"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	if !domain.IsVendorKYCKind(strings.TrimSpace(body.Kind)) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "kind 必须为 id_card 或 business_license"})
		return
	}
	ps, err := a.docStore().PresignPut(r.Context(), uid, strings.TrimSpace(body.Kind), strings.TrimSpace(body.Filename), strings.TrimSpace(body.ContentType), body.Size)
	if err != nil {
		logWarn(r.Context(), "event=VendorDocPresignFailed user_id=%d kind=%s err=%v", uid, body.Kind, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无法签发上传地址：" + err.Error()})
		return
	}
	logInfo("event=VendorDocPresignIssued user_id=%d kind=%s file_key=%s", uid, body.Kind, ps.FileKey)
	writeJSON(w, http.StatusOK, presignPutJSON(ps))
}

func (a *App) handleVendorApplicationUploadComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, _, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind    string `json:"kind"`
		FileKey string `json:"file_key"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	kind := strings.TrimSpace(body.Kind)
	key := strings.TrimSpace(body.FileKey)
	if !domain.IsVendorKYCKind(kind) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "kind 必须为 id_card 或 business_license"})
		return
	}
	if err := a.docStore().Head(r.Context(), uid, key); err != nil {
		logWarn(r.Context(), "event=VendorDocCompleteFailed user_id=%d kind=%s err=%v", uid, kind, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "对象未确认，请重新上传"})
		return
	}
	_ = a.eventBus().Publish(r.Context(), domain.EventVendorDocumentUploaded, map[string]any{
		"user_id":  uid,
		"kind":     kind,
		"file_key": key,
	})
	logInfo("event=VendorDocumentUploaded user_id=%d kind=%s file_key=%s", uid, kind, key)
	writeJSON(w, http.StatusOK, map[string]any{"kind": kind, "file_key": key})
}

func (a *App) handleVendorApplicationLocalPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, _, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("file_key"))
	buf, err := io.ReadAll(io.LimitReader(r.Body, domain.MaxVendorDocBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "本地写入失败"})
		return
	}
	if len(buf) == 0 || int64(len(buf)) > domain.MaxVendorDocBytes {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "文件过大或为空"})
		return
	}
	if err := a.docStore().Put(r.Context(), uid, key, r.Header.Get("Content-Type"), bytes.NewReader(buf), int64(len(buf))); err != nil {
		logWarn(r.Context(), "event=VendorDocLocalPutFailed user_id=%d err=%v", uid, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "本地写入失败"})
		return
	}
	logInfo("event=VendorDocLocalPut user_id=%d file_key=%s", uid, key)
	writeJSON(w, http.StatusOK, map[string]any{"file_key": key})
}

func (a *App) handleAdminVendorDocsStorage(w http.ResponseWriter, r *http.Request) {
	if !a.allowMarketplaceAdmin(w, r) {
		return
	}
	prefix, rule := defaultPathView(a)
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, vendorDocsStorageJSON(a, prefix, rule))
	case http.MethodPatch, http.MethodPut:
		var body struct {
			KeyPrefix *string `json:"keyPrefix"`
			PathRule  *string `json:"pathRule"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
			return
		}
		if body.KeyPrefix != nil {
			prefix = strings.TrimSpace(*body.KeyPrefix)
		}
		if body.PathRule != nil {
			rule = strings.TrimSpace(*body.PathRule)
		}
		if err := domain.ValidatePathRule(rule); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "pathRule 无效：" + err.Error()})
			return
		}
		if prefix == "" || strings.Contains(prefix, "..") || strings.Contains(prefix, "/") {
			writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "keyPrefix 无效"})
			return
		}
		if err := infrastructure.WriteVendorDocsPathFragment(a.Cfg.RepoRoot, prefix, rule); err != nil {
			logWarn(r.Context(), "event=VendorDocPathRuleWriteFailed err=%v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "写回 conf 失败"})
			return
		}
		if a.Path != nil {
			a.Path.Set(prefix, rule)
		}
		a.Cfg.VendorDocsCOS.KeyPrefix = prefix
		a.Cfg.VendorDocsCOS.PathRule = rule
		staffID := int64(0)
		if s, ok := a.requireStaffSilent(r); ok {
			staffID = s.ID
		}
		_ = a.eventBus().Publish(r.Context(), domain.EventVendorDocPathRuleUpdated, map[string]any{
			"staff_id":   staffID,
			"key_prefix": prefix,
			"path_rule":  rule,
		})
		logInfo("event=VendorDocPathRuleUpdated key_prefix=%s path_rule=%s", prefix, rule)
		writeJSON(w, http.StatusOK, vendorDocsStorageJSON(a, prefix, rule))
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
	}
}

func presignPutJSON(ps *domain.PresignPut) map[string]any {
	if ps == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"file_key":   ps.FileKey,
		"upload_url": ps.UploadURL,
		"method":     ps.Method,
		"headers":    ps.Headers,
		"expires_in": ps.ExpiresIn,
	}
	if len(ps.FormFields) > 0 {
		out["form_fields"] = ps.FormFields
	}
	return out
}

func defaultPathView(a *App) (string, string) {
	if a != nil && a.Path != nil {
		return a.Path.Get()
	}
	if a != nil && a.Cfg != nil {
		return a.Cfg.VendorDocsCOS.KeyPrefix, a.Cfg.VendorDocsCOS.PathRule
	}
	return "vendor-docs", "{keyPrefix}/{userId}/{kind}_{id}{ext}"
}

func vendorDocsStorageJSON(a *App, prefix, rule string) map[string]any {
	backend, bucket, region := domain.VendorDocBackendLocal, "", ""
	if a != nil && a.Cfg != nil {
		if a.Cfg.VendorDocsBackend != "" {
			backend = a.Cfg.VendorDocsBackend
		}
		bucket = a.Cfg.VendorDocsCOS.Bucket
		region = a.Cfg.VendorDocsCOS.Region
	}
	if a != nil && a.Docs != nil {
		backend = a.Docs.Backend()
	}
	return map[string]any{
		"backend":   backend,
		"bucket":    bucket,
		"region":    region,
		"keyPrefix": prefix,
		"pathRule":  rule,
	}
}

func (a *App) requireStaffSilent(r *http.Request) (*domain.Staff, bool) {
	tok := bearerToken(r)
	if tok == "" || a.DB == nil {
		return nil, false
	}
	claims, err := infrastructure.ParseHS256JWT(tok, a.Cfg.SecretKey, infrastructure.JWTIssuer, "", "staff")
	if err != nil {
		return nil, false
	}
	sub, _ := strconv.ParseInt(infrastructure.ClaimString(claims, "sub"), 10, 64)
	s, err := a.DB.GetStaffByID(sub)
	if err != nil || s == nil || !s.IsActive {
		return nil, false
	}
	return s, true
}
