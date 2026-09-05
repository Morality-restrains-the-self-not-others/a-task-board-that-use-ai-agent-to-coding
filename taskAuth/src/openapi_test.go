package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildOpenAPIJSONUsesGatewayBase(t *testing.T) {
	cfg.GatewayPublicBase = "https://gateway.example:8443"
	body, err := buildOpenAPIJSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	servers, ok := doc["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		t.Fatal("missing servers")
	}
	s0, ok := servers[0].(map[string]interface{})
	if !ok {
		t.Fatal("server[0] type")
	}
	if s0["url"] != "https://gateway.example:8443" {
		t.Fatalf("server url: %v", s0["url"])
	}
}

func TestHandleOpenAPISchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/schema/", nil)
	rec := httptest.NewRecorder()
	handleOpenAPISchema(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type %q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), `"openapi"`) {
		t.Fatalf("body missing openapi key")
	}
}

func TestHandleSwaggerUI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/swagger/", nil)
	rec := httptest.NewRecorder()
	handleSwaggerUI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "swagger-ui") {
		t.Fatal("expected swagger-ui html")
	}
}

func TestHandleOpenAPIInternalSchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/schema-internal/", nil)
	rec := httptest.NewRecorder()
	handleOpenAPIInternalSchema(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"openapi"`) {
		t.Fatalf("body missing openapi key")
	}
	if !strings.Contains(body, "/api/internal/") {
		t.Fatalf("internal schema missing /api/internal/ paths")
	}
	if strings.Contains(body, "/api/accounts/users/login/") {
		t.Fatalf("internal schema must not mix public login paths")
	}
}

func TestHandleSwaggerInternalUI(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/swagger-internal/", nil)
	rec := httptest.NewRecorder()
	handleSwaggerInternalUI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	html := rec.Body.String()
	if !strings.Contains(html, "swagger-ui") {
		t.Fatal("expected swagger-ui html")
	}
	if !strings.Contains(html, "/api/schema-internal/") {
		t.Fatal("expected schema-internal url in UI")
	}
}
