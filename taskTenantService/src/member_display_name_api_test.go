package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCompanyMembersListShowsPersonalNicknameNotCompanyName — 人员管理页
// 「公司成员名称」不得显示 onboarding 公司名「我的公司」，应回退个人昵称。
func TestCompanyMembersListShowsPersonalNicknameNotCompanyName(t *testing.T) {
	mux := setupTestService(t)

	// 误种子：member_name = 默认公司名
	_, err := db.Exec(`UPDATE tenant_company SET name=? WHERE id='c1'`, defaultPersonalCompanyName)
	if err != nil {
		t.Fatalf("rename company: %v", err)
	}
	_, err = db.Exec(`UPDATE tenant_company_member SET member_name=? WHERE id='m1'`, defaultPersonalCompanyName)
	if err != nil {
		t.Fatalf("seed mis-named member: %v", err)
	}

	prev := fetchPersonalNicknamesFn
	fetchPersonalNicknamesFn = func(userIDs []string) map[string]string {
		out := map[string]string{}
		for _, id := range userIDs {
			out[id] = "软刀"
		}
		return out
	}
	t.Cleanup(func() { fetchPersonalNicknamesFn = prev })

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/company_members/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	members, _ := out["members"].([]interface{})
	if len(members) < 1 {
		t.Fatalf("expected members, got %v", out)
	}
	first, _ := members[0].(map[string]interface{})
	got, _ := first["member_name"].(string)
	if got != "软刀" {
		t.Fatalf("member_name = %q, want 软刀 (personal nickname)", got)
	}
	if got == defaultPersonalCompanyName {
		t.Fatalf("member_name must not be default company name")
	}

	// 回填：DB 中误种子应被个人昵称覆盖
	var stored string
	if err := db.QueryRow(`SELECT member_name FROM tenant_company_member WHERE id='m1'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "软刀" {
		t.Fatalf("DB member_name not healed: got %q want 软刀", stored)
	}
}

// TestCreateCompanySeedsMemberNameFromPersonalNickname — onboarding 建公司时
// 创建者 member_name 取个人昵称，不写公司名。
func TestCreateCompanySeedsMemberNameFromPersonalNickname(t *testing.T) {
	mux := setupTestService(t)

	prev := fetchPersonalNicknamesFn
	fetchPersonalNicknamesFn = func(userIDs []string) map[string]string {
		return map[string]string{"creator-u1": "租户昵称"}
	}
	t.Cleanup(func() { fetchPersonalNicknamesFn = prev })

	body := `{"name":"我的公司"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/_/accounts/companies/", bytes.NewBufferString(body)), "creator-u1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	companyID, _ := created["company_id"].(string)
	if companyID == "" {
		t.Fatalf("missing company_id: %v", created)
	}

	var memberName string
	err := db.QueryRow(
		`SELECT COALESCE(member_name,'') FROM tenant_company_member WHERE user_id=? AND company_id=?`,
		"creator-u1", companyID,
	).Scan(&memberName)
	if err != nil {
		t.Fatalf("query member: %v", err)
	}
	if memberName != "租户昵称" {
		t.Fatalf("member_name = %q, want 租户昵称 (not company name)", memberName)
	}
}
