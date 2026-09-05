package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postCloudAuth(t *testing.T, tenantID, payload string) (status int, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/"+tenantID+"/cloud/cloud-platform-authorizations/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, nil)
	return rec.Code, rec.Body.String()
}

func listCloudAuth(t *testing.T, tenantID string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenantID+"/cloud/cloud-platform-authorizations/", nil)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, nil)
	if rec.Code != 200 {
		t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func createdAuthID(t *testing.T, body string) string {
	t.Helper()
	var created map[string]string
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode create response: %v body=%s", err, body)
	}
	id := created["id"]
	if id == "" {
		t.Fatalf("missing id in create response: %s", body)
	}
	return id
}

func getCloudAuth(t *testing.T, tenantID, authID string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenantID+"/cloud/cloud-platform-authorizations/"+authID+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{authID})
	if rec.Code != 200 {
		t.Fatalf("detail status=%d body=%s", rec.Code, rec.Body.String())
	}
	var item map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatalf("decode detail: %v body=%s", err, rec.Body.String())
	}
	return item
}

func listCloudAuthItems(t *testing.T, tenantID string) []map[string]interface{} {
	t.Helper()
	raw := listCloudAuth(t, tenantID)
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		t.Fatalf("decode list: %v body=%s", err, raw)
	}
	return items
}

func jsonBool(t *testing.T, item map[string]interface{}, key string) bool {
	t.Helper()
	v, ok := item[key]
	if !ok {
		t.Fatalf("missing %s in %#v", key, item)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("%s type %T want bool in %#v", key, v, item)
	}
	return b
}

func findListAuth(t *testing.T, items []map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, item := range items {
		if sid, _ := item["id"].(string); sid == id {
			return item
		}
	}
	t.Fatalf("id %s not in list %#v", id, items)
	return nil
}

func TestCloudAuthCreateHonorsIsActiveTrue(t *testing.T) {
	setupCloudTestDB(t)

	status, body := postCloudAuth(t, "t1", `{
		"platform_type":"aliyun",
		"authorization_type":"access_key",
		"secret_id":"AKCREATE1234",
		"secret_key":"SKCREATE5678",
		"remark":"enabled-on-create",
		"is_active":true
	}`)
	if status != 201 {
		t.Fatalf("create status=%d body=%s", status, body)
	}
	id := createdAuthID(t, body)
	if !authIsActive("t1", id, "access_key") {
		t.Fatalf("create with is_active:true must persist as active, id=%s", id)
	}
	listed := findListAuth(t, listCloudAuthItems(t, "t1"), id)
	if !jsonBool(t, listed, "is_active") {
		t.Fatalf("list must round-trip is_active true, item=%#v", listed)
	}
	detail := getCloudAuth(t, "t1", id)
	if !jsonBool(t, detail, "is_active") {
		t.Fatalf("detail must round-trip is_active true, item=%#v", detail)
	}
}

func TestCloudAuthCreateHonorsIsActiveFalse(t *testing.T) {
	setupCloudTestDB(t)

	status, body := postCloudAuth(t, "t1", `{
		"platform_type":"aliyun",
		"authorization_type":"access_key",
		"secret_id":"AKCREATE1234",
		"secret_key":"SKCREATE5678",
		"remark":"disabled-on-create",
		"is_active":false
	}`)
	if status != 201 {
		t.Fatalf("create status=%d body=%s", status, body)
	}
	id := createdAuthID(t, body)
	if authIsActive("t1", id, "access_key") {
		t.Fatalf("create with is_active:false must stay inactive, id=%s", id)
	}
	listed := findListAuth(t, listCloudAuthItems(t, "t1"), id)
	if jsonBool(t, listed, "is_active") {
		t.Fatalf("list must round-trip is_active false, item=%#v", listed)
	}
	detail := getCloudAuth(t, "t1", id)
	if jsonBool(t, detail, "is_active") {
		t.Fatalf("detail must round-trip is_active false, item=%#v", detail)
	}
}

