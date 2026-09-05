package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 1x1 PNG
var miniPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
	0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d, 0xb0, 0x00, 0x00, 0x00,
	0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestMemberAvatarUploadListAndServe(t *testing.T) {
	mux := setupTestService(t)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("user_id", "admin1")
	_ = w.WriteField("company_id", "c1")
	part, err := w.CreateFormFile("avatar", "a.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(miniPNG); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/members/avatar", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("upload status=%d body=%s", rec.Code, rec.Body.String())
	}
	var uploaded map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	url, _ := uploaded["member_avatar_url"].(string)
	if !strings.Contains(url, "/api/tenant/c1/accounts/members/m1/avatar") {
		t.Fatalf("member_avatar_url=%q", url)
	}
	stored, _ := uploaded["member_avatar"].(string)
	if stored == "" {
		t.Fatal("expected stored path")
	}
	abs := filepath.Join(memberAvatarMediaRoot, filepath.FromSlash(stored))
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("file missing: %v", err)
	}

	reqList := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "admin1")
	recList := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(recList, reqList)
	if recList.Code != 200 {
		t.Fatalf("list status=%d %s", recList.Code, recList.Body.String())
	}
	var listOut map[string]interface{}
	_ = json.Unmarshal(recList.Body.Bytes(), &listOut)
	members, _ := listOut["members"].([]interface{})
	if len(members) < 1 {
		t.Fatalf("members=%v", listOut)
	}
	m0, _ := members[0].(map[string]interface{})
	if m0["member_avatar_url"] != url {
		t.Fatalf("list avatar=%v want %v", m0["member_avatar_url"], url)
	}

	reqGet := httptest.NewRequest(http.MethodGet, url, nil)
	recGet := httptest.NewRecorder()
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != 200 {
		t.Fatalf("serve status=%d %s", recGet.Code, recGet.Body.String())
	}
	if !strings.HasPrefix(recGet.Header().Get("Content-Type"), "image/") {
		t.Fatalf("content-type=%q", recGet.Header().Get("Content-Type"))
	}
	if len(recGet.Body.Bytes()) < 8 {
		t.Fatalf("empty body")
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "/api/internal/tenant/members/avatar?user_id=admin1&company_id=c1", nil)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)
	if recDel.Code != 200 {
		t.Fatalf("delete status=%d %s", recDel.Code, recDel.Body.String())
	}
	var deleted map[string]interface{}
	_ = json.Unmarshal(recDel.Body.Bytes(), &deleted)
	if got, _ := deleted["member_avatar_url"].(string); strings.TrimSpace(got) != "" {
		t.Fatalf("after delete avatar_url=%v", deleted["member_avatar_url"])
	}
}
