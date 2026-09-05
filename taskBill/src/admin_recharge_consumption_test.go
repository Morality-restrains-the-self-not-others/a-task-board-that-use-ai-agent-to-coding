package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserRechargeConsumptionT1RechargeConsumeRemaining(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	userID := "42"

	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}

	if _, err := creditRecharge(
		t.Context(), tenantID, userID, 10000,
		"paypal:t1-recharge", paypalPointsSourceType, "充值", "cap-t1",
	); err != nil {
		t.Fatalf("recharge: %v", err)
	}

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("post_creation", "任务帖创建", 3)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	if _, err := recordConsumption(
		t.Context(), acc, unitID, 7000,
		"txn-t1-consume", "idem-t1-consume", "consume", "t1", userID, "w1", "p1",
	); err != nil {
		t.Fatalf("consume: %v", err)
	}

	// Use internal endpoint which now uses requireInternalOrGatewayAuth
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/admin/user-recharge-consumption/?user_ids=42", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	rr := httptest.NewRecorder()
	handleInternalUserRechargeConsumption(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("total=%v", body["total"])
	}
	results := body["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("results len=%d", len(results))
	}
	row := results[0].(map[string]interface{})
	assertUserRechargeConsumptionRow(t, row, userID, 10000, "100.00", 3000, 7000, 7000)
}

func TestUserRechargeConsumptionT2AdminGrantExcluded(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	userID := "grant-user"

	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}

	if _, err := creditRecharge(
		t.Context(), tenantID, userID, 5000,
		"admin:grant-t2", "admin_grant", "赠送", "",
	); err != nil {
		t.Fatalf("admin grant: %v", err)
	}
	if _, err := creditRecharge(
		t.Context(), tenantID, userID, 10000,
		"paypal:t2-recharge", paypalPointsSourceType, "充值", "cap-t2",
	); err != nil {
		t.Fatalf("user recharge: %v", err)
	}

	row, err := scanUserRechargeConsumptionRow(userID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if jsonInt64(row["recharge_points"]) != 10000 {
		t.Fatalf("recharge_points=%v want 10000", row["recharge_points"])
	}
	if row["recharge_amount_yuan"] != "100.00" {
		t.Fatalf("recharge_amount_yuan=%v want 100.00", row["recharge_amount_yuan"])
	}
}

func TestUserRechargeConsumptionT3CommissionableEqualsConsumed(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	userID := "commission-user"

	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}

	if _, err := creditRecharge(
		t.Context(), tenantID, userID, 5000,
		"paypal:t3-recharge", paypalPointsSourceType, "充值", "cap-t3",
	); err != nil {
		t.Fatalf("recharge: %v", err)
	}

	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	unitID, err := ensureBillingUnit("post_creation", "任务帖创建", 3)
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	if _, err := recordConsumption(
		t.Context(), acc, unitID, 1200,
		"txn-t3-consume", "idem-t3-consume", "consume", "t3", userID, "w1", "p1",
	); err != nil {
		t.Fatalf("consume: %v", err)
	}

	row, err := scanUserRechargeConsumptionRow(userID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	consumed := jsonInt64(row["consumed_points"])
	commissionable := jsonInt64(row["commissionable_points"])
	if commissionable != consumed {
		t.Fatalf("commissionable=%d consumed=%d", commissionable, consumed)
	}
	if consumed != 1200 {
		t.Fatalf("consumed=%d want 1200", consumed)
	}
}

func TestUserRechargeConsumptionT4NumericQExactSearch(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	for i, uid := range []string{"42", "43"} {
		if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
			t.Fatalf("account: %v", err)
		}
		if _, err := creditRecharge(
			t.Context(), tenantID, uid, 10000+int64(i),
			fmt.Sprintf("paypal:t4-%d", i), paypalPointsSourceType, "充值", "cap-t4",
		); err != nil {
			t.Fatalf("recharge %s: %v", uid, err)
		}
	}

	// q=42（纯数字）→ 精确用户 ID 过滤，不依赖 taskAuth
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/admin/user-recharge-consumption/?q=42", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	rr := httptest.NewRecorder()
	handleInternalUserRechargeConsumption(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("q=42 status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode q=42: %v", err)
	}
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("q=42 total=%v", body["total"])
	}
	results := body["results"].([]interface{})
	if len(results) != 1 || results[0].(map[string]interface{})["user_id"] != "42" {
		t.Fatalf("q=42 results=%v", results)
	}

	// q=42, 43（逗号分隔纯数字）→ 仍按精确 ID 列表过滤
	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/admin/user-recharge-consumption/?q=42,%2043", nil)
	req2.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	rr2 := httptest.NewRecorder()
	handleInternalUserRechargeConsumption(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("q=42,43 status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var body2 map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("decode q=42,43: %v", err)
	}
	if int(body2["total"].(float64)) != 2 {
		t.Fatalf("q=42,43 total=%v", body2["total"])
	}
}

