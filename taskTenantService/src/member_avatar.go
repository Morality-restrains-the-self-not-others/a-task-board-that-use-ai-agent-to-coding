package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxMemberAvatarBytes = 2 << 20 // 2 MiB
)

var memberAvatarMediaRoot string

func initMemberAvatarMediaRoot(repoRoot string) {
	root := filepath.Join(repoRoot, "data", "task_tenant_media")
	_ = os.MkdirAll(root, 0o755)
	memberAvatarMediaRoot = root
}

func memberAvatarPublicURL(companyID, memberID, stored string) string {
	stored = strings.TrimSpace(stored)
	companyID = strings.TrimSpace(companyID)
	memberID = strings.TrimSpace(memberID)
	if stored == "" || companyID == "" || memberID == "" {
		return ""
	}
	return fmt.Sprintf("/api/tenant/%s/accounts/members/%s/avatar", companyID, memberID)
}

func memberAvatarAbsPath(stored string) (string, error) {
	stored = strings.TrimSpace(stored)
	if stored == "" || memberAvatarMediaRoot == "" {
		return "", fmt.Errorf("empty avatar path")
	}
	// Prevent path traversal: only allow under media root.
	clean := filepath.Clean("/" + stored)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid avatar path")
	}
	abs := filepath.Join(memberAvatarMediaRoot, clean)
	rel, err := filepath.Rel(memberAvatarMediaRoot, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("avatar path escapes media root")
	}
	return abs, nil
}

func extForAvatarContentType(ct string) string {
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func detectAvatarContentType(data []byte, fallback string) string {
	ct := http.DetectContentType(data)
	if strings.HasPrefix(ct, "image/") {
		return ct
	}
	fb := strings.ToLower(strings.TrimSpace(fallback))
	if strings.HasPrefix(fb, "image/") {
		return fb
	}
	return ""
}

func saveMemberAvatarFile(companyID, memberID string, data []byte, contentType string) (string, error) {
	companyID = strings.TrimSpace(companyID)
	memberID = strings.TrimSpace(memberID)
	if companyID == "" || memberID == "" {
		return "", fmt.Errorf("company_id and member_id required")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("empty avatar")
	}
	if len(data) > maxMemberAvatarBytes {
		return "", fmt.Errorf("avatar too large")
	}
	ct := detectAvatarContentType(data, contentType)
	if ct == "" {
		return "", fmt.Errorf("unsupported avatar type")
	}
	ext := extForAvatarContentType(ct)
	var rnd [8]byte
	_, _ = rand.Read(rnd[:])
	rel := filepath.ToSlash(filepath.Join(
		"company_member_avatars",
		companyID,
		memberID+"_"+hex.EncodeToString(rnd[:])+ext,
	))
	abs, err := memberAvatarAbsPath(rel)
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

func deleteMemberAvatarFile(stored string) {
	abs, err := memberAvatarAbsPath(stored)
	if err != nil {
		return
	}
	_ = os.Remove(abs)
}

func setMemberAvatarPath(userID, companyID, rel string) error {
	_, err := db.Exec(
		`UPDATE tenant_company_member SET member_avatar=?, updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`,
		rel, userID, companyID,
	)
	return err
}

func clearMemberAvatarPath(userID, companyID string) (old string, err error) {
	m, err := getMember(userID, companyID)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", fmt.Errorf("not found")
	}
	old = m.MemberAvatar
	_, err = db.Exec(
		`UPDATE tenant_company_member SET member_avatar='', updated_at=CURRENT_TIMESTAMP WHERE user_id=? AND company_id=?`,
		userID, companyID,
	)
	return old, err
}

func handleMemberAvatarGET(w http.ResponseWriter, r *http.Request, tenantID, memberID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	m, err := getMemberByID(memberID, tenantID)
	if err != nil || m == nil {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if strings.TrimSpace(m.MemberAvatar) == "" {
		writeError(w, r, http.StatusNotFound, "avatar not set")
		return
	}
	abs, err := memberAvatarAbsPath(m.MemberAvatar)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "avatar missing")
		return
	}
	f, err := os.Open(abs)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "avatar missing")
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		writeError(w, r, http.StatusNotFound, "avatar missing")
		return
	}
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ct := http.DetectContentType(buf[:n])
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeError(w, r, http.StatusInternalServerError, "read failed")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeContent(w, r, filepath.Base(abs), stat.ModTime(), f)
}

func handleInternalMemberAvatar(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	switch r.Method {
	case http.MethodPost:
		handleInternalMemberAvatarUpload(w, r)
	case http.MethodDelete:
		handleInternalMemberAvatarDelete(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleInternalMemberAvatarUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxMemberAvatarBytes + 1<<20); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid multipart")
		return
	}
	uid := strings.TrimSpace(r.FormValue("user_id"))
	cid := strings.TrimSpace(r.FormValue("company_id"))
	if uid == "" || cid == "" {
		writeError(w, r, http.StatusBadRequest, "user_id and company_id required")
		return
	}
	m, err := getMember(uid, cid)
	if err != nil || m == nil {
		writeError(w, r, http.StatusNotFound, "member not found")
		return
	}
	file, hdr, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "avatar file required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxMemberAvatarBytes+1))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "read avatar failed")
		return
	}
	if len(data) > maxMemberAvatarBytes {
		writeError(w, r, http.StatusBadRequest, "avatar too large")
		return
	}
	ct := hdr.Header.Get("Content-Type")
	rel, err := saveMemberAvatarFile(cid, m.ID, data, ct)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	old := m.MemberAvatar
	if err := setMemberAvatarPath(uid, cid, rel); err != nil {
		deleteMemberAvatarFile(rel)
		writeError(w, r, http.StatusInternalServerError, "update failed")
		return
	}
	if old != "" && old != rel {
		deleteMemberAvatarFile(old)
	}
	m.MemberAvatar = rel
	writeJSON(w, http.StatusOK, memberToJSON(m))
}

func handleInternalMemberAvatarDelete(w http.ResponseWriter, r *http.Request) {
	uid := strings.TrimSpace(r.URL.Query().Get("user_id"))
	cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if uid == "" || cid == "" {
		body, _ := readJSONBody(r)
		if uid == "" {
			uid = strField(body, "user_id")
		}
		if cid == "" {
			cid = strField(body, "company_id")
		}
	}
	if uid == "" || cid == "" {
		writeError(w, r, http.StatusBadRequest, "user_id and company_id required")
		return
	}
	old, err := clearMemberAvatarPath(uid, cid)
	if err != nil {
		if err.Error() == "not found" {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if old != "" {
		deleteMemberAvatarFile(old)
	}
	m, _ := getMember(uid, cid)
	writeJSON(w, http.StatusOK, memberToJSON(m))
}
