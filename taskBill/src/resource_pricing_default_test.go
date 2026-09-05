package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultTaskPostUnitPriceCentsIsFiftyFiveFen(t *testing.T) {
	if DefaultTaskPostUnitPriceCents != 55 {
		t.Fatalf("DefaultTaskPostUnitPriceCents=%d want 55 (0.55 元/帖/12个月)", DefaultTaskPostUnitPriceCents)
	}
	if got := centsToYuanStr(DefaultTaskPostUnitPriceCents); got != "0.55" {
		t.Fatalf("centsToYuanStr(%d)=%q want 0.55", DefaultTaskPostUnitPriceCents, got)
	}
}

func TestMigratedTaskPostUnitPriceIsFiftyFiveFen(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	var price int64
	var unit string
	if err := db.QueryRow(`SELECT price, unit FROM billing_unit WHERE unit_type = 'server_start'`).Scan(&price, &unit); err != nil {
		t.Fatalf("read server_start: %v", err)
	}
	if price != DefaultTaskPostUnitPriceCents {
		t.Fatalf("server_start.price=%d want %d", price, DefaultTaskPostUnitPriceCents)
	}
	if unit != "帖/12个月" {
		t.Fatalf("server_start.unit=%q want 帖/12个月", unit)
	}

	var renewalPrice int64
	err := db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = 'server_start_renewal'`).Scan(&renewalPrice)
	if err == nil && renewalPrice != 0 {
		t.Fatalf("server_start_renewal.price=%d want 0 (续存不另扣费)", renewalPrice)
	}

	rp, err := getCurrentResourcePricing()
	if err != nil {
		t.Fatalf("getCurrentResourcePricing: %v", err)
	}
	if rp.TaskPostUnitPriceCents != DefaultTaskPostUnitPriceCents {
		t.Fatalf("TaskPostUnitPriceCents=%d want %d", rp.TaskPostUnitPriceCents, DefaultTaskPostUnitPriceCents)
	}

	got, err := getUnitPriceCents(ResourceTypeTaskPost)
	if err != nil {
		t.Fatalf("getUnitPriceCents: %v", err)
	}
	if got != DefaultTaskPostUnitPriceCents {
		t.Fatalf("getUnitPriceCents=%d want %d", got, DefaultTaskPostUnitPriceCents)
	}
}

func TestGetUnitPriceCentsFallsBackToDefaultWhenMissing(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	if _, err := db.Exec(`DELETE FROM billing_unit WHERE unit_type = 'server_start'`); err != nil {
		t.Fatalf("delete server_start: %v", err)
	}

	got, err := getUnitPriceCents(ResourceTypeTaskPost)
	if err != nil {
		t.Fatalf("getUnitPriceCents: %v", err)
	}
	if got != DefaultTaskPostUnitPriceCents {
		t.Fatalf("fallback getUnitPriceCents=%d want %d", got, DefaultTaskPostUnitPriceCents)
	}

	rp, err := getCurrentResourcePricing()
	if err != nil {
		t.Fatalf("getCurrentResourcePricing: %v", err)
	}
	if rp.TaskPostUnitPriceCents != DefaultTaskPostUnitPriceCents {
		t.Fatalf("fallback TaskPostUnitPriceCents=%d want %d", rp.TaskPostUnitPriceCents, DefaultTaskPostUnitPriceCents)
	}
}

func TestPublicProductPricingFallsBackToDefault(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	if _, err := db.Exec(`DELETE FROM billing_unit WHERE unit_type IN ('server_start', 'server_start_renewal')`); err != nil {
		t.Fatalf("delete units: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/public/product-pricing/", nil)
	rr := httptest.NewRecorder()
	handlePublicProductPricing(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	points, ok := resp["task_points"].(float64)
	if !ok {
		t.Fatalf("task_points type %T want number, body=%s", resp["task_points"], rr.Body.String())
	}
	if int64(points) != DefaultTaskPostUnitPriceCents {
		t.Fatalf("task_points=%v want %d", points, DefaultTaskPostUnitPriceCents)
	}
	renewal, ok := resp["task_renewal_points"].(float64)
	if !ok {
		t.Fatalf("task_renewal_points type %T want number, body=%s", resp["task_renewal_points"], rr.Body.String())
	}
	if int64(renewal) != 0 {
		t.Fatalf("task_renewal_points=%v want 0 (续存不另扣费)", renewal)
	}
}
