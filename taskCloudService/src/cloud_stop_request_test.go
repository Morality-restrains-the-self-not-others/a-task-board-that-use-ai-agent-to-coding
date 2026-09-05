package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecordCloudStopRequestIdempotent(t *testing.T) {
	setupCloudTestDB(t)
	if err := recordCloudStopRequest("sr-1", "t1", "w1", "task1", "i-1"); err != nil {
		t.Fatal(err)
	}
	// 同 stop_request_id 再次写入不得报错（INSERT IGNORE）且不产生第二行。
	if err := recordCloudStopRequest("sr-1", "t1", "w1", "task1", "i-1"); err != nil {
		t.Fatalf("replay record should be idempotent: %v", err)
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_stop_request WHERE stop_request_id = 'sr-1'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("count=%d want 1", cnt)
	}
}

func TestCloudStopRequestProcessed(t *testing.T) {
	setupCloudTestDB(t)
	if p, err := cloudStopRequestProcessed("sr-missing"); err != nil || p {
		t.Fatalf("missing: processed=%v err=%v", p, err)
	}
	if err := recordCloudStopRequest("sr-2", "t1", "w1", "task1", "i-2"); err != nil {
		t.Fatal(err)
	}
	if p, err := cloudStopRequestProcessed("sr-2"); err != nil || !p {
		t.Fatalf("recorded: processed=%v err=%v", p, err)
	}
}

func TestInternalStopRequestProcessedEndpoint(t *testing.T) {
	setupCloudTestDB(t)
	if err := recordCloudStopRequest("sr-3", "t1", "w1", "task1", "i-3"); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud/stops/processed?stop_request_id=sr-3", nil)
	rec := httptest.NewRecorder()
	handleInternalStopRequestRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["processed"] != true {
		t.Fatalf("out=%v", out)
	}
}

func TestInternalClearAfterStopRecordsStopRequest(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudConfig(t, "t1", "w1", "task1", "http://127.0.0.1:8080")
	body := strings.NewReader(`{"tenant_id":"t1","workspace_id":"w1","task_id":"task1","stop_reason":"stop_vm","instance_id":"i-1","stop_request_id":"sr-5"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud-server-config/clear-after-stop/", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalCloudServerConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if p, err := cloudStopRequestProcessed("sr-5"); err != nil || !p {
		t.Fatalf("after clear-after-stop: processed=%v err=%v", p, err)
	}
}