func TestUserRechargeConsumptionT5FuzzyQDelegatesToTaskAuth(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	userID := "abc-user-id"
	if _, _, err := getOrCreateBillingAccount(tenantID, false); err != nil {
		t.Fatalf("account: %v", err)
	}
	if _, err := creditRecharge(
		t.Context(), tenantID, userID, 8000,
		"paypal:t5-recharge", paypalPointsSourceType, "充值", "cap-t5",
	); err != nil {
		t.Fatalf("recharge: %v", err)
	}

	// 模拟 taskAuth：搜索返回匹配 ID，批量详情返回邮箱/用户名
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/internal/users/":
			if r.URL.Query().Get("q") == "fuzzy@test.com" {
				w.Write([]byte(`{"users":[{"id":"abc-user-id"}],"total":1}`))
			} else {
				w.Write([]byte(`{"users":[],"total":0}`))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/users/batch/details/":
			w.Write([]byte(`{"results":{"abc-user-id":{"user_id":"abc-user-id","email":"fuzzy@test.com","username":"fuzzy_user"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mock.Close()
	oldBase := cfg.TaskAuthBaseURL
	cfg.TaskAuthBaseURL = mock.URL
	defer func() { cfg.TaskAuthBaseURL = oldBase }()

	// q=邮箱 → 委托 taskAuth 解析 + 富化 email/username
	req := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/admin/user-recharge-consumption/?q=fuzzy%40test.com", nil)
	req.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	rr := httptest.NewRecorder()
	handleInternalUserRechargeConsumption(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("q=fuzzy status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int(body["total"].(float64)) != 1 {
		t.Fatalf("q=fuzzy total=%v", body["total"])
	}
	results := body["results"].([]interface{})
	row := results[0].(map[string]interface{})
	if row["user_id"] != userID {
		t.Fatalf("user_id=%v", row["user_id"])
	}
	if row["email"] != "fuzzy@test.com" || row["username"] != "fuzzy_user" {
		t.Fatalf("enrich email/username=%v/%v", row["email"], row["username"])
	}

	// q=无匹配 → 空结果而非全量
	req2 := httptest.NewRequest(http.MethodGet, "/api/internal/taskbill/admin/user-recharge-consumption/?q=nomatch", nil)
	req2.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	rr2 := httptest.NewRecorder()
	handleInternalUserRechargeConsumption(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("q=nomatch status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var body2 map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("decode nomatch: %v", err)
	}
	if int(body2["total"].(float64)) != 0 {
		t.Fatalf("q=nomatch total=%v", body2["total"])
	}
}

func assertUserRechargeConsumptionRow(
	t *testing.T,
	row map[string]interface{},
	userID string,
	recharge int64,
	amountYuan string,
	unconsumed, consumed, commissionable int64,
) {
	t.Helper()
	if row["user_id"] != userID {
		t.Fatalf("user_id=%v want %s", row["user_id"], userID)
	}
	if jsonInt64(row["recharge_points"]) != recharge {
		t.Fatalf("recharge_points=%v want %d", row["recharge_points"], recharge)
	}
	if row["recharge_amount_yuan"] != amountYuan {
		t.Fatalf("recharge_amount_yuan=%v want %s", row["recharge_amount_yuan"], amountYuan)
	}
	if jsonInt64(row["unconsumed_points"]) != unconsumed {
		t.Fatalf("unconsumed_points=%v want %d", row["unconsumed_points"], unconsumed)
	}
	if jsonInt64(row["consumed_points"]) != consumed {
		t.Fatalf("consumed_points=%v want %d", row["consumed_points"], consumed)
	}
	if jsonInt64(row["commissionable_points"]) != commissionable {
		t.Fatalf("commissionable_points=%v want %d", row["commissionable_points"], commissionable)
	}
}

func jsonInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
