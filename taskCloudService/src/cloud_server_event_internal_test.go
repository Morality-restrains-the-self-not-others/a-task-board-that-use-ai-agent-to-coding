package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInternalLatestPendingStart(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-events/latest-pending-start?company_id=c1&task_id=task1", nil)
	rec := httptest.NewRecorder()
	handleInternalLatestPendingStart(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["id"] != eventID || out["status"] != cloudServerEventStatusPending {
		t.Fatalf("out=%v", out)
	}
}

func TestInternalLatestPendingStartNotFound(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-events/latest-pending-start?company_id=c1&task_id=missing", nil)
	rec := httptest.NewRecorder()
	handleInternalLatestPendingStart(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalUpdateCloudServerEventStatusAndData(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"event_id":"` + eventID + `","status":"processing"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-events/update-status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalUpdateCloudServerEventStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	body = `{"event_id":"` + eventID + `","event_data":{"instance_id":"i-1"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-events/update-data", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handleInternalUpdateCloudServerEventData(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update-data status=%d body=%s", rec.Code, rec.Body.String())
	}

	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != cloudServerEventStatusProcessing {
		t.Fatalf("status=%s", ev.Status)
	}
	if ev.EventData["instance_id"] != "i-1" {
		t.Fatalf("event_data=%v", ev.EventData)
	}
}

// TestInternalUpdateCloudServerEventStatusGuardConflict — OPT-20260818-015 剩余 gap：
// 带 from_status 守卫的重复 claim 命中 409，taskEvents 消费者据此把丢失的 claim 当幂等跳过。
func TestInternalUpdateCloudServerEventStatusGuardConflict(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	claimBody := func() string {
		return `{"event_id":"` + eventID + `","status":"processing","from_status":"pending"}`
	}

	// 首次带守卫 claim → 200。
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-events/update-status", strings.NewReader(claimBody()))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalUpdateCloudServerEventStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first claim status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 重复 claim（事件已是 processing）→ 409。
	req = httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-events/update-status", strings.NewReader(claimBody()))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handleInternalUpdateCloudServerEventStatus(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second claim status=%d want 409 body=%s", rec.Code, rec.Body.String())
	}
}

func TestInternalCloudServerEventStatusEndpoint(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud-server-events/status?event_id="+eventID, nil)
	rec := httptest.NewRecorder()
	handleInternalCloudServerEventStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["event_id"] != eventID || out["status"] != cloudServerEventStatusSuccess {
		t.Fatalf("out=%v", out)
	}
}

func TestInternalAccessKeyIAMAssociation(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"auth_id":"auth1","access_key":"AK_TEST","iam_id":"iam-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/access-key-iam-associations/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalAccessKeyIAMRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var authID, accessKey, iamID string
	if err := db.QueryRow(`
		SELECT cloud_platform_auth_id, access_key, iam_id FROM cloud_access_key_iam_associations LIMIT 1
	`).Scan(&authID, &accessKey, &iamID); err != nil {
		t.Fatal(err)
	}
	if authID != "auth1" || accessKey != "AK_TEST" || iamID != "iam-1" {
		t.Fatalf("row=%s %s %s", authID, accessKey, iamID)
	}
}

func TestInternalImportCloudServerEvents(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"events":[{"id":"ev-import-1","company_id":"c1","workspace_id":"w1","task_id":"t1","company_member_id":"m1","event_type":"start","event_data":"{\"task_id\":\"t1\"}","status":"pending"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-events/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalImportCloudServerEvents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ev, err := loadCloudServerEvent("ev-import-1")
	if err != nil {
		t.Fatal(err)
	}
	if ev.TaskID != "t1" || ev.Status != cloudServerEventStatusPending {
		t.Fatalf("ev=%+v", ev)
	}
}

func TestInternalImportAccessKeyIAMAssociations(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"items":[{"id":"iam-import-1","cloud_platform_auth_id":"auth1","access_key":"AK","iam_id":"iam-1"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/access-key-iam-associations/import", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalImportAccessKeyIAMAssociations(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_access_key_iam_associations WHERE id='iam-import-1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row, got %d", cnt)
	}
}

func TestStatusPayloadFromEventPending(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	payload := statusPayloadFromEvent(ev, "c1", "task1")
	if payload["status"] != "processing" || payload["progress"] != 60 {
		t.Fatalf("payload=%v", payload)
	}
}

func TestStatusPayloadFromEventSuccess(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'aliyun', 'i-99', '8.8.8.8', 'http://srv/', 'cn-h', 'cn-h-a', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	payload := statusPayloadFromEvent(ev, "c1", "task1")
	if payload["status"] != "success" || payload["progress"] != 100 {
		t.Fatalf("payload=%v", payload)
	}
	vmInfo, _ := payload["vm_info"].(map[string]interface{})
	if vmInfo["instance_id"] != "i-99" || vmInfo["public_ip"] != "8.8.8.8" {
		t.Fatalf("vm_info=%v", vmInfo)
	}
}
