package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUsagesListFilteredByWorkspaceID(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	wsKeep := "ws-keep-001"
	wsDrop := "ws-drop-002"
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	now := utcNow()
	keepID := generateSnowflakeID()
	dropID := generateSnowflakeID()
	for _, row := range []struct {
		id int64
		ws string
	}{
		{keepID, wsKeep},
		{dropID, wsDrop},
	} {
		_, err = db.Exec(`
			INSERT INTO billing_usage (
				id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
				description, usage_time
			) VALUES (?, ?, ?, 1, NULL, NULL, ?, NULL, '启动消耗', ?)`,
			row.id, acc.ID, unitID, row.ws, now,
		)
		if err != nil {
			t.Fatalf("insert usage %s: %v", row.ws, err)
		}
	}

	ureq := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562496/billing/usages/?workspace_id="+wsKeep+"&page=1&page_size=20",
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
		// Unpaged list shape fallback
		var list []map[string]interface{}
		if err2 := json.Unmarshal(urr.Body.Bytes(), &list); err2 != nil {
			t.Fatalf("decode: %v / %v body=%s", err, err2, urr.Body.String())
		}
		page.Results = list
		page.Total = int64(len(list))
	}
	if page.Total != 1 {
		t.Fatalf("total=%d want 1 body=%s", page.Total, urr.Body.String())
	}
	if len(page.Results) != 1 {
		t.Fatalf("len(results)=%d want 1", len(page.Results))
	}
	if page.Results[0]["workspace_id"] != wsKeep {
		t.Fatalf("workspace_id=%v want %s", page.Results[0]["workspace_id"], wsKeep)
	}
	if page.Results[0]["id"] != formatID(keepID) {
		t.Fatalf("id=%v want %s", page.Results[0]["id"], formatID(keepID))
	}
}
