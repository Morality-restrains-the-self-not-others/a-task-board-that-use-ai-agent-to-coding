package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260823-058: admin_grant 幂等键落库须附带操作者 X-User-Id 与目标租户，
// 管理端可追溯「哪次操作产生了哪张赠送订单」。

func TestAdminGrantRecordsIdempotencyAudit(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000101)
	ik := "admin-grant-audit-test-ik"
	grants := []ResourceGrantInput{{
		ResourceType: ResourceTypeTaskPost,
		Quantity:     5,
		Reason:       "audit test",
	}}
	if _, err := adminGrantResourcesWithMembership(context.Background(), tenantID, grants, "admin-7", ik, "", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	var operator string
	var gotTenant int64
	var opType string
	if err := db.QueryRow(
		`SELECT COALESCE(operator_user_id,''), COALESCE(tenant_id,0), COALESCE(op_type,'')
		 FROM billing_idempotency_key WHERE `+"`key`"+` = ?`, ik,
	).Scan(&operator, &gotTenant, &opType); err != nil {
		t.Fatalf("query idempotency record: %v", err)
	}
	if operator != "admin-7" || gotTenant != tenantID || opType != "admin_grant" {
		t.Fatalf("audit record operator=%q tenant=%d op_type=%q, want admin-7/%d/admin_grant", operator, gotTenant, opType, tenantID)
	}
}

func TestAdminGrantDuplicateIdempotencyKeySkipsReGrant(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000101)
	ik := "admin-grant-audit-test-ik"
	grants := []ResourceGrantInput{{
		ResourceType: ResourceTypeTaskPost,
		Quantity:     5,
		Reason:       "audit test",
	}}
	if _, err := adminGrantResourcesWithMembership(context.Background(), tenantID, grants, "admin-7", ik, "", ""); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	res, err := adminGrantResourcesWithMembership(context.Background(), tenantID, grants, "admin-7", ik, "", "")
	if err != nil {
		t.Fatalf("second grant: %v", err)
	}
	if idem, _ := res["idempotent"].(bool); !idem {
		t.Fatalf("second grant with same key should be idempotent, got %+v", res)
	}
}

func TestSystemAdminListIdempotencyRecords(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tenantID = int64(9300000102)
	ik := "admin-grant-http-test-ik"
	grants := []ResourceGrantInput{{
		ResourceType: ResourceTypeTaskPost,
		Quantity:     3,
		Reason:       "http audit",
	}}
	if _, err := adminGrantResourcesWithMembership(context.Background(), tenantID, grants, "admin-9", ik, "", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/idempotency-records/?op_type=admin_grant&tenant_id=9300000102", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "super_admin")
	req.Header.Set("X-User-Id", "admin-9")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Total   int `json:"total"`
		Records []struct {
			Key            string `json:"key"`
			OpType         string `json:"op_type"`
			OperatorUserID string `json:"operator_user_id"`
			TenantID       string `json:"tenant_id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Total < 1 {
		t.Fatalf("total=%d want >=1", out.Total)
	}
	found := false
	for _, r := range out.Records {
		if r.Key == ik {
			found = true
			if r.OpType != "admin_grant" || r.OperatorUserID != "admin-9" || r.TenantID != formatID(tenantID) {
				t.Fatalf("record=%+v, want op=admin_grant operator=admin-9 tenant=%s", r, formatID(tenantID))
			}
		}
	}
	if !found {
		t.Fatalf("record with key %s not found: %+v", ik, out.Records)
	}
}

func TestSystemAdminListIdempotencyRecordsRejectsNonStaff(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/idempotency-records/", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-User-Roles", "member")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}
