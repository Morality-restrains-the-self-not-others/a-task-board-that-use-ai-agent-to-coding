package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health/", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["service"] != "ai-provider" {
		t.Fatalf("unexpected %v", body)
	}
	if body["ok"] != true {
		t.Fatalf("not ok: %v", body)
	}
}
