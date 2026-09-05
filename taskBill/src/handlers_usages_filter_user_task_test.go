package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUsagesListFilteredByUserIDAndTaskID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	now := utcNow()
	keepUser := "user-keep"
	keepTask := "task-keep"
	keepID := generateSnowflakeID()
	dropID := generateSnowflakeID()
	_, err = db.Exec(`INSERT INTO billing_usage (
		id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id, description, usage_time
	) VALUES (?, ?, ?, 1, NULL, ?, NULL, ?, 'keep', ?)`, keepID, acc.ID, unitID, keepUser, keepTask, now)
	if err != nil {
		t.Fatalf("insert keep: %v", err)
	}
	_, err = db.Exec(`INSERT INTO billing_usage (
		id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id, description, usage_time
	) VALUES (?, ?, ?, 1, NULL, ?, NULL, ?, 'drop', ?)`, dropID, acc.ID, unitID, "user-drop", "task-drop", now)
	if err != nil {
		t.Fatalf("insert drop: %v", err)
	}

	ureq := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562496/billing/usages/?user_id="+keepUser+"&task_id="+keepTask+"&page=1&page_size=20",
		nil,
	)
	urr := httptest.NewRecorder()
	handleUsagesList(urr, ureq)
	if urr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", urr.Code, urr.Body.String())
	}
	var page struct {
		Results []map[string]interface{} `json:"results"`
		Total   int64                    `json:"total"`
	}
	if err := json.Unmarshal(urr.Body.Bytes(), &page); err != nil {
		var list []map[string]interface{}
		if err2 := json.Unmarshal(urr.Body.Bytes(), &list); err2 != nil {
			t.Fatalf("decode: %v / %v body=%s", err, err2, urr.Body.String())
		}
		page.Results = list
		page.Total = int64(len(list))
	}
	if page.Total != 1 || len(page.Results) != 1 {
		t.Fatalf("total=%d len=%d body=%s", page.Total, len(page.Results), urr.Body.String())
	}
	if page.Results[0]["user_id"] != keepUser || page.Results[0]["task_id"] != keepTask {
		t.Fatalf("row=%#v", page.Results[0])
	}
}
