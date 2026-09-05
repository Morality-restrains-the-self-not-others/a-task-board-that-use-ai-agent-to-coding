package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func feedbackAdminHeaders(req *http.Request, idem string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	req.Header.Set("X-User-Id", "admin1")
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
}

func feedbackTenantGET(t *testing.T, mux *http.ServeMux, tenantID int64, withRegion bool) *httptest.ResponseRecorder {
	t.Helper()
	tid := formatID(tenantID)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tid+"/billing/feedback-links/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	if withRegion {
		req.Header.Set("X-Tenant-Perms", tid+":region:nav.feedback.main:view")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func postFeedbackGroup(t *testing.T, mux *http.ServeMux, body string, idem string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/feedback-link-groups/", strings.NewReader(body))
	feedbackAdminHeaders(req, idem)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func seedTaskPostConsumed(t *testing.T, tenantID, quantity, remaining int64) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_resource_grant (id, tenant_id, resource_type, quantity, remaining, reason, created_at)
		VALUES (?, ?, ?, ?, ?, 'test', ?)`,
		generateSnowflakeID(), tenantID, ResourceTypeTaskPost, quantity, remaining, now,
	); err != nil {
		t.Fatalf("seed grant: %v", err)
	}
}

func seedGitlabTrafficUsed(t *testing.T, tenantID int64, usedGB float64) {
	t.Helper()
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, 100, 1, '', 0, ?, 'active', ?, ?)`,
		tenantID, "tencent-sh-1", usedGB, now, now,
	); err != nil {
		t.Fatalf("seed gitlab: %v", err)
	}
}

func TestFeedbackTopicsRegistered(t *testing.T) {
	if billingEventTopics["FEEDBACK_LINK_GROUP_CREATED"] != "feedback-link-group-created" {
		t.Fatal(billingEventTopics["FEEDBACK_LINK_GROUP_CREATED"])
	}
}

func TestFeedbackT1EmptyThresholdsVisible(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000001
	body := `{"name":"公开","enabled":true,"sort_order":0,"thresholds":[],"links":[{"title":"问卷","url":"https://example.com/q","enabled":true}]}`
	rec := postFeedbackGroup(t, mux, body, "ik-t1")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if got.Code != 200 {
		t.Fatalf("get %d %s", got.Code, got.Body.String())
	}
	if !strings.Contains(got.Body.String(), "公开") {
		t.Fatalf("missing group: %s", got.Body.String())
	}
}

func TestFeedbackT2HiddenBelowThreshold(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000002
	seedTaskPostConsumed(t, tid, 100, 1) // consumed 99
	body := `{"name":"百帖","enabled":true,"thresholds":[{"resource_kind":"task_post","min_quantity":100}],"links":[{"title":"VIP","url":"https://example.com/v"}]}`
	if rec := postFeedbackGroup(t, mux, body, "ik-t2"); rec.Code != 201 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if strings.Contains(got.Body.String(), "百帖") {
		t.Fatalf("should hide: %s", got.Body.String())
	}
}

func TestFeedbackT3VisibleAtThreshold(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000003
	seedTaskPostConsumed(t, tid, 100, 0)
	body := `{"name":"百帖","enabled":true,"thresholds":[{"resource_kind":"task_post","min_quantity":100}],"links":[{"title":"VIP","url":"https://example.com/v"}]}`
	if rec := postFeedbackGroup(t, mux, body, "ik-t3"); rec.Code != 201 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if !strings.Contains(got.Body.String(), "百帖") {
		t.Fatalf("should show: %s", got.Body.String())
	}
}

func TestFeedbackT4ANDFails(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000004
	seedTaskPostConsumed(t, tid, 100, 0)
	seedGitlabTrafficUsed(t, tid, 9)
	body := `{"name":"双门槛","enabled":true,"thresholds":[{"resource_kind":"task_post","min_quantity":100},{"resource_kind":"gitlab_traffic","min_quantity":10}],"links":[{"title":"A","url":"https://example.com/a"}]}`
	if rec := postFeedbackGroup(t, mux, body, "ik-t4"); rec.Code != 201 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if strings.Contains(got.Body.String(), "双门槛") {
		t.Fatalf("AND fail should hide: %s", got.Body.String())
	}
}