func TestCloudAuthCreateDefaultsIsActiveTrueWhenOmitted(t *testing.T) {
	setupCloudTestDB(t)

	status, body := postCloudAuth(t, "t1", `{
		"platform_type":"aliyun",
		"authorization_type":"access_key",
		"secret_id":"AKCREATE1234",
		"secret_key":"SKCREATE5678",
		"remark":"default-enabled"
	}`)
	if status != 201 {
		t.Fatalf("create status=%d body=%s", status, body)
	}
	id := createdAuthID(t, body)
	if !authIsActive("t1", id, "access_key") {
		t.Fatalf("create without is_active must default to enabled (UI checkbox default), id=%s", id)
	}
	listed := findListAuth(t, listCloudAuthItems(t, "t1"), id)
	if !jsonBool(t, listed, "is_active") {
		t.Fatalf("omitted is_active must list as true, item=%#v", listed)
	}
	detail := getCloudAuth(t, "t1", id)
	if !jsonBool(t, detail, "is_active") {
		t.Fatalf("omitted is_active must detail as true, item=%#v", detail)
	}
}

func TestCloudAuthUpdateHonorsIsActive(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")

	payload := `{"remark":"activate-via-edit","is_active":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/cloud/cloud-platform-authorizations/auth1/", strings.NewReader(payload))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleCloudAuthRoutes(rec, req, []string{"auth1"})
	if rec.Code != 200 {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !authIsActive("t1", "auth1", "access_key") {
		t.Fatalf("PUT is_active:true must persist as active")
	}
}

func TestCloudAuthListUsesAuthorizationTypeForIsActive(t *testing.T) {
	setupCloudTestDB(t)

	// An oauth-type CPA must key its active-method lookup on authorization_type
	// rather than a hardcoded access_key (OPT-20260828-023).
	status, body := postCloudAuth(t, "t1", `{
		"platform_type":"aliyun",
		"authorization_type":"oauth",
		"secret_id":"OAUTHLIST1234",
		"secret_key":"OAUTHLIST5678",
		"remark":"oauth-type-cpa",
		"is_active":true
	}`)
	if status != 201 {
		t.Fatalf("create status=%d body=%s", status, body)
	}
	id := createdAuthID(t, body)

	listed := findListAuth(t, listCloudAuthItems(t, "t1"), id)
	if !jsonBool(t, listed, "is_active") {
		t.Fatalf("oauth-type CPA must list as active, item=%#v", listed)
	}
	detail := getCloudAuth(t, "t1", id)
	if !jsonBool(t, detail, "is_active") {
		t.Fatalf("oauth-type CPA must detail as active, item=%#v", detail)
	}

	// Toggle-off must also key on the row's authorization_type.
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/toggle-active/", strings.NewReader(`{"id":"`+id+`","is_active":false}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleToggleActive(rec, req)
	if rec.Code != 200 {
		t.Fatalf("toggle-off status=%d body=%s", rec.Code, rec.Body.String())
	}
	if authIsActive("t1", id, "oauth") {
		t.Fatalf("oauth-type CPA must be deactivated after toggle-off")
	}
}

func TestCloudAuthCreateRollsBackOnActiveMethodWriteFailure(t *testing.T) {
	setupCloudTestDB(t)

	prev := failSetExclusiveActiveMethodTx
	failSetExclusiveActiveMethodTx = func() error { return fmt.Errorf("injected active method write failure") }
	t.Cleanup(func() { failSetExclusiveActiveMethodTx = prev })

	const secretID = "AKROLLBACK1234"
	status, body := postCloudAuth(t, "t1", `{
		"platform_type":"aliyun",
		"authorization_type":"access_key",
		"secret_id":"`+secretID+`",
		"secret_key":"SKROLLBACK5678",
		"remark":"rollback-on-failure",
		"is_active":true
	}`)
	if status != 500 {
		t.Fatalf("create status=%d want 500 body=%s", status, body)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_platform_authorizations WHERE company_id='t1' AND secret_id=?`, secretID).Scan(&n); err != nil {
		t.Fatalf("count residual auth rows: %v", err)
	}
	if n != 0 {
		t.Fatalf("residual cloud_platform_authorizations row after failed create: count=%d", n)
	}
	var activeN int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_platform_authorization_active_methods WHERE company_id='t1'`).Scan(&activeN); err != nil {
		t.Fatalf("count active method rows: %v", err)
	}
	if activeN != 0 {
		t.Fatalf("residual active method row after failed create: count=%d", activeN)
	}
}

func TestBoolFieldJSONEncodings(t *testing.T) {
	if !boolField(map[string]interface{}{"is_active": true}, "is_active") {
		t.Fatalf("bool true")
	}
	if boolField(map[string]interface{}{"is_active": false}, "is_active") {
		t.Fatalf("bool false")
	}
	if !boolField(map[string]interface{}{"is_active": "true"}, "is_active") {
		t.Fatalf("string true")
	}
	if boolField(map[string]interface{}{}, "is_active") {
		t.Fatalf("missing is false")
	}
}
