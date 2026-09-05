package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInsertPendingStartEventCreatesRow(t *testing.T) {
	setupCloudTestDB(t)
	eventID, merged, err := insertPendingStartEvent(
		"c1", "w1", "task1", "m1", "",
		map[string]interface{}{"task_id": "task1", "region_id": "cn-hongkong"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if eventID == "" {
		t.Fatal("expected event id")
	}
	if merged["region_id"] != "cn-hongkong" {
		t.Fatalf("merged=%v", merged)
	}

	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != cloudServerEventStatusPending || ev.EventType != cloudServerEventTypeStart {
		t.Fatalf("ev=%+v", ev)
	}
	if ev.EventData["task_id"] != "task1" {
		t.Fatalf("event_data=%v", ev.EventData)
	}
}

func TestInsertPendingStartEventPrunesFailedEvents(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`
		INSERT INTO cloud_server_events
			(id, company_id, workspace_id, task_id, company_member_id, event_type, event_data, status, error_message)
		VALUES ('old-failed', 'c1', 'w1', 'task1', 'm1', 'start', '{"task_id":"task1"}', 'error', 'old failure')
	`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO cloud_server_events
			(id, company_id, workspace_id, task_id, company_member_id, event_type, event_data, status)
		VALUES ('old-success', 'c1', 'w1', 'task1', 'm1', 'start', '{"task_id":"task1"}', 'success')
	`)
	if err != nil {
		t.Fatal(err)
	}

	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE id='old-failed'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 0 {
		t.Fatal("failed start event should be pruned")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE id='old-success'`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatal("success start event should remain")
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM cloud_server_events WHERE id=?`, eventID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != cloudServerEventStatusPending {
		t.Fatalf("status=%s", status)
	}
}

func TestDeletePendingStartEvent(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := deletePendingStartEvent(eventID, "c1"); err != nil {
		t.Fatal(err)
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_server_events WHERE id=?`, eventID).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 0 {
		t.Fatal("pending event should be deleted")
	}
}

func TestUpdateCloudServerEventStatusAndData(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1", "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusProcessing, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventData(eventID, map[string]interface{}{"b": 2}); err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusError, "boom", ""); err != nil {
		t.Fatal(err)
	}

	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != cloudServerEventStatusError || ev.ErrorMessage != "boom" {
		t.Fatalf("ev=%+v", ev)
	}
	if ev.EventData["a"] != float64(1) || ev.EventData["b"] != float64(2) {
		t.Fatalf("merged event_data=%v", ev.EventData)
	}
}

