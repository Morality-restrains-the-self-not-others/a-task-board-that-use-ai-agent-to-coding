package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOpenAPIJSONPublicOnly(t *testing.T) {
	t.Setenv("TASK_GATEWAY_PUBLIC_BASE", "https://gateway.example:8443")
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
	s0 := servers[0].(map[string]interface{})
	if s0["url"] != "https://gateway.example:8443" {
		t.Fatalf("server url: %v", s0["url"])
	}
	paths, ok := doc["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("missing paths")
	}
	required := []string{
		"/api/health/",
		"/api/tenant/{tenant_id}/billing/accounts/balance/",
		"/api/tenant/{tenant_id}/billing/transactions/list_filtered/",
		"/api/tenant/{tenant_id}/billing/statistics/",
		"/api/billing/wechat/notify/",
		"/api/tenant/{tenant_id}/billing/orders/",
		"/api/tenant/{tenant_id}/billing/orders/{order_id}/comments/",
		"/api/tenant/{tenant_id}/billing/quotas/",
		"/api/tenant/{tenant_id}/billing/order-pricing/",
	}
	for _, p := range required {
		if _, ok := paths[p]; !ok {
			t.Fatalf("missing path %s", p)
		}
	}
	for p := range paths {
		if strings.HasPrefix(p, "/api/internal/") {
			t.Fatalf("public openapi must not include internal path %s", p)
		}
	}
}

func TestBuildOpenAPIInternalJSON(t *testing.T) {
	body, err := buildOpenAPIInternalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatal(err)
	}
	paths, ok := doc["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("missing paths")
	}
	required := []string{
		"/api/internal/taskbill/consume-task-post-quota/",
		"/api/internal/taskbill/accounts/",
		"/api/internal/taskbill/charge-server-start/",
	}
	for _, p := range required {
		if _, ok := paths[p]; !ok {
			t.Fatalf("missing internal path %s", p)
		}
	}
}

func TestHandleOpenAPISchema(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/schema/", nil)
	rec := httptest.NewRecorder()
	handleOpenAPISchema(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"openapi"`) {
		t.Fatal("missing openapi key")
	}
	if strings.Contains(body, "consume-task-post-quota") {
		t.Fatal("public schema must not mix internal charge path")
	}
	if !strings.Contains(body, "order-pricing") {
		t.Fatal("missing order-pricing path")
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
	if !strings.Contains(body, "consume-task-post-quota") {
		t.Fatal("internal schema missing consume-task-post-quota")
	}
	if strings.Contains(body, "order-pricing") {
		t.Fatal("internal schema must not mix public order-pricing path")
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
	if !strings.Contains(html, "schema-internal") {
		t.Fatal("expected schema-internal url in UI")
	}
}

func TestMountRoutesServesSchema(t *testing.T) {
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/schema/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/schema-internal/", nil)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("internal status %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestOpenAPIRoutesScriptNoDrift(t *testing.T) {
	candidates := []string{
		filepath.Join("..", "..", "db", "scripts", "ci", "check_go_openapi_routes.py"),
		filepath.Join("..", "scripts", "check_openapi_routes.py"),
	}
	var script string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			script = c
			break
		}
	}
	if script == "" {
		t.Fatal("openapi routes checker not found")
	}
	cmd := exec.Command("python3", script, "--service", "taskBill", "--fail-on-extra", "--no-warn-extra")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", script, err, out)
	}
	if !strings.Contains(string(out), "OK [taskBill]") {
		t.Fatalf("unexpected output: %s", out)
	}
}
