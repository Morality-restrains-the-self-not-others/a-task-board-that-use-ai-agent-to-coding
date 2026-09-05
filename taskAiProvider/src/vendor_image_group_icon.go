package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"taskAiProvider/domain"
)

func parseImageGroupPathID(path string) (int64, bool) {
	if id, ok := parsePathID(path, "/api/vendor/image-groups/"); ok {
		return id, true
	}
	return parsePathID(path, "/api/ai-provider/vendor-image-groups/")
}

func imageGroupIconSubpath(path string) string {
	p := strings.TrimSuffix(path, "/")
	switch {
	case strings.HasSuffix(p, "/icon-upload-url"):
		return "upload-url"
	case strings.HasSuffix(p, "/icon-upload-complete"):
		return "upload-complete"
	case strings.HasSuffix(p, "/icon-local-put"):
		return "local-put"
	default:
		return ""
	}
}

func rewriteIconLocalPutURL(u string) string {
	u = strings.Replace(u, "/api/ai-provider/vendor-application/local-put/", "/api/vendor/image-groups/icon-local-put/", 1)
	u = strings.Replace(u, "/api/vendor/application/local-put/", "/api/vendor/image-groups/icon-local-put/", 1)
	return u
}

func (a *App) handleImageGroupIconUploadURL(w http.ResponseWriter, r *http.Request, v *domain.Vendor) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	var body struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int64  `json:"size"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	ps, err := a.docStore().PresignPut(r.Context(), v.ID, domain.VendorDocKindImageGroupIcon, strings.TrimSpace(body.Filename), strings.TrimSpace(body.ContentType), body.Size)
	if err != nil {
		logWarn(r.Context(), "event=ImageGroupIconPresignFailed vendor_id=%d err=%v", v.ID, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无法签发上传地址：" + err.Error()})
		return
	}
	ps.UploadURL = rewriteIconLocalPutURL(ps.UploadURL)
	logInfo("event=ImageGroupIconPresignIssued vendor_id=%d", v.ID)
	writeJSON(w, http.StatusOK, presignPutJSON(ps))
}

func (a *App) handleImageGroupIconUploadComplete(w http.ResponseWriter, r *http.Request, v *domain.Vendor) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	var body struct {
		FileKey string `json:"file_key"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	key := strings.TrimSpace(body.FileKey)
	if err := a.docStore().Head(r.Context(), v.ID, key); err != nil {
		logWarn(r.Context(), "event=ImageGroupIconCompleteFailed vendor_id=%d err=%v", v.ID, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "对象未确认，请重新上传"})
		return
	}
	_ = a.eventBus().Publish(r.Context(), domain.EventImageGroupIconUploaded, map[string]any{
		"vendor_id": v.ID,
	})
	logInfo("event=ImageGroupIconUploaded vendor_id=%d", v.ID)
	writeJSON(w, http.StatusOK, map[string]any{"file_key": key})
}

func (a *App) handleImageGroupIconLocalPut(w http.ResponseWriter, r *http.Request, v *domain.Vendor) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	key := strings.TrimSpace(r.URL.Query().Get("file_key"))
	buf, err := io.ReadAll(io.LimitReader(r.Body, domain.MaxImageGroupIconBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "本地写入失败"})
		return
	}
	if len(buf) == 0 || int64(len(buf)) > domain.MaxImageGroupIconBytes {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "图标须为 PNG/JPEG/WEBP 且不超过 512KB"})
		return
	}
	if err := a.docStore().Put(r.Context(), v.ID, key, r.Header.Get("Content-Type"), bytes.NewReader(buf), int64(len(buf))); err != nil {
		logWarn(r.Context(), "event=ImageGroupIconLocalPutFailed vendor_id=%d err=%v", v.ID, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "本地写入失败"})
		return
	}
	logInfo("event=ImageGroupIconLocalPut vendor_id=%d", v.ID)
	writeJSON(w, http.StatusOK, map[string]any{"file_key": key})
}

func (a *App) handlePublicImageGroupIcon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	id, action, ok := parsePublicImageGroupIconPath(r.URL.Path)
	if !ok || action != "icon" {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "not found"})
		return
	}
	vendorID, key, err := a.DB.GetImageGroupIcon(id)
	if err != nil || strings.TrimSpace(key) == "" {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "not found"})
		return
	}
	rc, ct, err := a.docStore().Open(r.Context(), vendorID, key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"detail": "not found"})
		return
	}
	defer rc.Close()
	if ct == "" {
		ct = domain.VendorDocContentType(key)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(w, rc)
}

func parsePublicImageGroupIconPath(path string) (id int64, action string, ok bool) {
	if id, action, ok = pathAction(path, "/api/public/image-groups/"); ok {
		return id, action, true
	}
	return pathAction(path, "/api/ai-provider/public-image-groups/")
}
