package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func TestVendorImageGroupCreateRequiresNameDescriptionAndIcon(t *testing.T) {
	app := testApp(t)
	_, _, tok := seedVendorImageGroup(t, app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/", bytes.NewReader([]byte(`{"name":"g"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 || !strings.Contains(rr.Body.String(), "镜像组描述必填") {
		t.Fatalf("empty desc: status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/", bytes.NewReader([]byte(`{"description":"desc"}`)))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 400 || !strings.Contains(rr2.Body.String(), "镜像组名称必填") {
		t.Fatalf("empty name: status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	rrIcon := httptest.NewRecorder()
	reqIcon := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/", bytes.NewReader([]byte(`{"name":"g1","description":"d1"}`)))
	reqIcon.Header.Set("Authorization", "Bearer "+tok)
	reqIcon.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rrIcon, reqIcon)
	if rrIcon.Code != 400 || !strings.Contains(rrIcon.Body.String(), "镜像组图标必填") {
		t.Fatalf("empty icon: status=%d body=%s", rrIcon.Code, rrIcon.Body.String())
	}

	key := uploadVendorGroupIcon(t, mux, tok)

	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/", bytes.NewReader([]byte(`{"name":"  g1  ","description":" desc1 ","icon_file_key":"`+key+`"}`)))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr3, req3)
	if rr3.Code != 201 {
		t.Fatalf("create: status=%d body=%s", rr3.Code, rr3.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rr3.Body.Bytes(), &created)
	if created["name"] != "g1" || created["description"] != "desc1" {
		t.Fatalf("unexpected created: %v", created)
	}
	iconURL, _ := created["icon_url"].(string)
	if iconURL == "" || !strings.Contains(iconURL, "/icon") {
		t.Fatalf("icon_url=%v", created["icon_url"])
	}
	gid, err := parseIDFlexible(created["id"])
	if err != nil {
		t.Fatal(err)
	}
	gr := httptest.NewRecorder()
	greq := httptest.NewRequest(http.MethodGet, iconURL, nil)
	mux.ServeHTTP(gr, greq)
	if gr.Code != 200 || !strings.HasPrefix(gr.Header().Get("Content-Type"), "image/") {
		t.Fatalf("public icon: status=%d ct=%s body=%s", gr.Code, gr.Header().Get("Content-Type"), gr.Body.String())
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
}

func uploadVendorGroupIcon(t *testing.T, mux http.Handler, tok string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/icon-upload-url/", strings.NewReader(`{"filename":"icon.png","content_type":"image/png","size":8}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("upload-url: status=%d body=%s", rr.Code, rr.Body.String())
	}
	var issued map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	key, _ := issued["file_key"].(string)
	uploadURL, _ := issued["upload_url"].(string)
	if key == "" || uploadURL == "" {
		t.Fatalf("issued=%v", issued)
	}
	put := httptest.NewRequest(http.MethodPut, uploadURL, bytes.NewReader([]byte("12345678")))
	put.Header.Set("Authorization", "Bearer "+tok)
	put.Header.Set("Content-Type", "image/png")
	pr := httptest.NewRecorder()
	mux.ServeHTTP(pr, put)
	if pr.Code != 200 {
		t.Fatalf("local-put: status=%d body=%s", pr.Code, pr.Body.String())
	}
	cr := httptest.NewRecorder()
	creq := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/icon-upload-complete/", strings.NewReader(`{"file_key":"`+key+`"}`))
	creq.Header.Set("Authorization", "Bearer "+tok)
	creq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(cr, creq)
	if cr.Code != 200 {
		t.Fatalf("complete: status=%d body=%s", cr.Code, cr.Body.String())
	}
	return key
}

func TestVendorImageGroupPutRejectsEmptyIcon(t *testing.T) {
	app := testApp(t)
	_, groupID, tok := seedVendorImageGroup(t, app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/vendor/image-groups/"+infrastructure.IDStr(groupID)+"/", bytes.NewReader([]byte(`{"name":"g","description":"d"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 || !strings.Contains(rr.Body.String(), "镜像组图标必填") {
		t.Fatalf("put empty icon: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestVendorImageGroupRejectsForeignIconKey(t *testing.T) {
	app := testApp(t)
	_, _, tok := seedVendorImageGroup(t, app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/", bytes.NewReader([]byte(`{"name":"g","description":"d","icon_file_key":"999/image_group_icon_1.png"}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 || !strings.Contains(rr.Body.String(), "图标文件无效") {
		t.Fatalf("foreign key: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRewriteIconLocalPutURL(t *testing.T) {
	got := rewriteIconLocalPutURL("/api/ai-provider/vendor-application/local-put/?file_key=1%2Fx.png")
	if got != "/api/vendor/image-groups/icon-local-put/?file_key=1%2Fx.png" {
		t.Fatalf("got %s", got)
	}
}

func TestVendorImageGroupIconCOSPostUpload(t *testing.T) {
	app := testApp(t)
	fake := infrastructure.NewFakeCOS()
	app.Cfg.VendorDocsBackend = domain.VendorDocBackendCOS
	app.Docs = infrastructure.NewVendorDocStore(app.Cfg, app.Path, fake)
	vendorID, _, tok := seedVendorImageGroup(t, app)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/icon-upload-url/", strings.NewReader(`{"filename":"icon.png","content_type":"image/png","size":8}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("upload-url: status=%d body=%s", rr.Code, rr.Body.String())
	}
	var issued map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	up, _ := issued["upload_url"].(string)
	method, _ := issued["method"].(string)
	key, _ := issued["file_key"].(string)
	fields, _ := issued["form_fields"].(map[string]any)
	if method != "POST" || strings.Contains(up, "local-put") || !strings.Contains(up, "fake-cos.example") {
		t.Fatalf("COS icon upload must be POST Object, got %v", issued)
	}
	if fields["key"] != key || fields["x-cos-server-side-encryption"] != "AES256" {
		t.Fatalf("form_fields=%v", fields)
	}
	headers, _ := issued["headers"].(map[string]any)
	if headers != nil {
		if _, ok := headers["x-cos-server-side-encryption"]; ok {
			t.Fatalf("browser must not send SSE header: %v", headers)
		}
	}
	if err := app.Docs.Put(req.Context(), vendorID, key, "image/png", bytes.NewReader([]byte("12345678")), 8); err != nil {
		t.Fatalf("simulate COS POST: %v", err)
	}
	cr := httptest.NewRecorder()
	creq := httptest.NewRequest(http.MethodPost, "/api/vendor/image-groups/icon-upload-complete/", strings.NewReader(`{"file_key":"`+key+`"}`))
	creq.Header.Set("Authorization", "Bearer "+tok)
	creq.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(cr, creq)
	if cr.Code != 200 {
		t.Fatalf("complete: status=%d body=%s", cr.Code, cr.Body.String())
	}
}