func TestFeedbackT5HighConsumptionSeesBoth(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000005
	seedTaskPostConsumed(t, tid, 200, 0)
	if rec := postFeedbackGroup(t, mux, `{"name":"低","enabled":true,"thresholds":[{"resource_kind":"task_post","min_quantity":0}],"links":[{"title":"L","url":"https://example.com/l"}]}`, "ik-t5a"); rec.Code != 201 {
		t.Fatalf("low %d %s", rec.Code, rec.Body.String())
	}
	if rec := postFeedbackGroup(t, mux, `{"name":"高","enabled":true,"thresholds":[{"resource_kind":"task_post","min_quantity":100}],"links":[{"title":"H","url":"https://example.com/h"}]}`, "ik-t5b"); rec.Code != 201 {
		t.Fatalf("high %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if !strings.Contains(got.Body.String(), `"name":"低"`) || !strings.Contains(got.Body.String(), `"name":"高"`) {
		t.Fatalf("want both: %s", got.Body.String())
	}
}

func TestFeedbackT6DisabledHidden(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000006
	if rec := postFeedbackGroup(t, mux, `{"name":"关组","enabled":false,"thresholds":[],"links":[{"title":"X","url":"https://example.com/x"}]}`, "ik-t6g"); rec.Code != 201 {
		t.Fatalf("group %d %s", rec.Code, rec.Body.String())
	}
	if rec := postFeedbackGroup(t, mux, `{"name":"开组","enabled":true,"thresholds":[],"links":[{"title":"隐链","url":"https://example.com/h","enabled":false}]}`, "ik-t6l"); rec.Code != 201 {
		t.Fatalf("link %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	if strings.Contains(got.Body.String(), "关组") || strings.Contains(got.Body.String(), "隐链") {
		t.Fatalf("disabled leaked: %s", got.Body.String())
	}
}

func TestFeedbackT7HTTPURLRejected(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	rec := postFeedbackGroup(t, mux, `{"name":"坏","links":[{"title":"x","url":"http://example.com"}]}`, "ik-t7")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestFeedbackT8NonAdminForbidden(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPut, "/api/system-admin/feedback-link-groups/abc/", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	req.Header.Set("Idempotency-Key", "ik-t8")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestFeedbackT9NoRegionForbidden(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	got := feedbackTenantGET(t, mux, 881000000000000009, false)
	if got.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", got.Code, got.Body.String())
	}
}

func TestFeedbackT10PublishesCreated(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	var events []string
	orig := feedbackEventPublisher
	t.Cleanup(func() { feedbackEventPublisher = orig })
	feedbackEventPublisher = func(ctx context.Context, eventType string, payload map[string]interface{}, key string) error {
		events = append(events, eventType)
		return nil
	}
	rec := postFeedbackGroup(t, mux, `{"name":"事件","links":[{"title":"t","url":"https://example.com/t"}]}`, "ik-t10")
	if rec.Code != 201 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	if len(events) != 1 || events[0] != "FEEDBACK_LINK_GROUP_CREATED" {
		t.Fatalf("events=%v", events)
	}
}

func TestFeedbackT11UnknownKind(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	rec := postFeedbackGroup(t, mux, `{"name":"x","thresholds":[{"resource_kind":"not_a_kind","min_quantity":1}]}`, "ik-t11")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestFeedbackT12NoThresholdsInTenantJSON(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid int64 = 881000000000000012
	if rec := postFeedbackGroup(t, mux, `{"name":"公开","thresholds":[{"resource_kind":"task_post","min_quantity":0}],"links":[{"title":"t","url":"https://example.com/t"}]}`, "ik-t12"); rec.Code != 201 {
		t.Fatalf("create %d %s", rec.Code, rec.Body.String())
	}
	got := feedbackTenantGET(t, mux, tid, true)
	var body map[string]interface{}
	if err := json.Unmarshal(got.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(body)
	if strings.Contains(string(raw), "thresholds") || strings.Contains(string(raw), "min_quantity") {
		t.Fatalf("thresholds leaked: %s", raw)
	}
}
