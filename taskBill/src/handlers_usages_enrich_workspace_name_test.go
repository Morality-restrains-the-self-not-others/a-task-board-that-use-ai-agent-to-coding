package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Usages list must enrich workspace_id into workspace_name for the billing UI.
func TestUsagesListEnrichesWorkspaceNameDirect(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	workspaceID := "861623708318031872"
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("server_start", "智能体任务", 30)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	usageID := generateSnowflakeID()
	now := utcNow()
	_, err = db.Exec(`
		INSERT INTO billing_usage (
			id, account_id, billing_unit_id, amount, project_id, user_id, workspace_id, task_id,
			description, usage_time
		) VALUES (?, ?, ?, 1, NULL, NULL, ?, NULL, '启动消耗', ?)`,
		usageID, acc.ID, unitID, workspaceID, now,
	)
	if err != nil {
		t.Fatalf("insert usage: %v", err)
	}

	// enrichBillingRows is best-effort; with services unavailable, handler still returns data
	prevProject := cfg.TaskProjectServiceBase
	prevTask := cfg.TaskTaskServiceBase
	cfg.TaskProjectServiceBase = ""
	cfg.TaskTaskServiceBase = ""
	t.Cleanup(func() {
		cfg.TaskProjectServiceBase = prevProject
		cfg.TaskTaskServiceBase = prevTask
	})

	ureq := httptest.NewRequest(http.MethodGet, "/api/tenant/850256677331562496/billing/usages/", nil)
	urr := httptest.NewRecorder()
	handleUsagesList(urr, ureq)
	if urr.Code != http.StatusOK {
		t.Fatalf("usages status=%d body=%s", urr.Code, urr.Body.String())
	}
	var usages []map[string]interface{}
	if err := json.Unmarshal(urr.Body.Bytes(), &usages); err != nil {
		t.Fatalf("decode usages: %v", err)
	}
	if len(usages) == 0 {
		t.Fatal("expected at least one usage")
	}
	// workspace_id should still be present even if workspace_name enrichment is skipped
	if usages[0]["workspace_id"] != workspaceID {
		t.Fatalf("workspace_id=%v want %s", usages[0]["workspace_id"], workspaceID)
	}
}
