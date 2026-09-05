package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedWorkspace(t *testing.T, tenantID, workspaceID string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO project_workspace_entries(id,name,description,company_id) VALUES(?,?,?,?)`,
		workspaceID, "WS "+workspaceID, "", tenantID,
	)
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
}

func doWorkPanelFilter(method, tenant, ws, user, body string) *httptest.ResponseRecorder {
	return doWorkPanelFilterWithAuth(method, tenant, ws, body, func(req *http.Request) {
		if user != "" {
			req.Header.Set("X-Auth-User-Id", user)
		}
	})
}

func doWorkPanelFilterWithAuth(method, tenant, ws, body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/workspaces/%s/work-panel-filters/tenant_id/%s/", ws, tenant)
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	return rec
}

func TestWorkPanelFilterGetDefault(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doWorkPanelFilter(http.MethodGet, "t1", "ws1", "u1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	bars, _ := body["deliverable_filter_bars"].([]interface{})
	if len(bars) != 1 {
		t.Fatalf("expected 1 default bar, got %d", len(bars))
	}
}

func TestWorkPanelFilterPutGetRoundtrip(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	payload := `{
		"version": 1,
		"deliverable_filter_bars": [
			{"id":"bar-0","path":[{"type":"root"}]},
			{"id":"bar-1","path":[{"type":"root"},{"type":"category","id":"c1","label":"Cat"}]}
		]
	}`
	rec := doWorkPanelFilter(http.MethodPut, "t1", "ws1", "u1", payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := doWorkPanelFilter(http.MethodGet, "t1", "ws1", "u1", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&body)
	bars, _ := body["deliverable_filter_bars"].([]interface{})
	if len(bars) != 2 {
		t.Fatalf("expected 2 bars, got %d", len(bars))
	}
}

func TestWorkPanelFilterWorkspaceIsolation(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "wsA")
	seedWorkspace(t, "t1", "wsB")

	putA := `{"deliverable_filter_bars":[{"id":"a1","path":[{"type":"root"}]},{"id":"a2","path":[{"type":"root"}]}]}`
	doWorkPanelFilter(http.MethodPut, "t1", "wsA", "u1", putA)
	putB := `{"deliverable_filter_bars":[{"id":"b1","path":[{"type":"root"}]}]}`
	doWorkPanelFilter(http.MethodPut, "t1", "wsB", "u1", putB)

	recA := doWorkPanelFilter(http.MethodGet, "t1", "wsA", "u1", "")
	var bodyA map[string]interface{}
	json.NewDecoder(recA.Body).Decode(&bodyA)
	barsA, _ := bodyA["deliverable_filter_bars"].([]interface{})
	if len(barsA) != 2 {
		t.Fatalf("wsA expected 2 bars, got %d", len(barsA))
	}

	recB := doWorkPanelFilter(http.MethodGet, "t1", "wsB", "u1", "")
	var bodyB map[string]interface{}
	json.NewDecoder(recB.Body).Decode(&bodyB)
	barsB, _ := bodyB["deliverable_filter_bars"].([]interface{})
	if len(barsB) != 1 {
		t.Fatalf("wsB expected 1 bar, got %d", len(barsB))
	}
}

func TestWorkPanelFilterUserIsolation(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	putU1 := `{"deliverable_filter_bars":[{"id":"u1a","path":[{"type":"root"}]},{"id":"u1b","path":[{"type":"root"}]}]}`
	doWorkPanelFilter(http.MethodPut, "t1", "ws1", "u1", putU1)

	recU2 := doWorkPanelFilter(http.MethodGet, "t1", "ws1", "u2", "")
	var body map[string]interface{}
	json.NewDecoder(recU2.Body).Decode(&body)
	bars, _ := body["deliverable_filter_bars"].([]interface{})
	if len(bars) != 1 {
		t.Fatalf("u2 should see default 1 bar, got %d", len(bars))
	}
}

func TestWorkPanelFilterMissingWorkspace(t *testing.T) {
	setupTestDB(t)
	rec := doWorkPanelFilter(http.MethodGet, "t1", "missing", "u1", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestWorkPanelFilterUnauthorized(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")
	rec := doWorkPanelFilter(http.MethodGet, "t1", "ws1", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// APISIX forward-auth injects X-User-Id + verified gateway headers; middleware
// promotes them to X-Auth-User-Id. Regression for production work-panel 401.
func TestWorkPanelFilterAcceptsGatewayXUserId(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")
	cfg.GatewayInternalSecret = "test-gw-secret"
	rec := doWorkPanelFilterWithAuth(http.MethodGet, "t1", "ws1", "", func(req *http.Request) {
		req.Header.Set("X-Gateway-Auth-Verified", "1")
		req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
		req.Header.Set("X-User-Id", "u-gw")
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with verified gateway X-User-Id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkPanelFilterRejectsUnverifiedXUserId(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")
	cfg.GatewayInternalSecret = "test-gw-secret"
	rec := doWorkPanelFilterWithAuth(http.MethodGet, "t1", "ws1", "", func(req *http.Request) {
		req.Header.Set("X-User-Id", "spoofed")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unverified X-User-Id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkPanelFilterTooManyBars(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")
	bars := make([]string, 0, 21)
	for i := 0; i < 21; i++ {
		bars = append(bars, fmt.Sprintf(`{"id":"b%d","path":[{"type":"root"}]}`, i))
	}
	payload := `{"deliverable_filter_bars":[` + strings.Join(bars, ",") + `]}`
	rec := doWorkPanelFilter(http.MethodPut, "t1", "ws1", "u1", payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestNormalizeFilterPayloadDefaults(t *testing.T) {
	p, err := normalizeFilterPayload(filterPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.DeliverableFilterBars) != 1 || p.DeliverableFilterBars[0].ID != "bar-0" {
		t.Fatalf("unexpected default: %+v", p)
	}
	if p.AccessFilter != nil {
		t.Fatalf("expected nil access_filter, got %+v", p.AccessFilter)
	}
}

func TestWorkPanelFilterAccessFilterRoundtrip(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	payload := `{
		"version": 2,
		"deliverable_filter_bars": [{"id":"bar-0","path":[{"type":"root"}]}],
		"access_filter": {"kind":"person","id":"cm-1","label":"Alice"}
	}`
	rec := doWorkPanelFilter(http.MethodPut, "t1", "ws1", "u1", payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var putBody map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&putBody)
	af, _ := putBody["access_filter"].(map[string]interface{})
	if af == nil || af["kind"] != "person" || af["id"] != "cm-1" {
		t.Fatalf("PUT response access_filter: %+v", putBody["access_filter"])
	}

	rec2 := doWorkPanelFilter(http.MethodGet, "t1", "ws1", "u1", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}
	var getBody map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&getBody)
	af2, _ := getBody["access_filter"].(map[string]interface{})
	if af2 == nil || af2["kind"] != "person" || af2["id"] != "cm-1" || af2["label"] != "Alice" {
		t.Fatalf("GET access_filter: %+v", getBody["access_filter"])
	}
}

func TestWorkPanelFilterInvalidAccessFilterCleared(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")
	payload := `{
		"deliverable_filter_bars": [{"id":"bar-0","path":[{"type":"root"}]}],
		"access_filter": {"kind":"nope","id":"x"}
	}`
	rec := doWorkPanelFilter(http.MethodPut, "t1", "ws1", "u1", payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	if body["access_filter"] != nil {
		t.Fatalf("expected null access_filter, got %+v", body["access_filter"])
	}
}