// TestUpdateCloudServerEventStatusFromStatusGuard — OPT-20260818-015 剩余 gap：
// pending→processing 的 claim 必须是原子的，携带 from_status 守卫后，
// 第二次（并发）claim 因状态已迁移而失败，防止双消费者同时启动 VM。
func TestUpdateCloudServerEventStatusFromStatusGuard(t *testing.T) {
	setupCloudTestDB(t)
	eventID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	// 首次 claim：pending→processing 原子迁移成功。
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusProcessing, "", cloudServerEventStatusPending); err != nil {
		t.Fatalf("first guarded claim should succeed: %v", err)
	}
	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != cloudServerEventStatusProcessing {
		t.Fatalf("status=%s want processing", ev.Status)
	}

	// 并发重复 claim：状态已不是 pending → conflict。
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusProcessing, "", cloudServerEventStatusPending); !errors.Is(err, errCloudServerEventStatusConflict) {
		t.Fatalf("second guarded claim want errCloudServerEventStatusConflict, got %v", err)
	}

	// 对不存在的行带守卫 → conflict（而非 "not found"，让消费者统一按丢 claim 处理）。
	if err := updateCloudServerEventStatus("missing-event", cloudServerEventStatusProcessing, "", cloudServerEventStatusPending); !errors.Is(err, errCloudServerEventStatusConflict) {
		t.Fatalf("guarded update on missing row want conflict, got %v", err)
	}

	// 无守卫更新不存在的行仍返回 "not found"（既有语义）。
	if err := updateCloudServerEventStatus("missing-event", cloudServerEventStatusError, "", ""); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unguarded update on missing row want not found error, got %v", err)
	}

	// 成功迁移（processing→success）不受守卫限制（终态推进幂等无害）。
	if err := updateCloudServerEventStatus(eventID, cloudServerEventStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestLatestPendingStartEvent(t *testing.T) {
	setupCloudTestDB(t)
	firstID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(firstID, cloudServerEventStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
	secondID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1"})
	if err != nil {
		t.Fatal(err)
	}

	id, status, err := latestPendingStartEvent("c1", "task1")
	if err != nil {
		t.Fatal(err)
	}
	if id != secondID || status != cloudServerEventStatusPending {
		t.Fatalf("id=%s status=%s want %s pending", id, status, secondID)
	}
}

func TestLatestPendingStartEventNotFound(t *testing.T) {
	setupCloudTestDB(t)
	_, _, err := latestPendingStartEvent("c1", "missing-task")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pending start event not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestLoadLatestStartEvent(t *testing.T) {
	setupCloudTestDB(t)
	firstID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1", "n": 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCloudServerEventStatus(firstID, cloudServerEventStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
	secondID, _, err := insertPendingStartEvent("c1", "w1", "task1", "m1", "", map[string]interface{}{"task_id": "task1", "n": 2})
	if err != nil {
		t.Fatal(err)
	}

	latest, err := loadLatestStartEvent("c1", "task1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if latest.ID != secondID {
		t.Fatalf("latest=%s want %s", latest.ID, secondID)
	}

	byID, err := loadLatestStartEvent("c1", "task1", firstID, "")
	if err != nil {
		t.Fatal(err)
	}
	if byID.ID != firstID || byID.EventData["n"] != float64(1) {
		t.Fatalf("byID=%+v", byID)
	}
}

func TestCreateAccessKeyIAMAssociation(t *testing.T) {
	setupCloudTestDB(t)
	if err := createAccessKeyIAMAssociation("auth1", "AK_TEST", "iam-123"); err != nil {
		t.Fatal(err)
	}
	var authID, accessKey, iamID string
	if err := db.QueryRow(`
		SELECT cloud_platform_auth_id, access_key, iam_id
		FROM cloud_access_key_iam_associations LIMIT 1
	`).Scan(&authID, &accessKey, &iamID); err != nil {
		t.Fatal(err)
	}
	if authID != "auth1" || accessKey != "AK_TEST" || iamID != "iam-123" {
		t.Fatalf("row=%s %s %s", authID, accessKey, iamID)
	}
}

// OPT-20260818-015: 重放的授权事件不得产生重复关联行（owner DB 唯一键兜底）。
func TestCreateAccessKeyIAMAssociationIdempotent(t *testing.T) {
	setupCloudTestDB(t)
	if err := createAccessKeyIAMAssociation("auth1", "AK_TEST", "iam-123"); err != nil {
		t.Fatal(err)
	}
	// 同一 (cloud_platform_auth_id, access_key) 再次投递 → upsert 而非新行。
	if err := createAccessKeyIAMAssociation("auth1", "AK_TEST", "iam-456"); err != nil {
		t.Fatal(err)
	}
	var cnt int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM cloud_access_key_iam_associations
		WHERE cloud_platform_auth_id='auth1' AND access_key='AK_TEST'
	`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row after re-delivery, got %d", cnt)
	}
	var iamID string
	if err := db.QueryRow(`
		SELECT iam_id FROM cloud_access_key_iam_associations
		WHERE cloud_platform_auth_id='auth1' AND access_key='AK_TEST'
	`).Scan(&iamID); err != nil {
		t.Fatal(err)
	}
	if iamID != "iam-456" {
		t.Fatalf("iam_id=%s want iam-456 (re-delivery should update)", iamID)
	}
}

// OPT-20260818-015: 导入接口对同一自然键批量重放同样只保留一行。
func TestInternalImportAccessKeyIAMAssociationsDedup(t *testing.T) {
	setupCloudTestDB(t)
	body := `{"items":[{"cloud_platform_auth_id":"auth1","access_key":"AK","iam_id":"iam-1"}]}`
	postImport := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/access-key-iam-associations/import", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleInternalImportAccessKeyIAMAssociations(rec, req)
		return rec
	}
	if rec := postImport(); rec.Code != http.StatusOK {
		t.Fatalf("first import status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec := postImport(); rec.Code != http.StatusOK {
		t.Fatalf("second import status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cnt int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM cloud_access_key_iam_associations
		WHERE cloud_platform_auth_id='auth1' AND access_key='AK'
	`).Scan(&cnt); err != nil {
		t.Fatal(err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row after duplicate import, got %d", cnt)
	}
}
