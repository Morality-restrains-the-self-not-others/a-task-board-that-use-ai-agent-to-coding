package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func listSystemAdminUsers(t *testing.T, rawQuery string, userID string) (int, []map[string]interface{}, int64) {
	t.Helper()
	path := "/api/system-admin/users/"
	if rawQuery != "" {
		path += "?" + rawQuery
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-User-Id", userID)
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		return rec.Code, nil, 0
	}
	var resp struct {
		Users []map[string]interface{} `json:"users"`
		Total int64                    `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
	}
	return rec.Code, resp.Users, resp.Total
}

func userIDsInList(users []map[string]interface{}) map[string]bool {
	out := map[string]bool{}
	for _, u := range users {
		id, _ := u["id"].(string)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func TestSystemAdminListUsersEmailContainsFilter(t *testing.T) {
	setupAuthTestDB(t)
	alphaID, _, err := createUserWithEmailLogin("colfilter-alpha@example.com", "hash")
	if err != nil {
		t.Fatalf("alpha: %v", err)
	}
	betaID, _, err := createUserWithEmailLogin("colfilter-beta@example.com", "hash")
	if err != nil {
		t.Fatalf("beta: %v", err)
	}

	code, users, total := listSystemAdminUsers(t, "email="+url.QueryEscape("colfilter-alpha"), "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[alphaID] {
		t.Fatalf("expected alpha in results, got %v", ids)
	}
	if ids[betaID] {
		t.Fatalf("beta should be excluded, got %v", ids)
	}
	if total < 1 {
		t.Fatalf("expected total>=1, got %d", total)
	}
}

func TestSystemAdminListUsersRoleSuperuserFilter(t *testing.T) {
	setupAuthTestDB(t)
	normalID, _, err := createUserWithEmailLogin("colfilter-role-user@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}

	code, users, _ := listSystemAdminUsers(t, "role=superuser", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if ids[normalID] {
		t.Fatalf("normal user must not appear in role=superuser, got %v", ids)
	}
}

func TestSystemAdminListUsersLoginMethodPhoneFilter(t *testing.T) {
	setupAuthTestDB(t)
	phoneID, _, err := createUserWithEmailLogin("colfilter-phone@example.com", "hash")
	if err != nil {
		t.Fatalf("phone user: %v", err)
	}
	emailOnlyID, _, err := createUserWithEmailLogin("colfilter-emailonly@example.com", "hash")
	if err != nil {
		t.Fatalf("email-only: %v", err)
	}
	seedWechatLinkedLoginMethod(t, phoneID, "phone", "13900001111")

	code, users, _ := listSystemAdminUsers(t, "login_method=phone", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[phoneID] {
		t.Fatalf("expected phone user, got %v", ids)
	}
	if ids[emailOnlyID] {
		t.Fatalf("email-only user should be excluded, got %v", ids)
	}
}

func TestSystemAdminListUsersDateJoinedFromFilter(t *testing.T) {
	setupAuthTestDB(t)
	oldID, _, err := createUserWithEmailLogin("colfilter-old@example.com", "hash")
	if err != nil {
		t.Fatalf("old: %v", err)
	}
	newID, _, err := createUserWithEmailLogin("colfilter-new@example.com", "hash")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET date_joined = '2020-01-02 00:00:00' WHERE id = ?`, oldID); err != nil {
		t.Fatalf("stamp old: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET date_joined = '2026-08-20 00:00:00' WHERE id = ?`, newID); err != nil {
		t.Fatalf("stamp new: %v", err)
	}

	code, users, _ := listSystemAdminUsers(t, "date_joined_from=2026-01-01", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if ids[oldID] {
		t.Fatalf("old user should be excluded, got %v", ids)
	}
	if !ids[newID] {
		t.Fatalf("expected new user, got %v", ids)
	}
}

func TestSystemAdminListUsersInactiveFilter(t *testing.T) {
	setupAuthTestDB(t)
	disabledID, _, err := createUserWithEmailLogin("colfilter-disabled@example.com", "hash")
	if err != nil {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_active = 0 WHERE id = ?`, disabledID); err != nil {
		t.Fatalf("disable: %v", err)
	}

	code, users, _ := listSystemAdminUsers(t, "is_active=false", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[disabledID] {
		t.Fatalf("expected disabled user, got %v", ids)
	}
}

func TestSystemAdminListUsersIDContainsFilter(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("colfilter-id@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if len(userID) < 6 {
		t.Fatalf("unexpected short id %q", userID)
	}
	needle := userID[len(userID)-6:]

	code, users, _ := listSystemAdminUsers(t, "id="+url.QueryEscape(needle), "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !userIDsInList(users)[userID] {
		t.Fatalf("expected id contains match for %s", userID)
	}
}

func TestSystemAdminListUsersPhoneContainsFilter(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("colfilter-phonenum@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	otherID, _, err := createUserWithEmailLogin("colfilter-phonenum-other@example.com", "hash")
	if err != nil {
		t.Fatalf("other: %v", err)
	}
	seedWechatLinkedLoginMethod(t, userID, "phone", "13811112222")
	seedWechatLinkedLoginMethod(t, otherID, "phone", "13700000000")

	code, users, _ := listSystemAdminUsers(t, "phone=1381111", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[userID] {
		t.Fatalf("expected phone match, got %v", ids)
	}
	if ids[otherID] {
		t.Fatalf("other phone should be excluded, got %v", ids)
	}
}

func TestSystemAdminListUsersQAndEmailAND(t *testing.T) {
	setupAuthTestDB(t)
	aID, _, err := createUserWithEmailLogin("colfilter-and-a@example.com", "hash")
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	_, _, err = createUserWithEmailLogin("colfilter-and-b@example.com", "hash")
	if err != nil {
		t.Fatalf("b: %v", err)
	}

	q := url.QueryEscape("colfilter-and-a@example.com")
	email := url.QueryEscape("colfilter-and-b")
	code, users, total := listSystemAdminUsers(t, "q="+q+"&email="+email, "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if userIDsInList(users)[aID] {
		t.Fatalf("AND should exclude A when email filter is B")
	}
	if total != 0 && len(users) != 0 {
		t.Fatalf("expected empty AND result, total=%d users=%d", total, len(users))
	}
}

func TestSystemAdminListUsersForbiddenForNonSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	uid, _, err := createUserWithEmailLogin("colfilter-nosuper@example.com", "hash")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	code, _, _ := listSystemAdminUsers(t, "email=colfilter", uid)
	if code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", code)
	}
}

func TestSystemAdminListUsersProfitSharingFilter(t *testing.T) {
	setupAuthTestDB(t)
	yesID, _, err := createUserWithEmailLogin("colfilter-psq-yes@example.com", "hash")
	if err != nil {
		t.Fatalf("yes: %v", err)
	}
	noID, _, err := createUserWithEmailLogin("colfilter-psq-no@example.com", "hash")
	if err != nil {
		t.Fatalf("no: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/qualification/active/batch") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"qualifications": map[string]bool{yesID: true, noID: false},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	code, users, _ := listSystemAdminUsers(t, "has_profit_sharing=true", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[yesID] {
		t.Fatalf("expected qualified user, got %v", ids)
	}
	if ids[noID] {
		t.Fatalf("unqualified user should be excluded, got %v", ids)
	}
}

func TestSystemAdminListUsersTenantCompanyFilter(t *testing.T) {
	setupAuthTestDB(t)
	hitID, _, err := createUserWithEmailLogin("colfilter-co-hit@example.com", "hash")
	if err != nil {
		t.Fatalf("hit: %v", err)
	}
	missID, _, err := createUserWithEmailLogin("colfilter-co-miss@example.com", "hash")
	if err != nil {
		t.Fatalf("miss: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/members/batch-get") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": hitID, "company_id": "c-acme", "company_name": "Acme FilterCo", "is_active": true},
				{"user_id": missID, "company_id": "c-zzz", "company_name": "Other Org", "is_active": true},
			},
		})
	}))
	defer srv.Close()
	prev := cfg.TenantServiceURL
	cfg.TenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TenantServiceURL = prev })

	code, users, _ := listSystemAdminUsers(t, "tenant_company="+url.QueryEscape("FilterCo"), "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[hitID] {
		t.Fatalf("expected company hit, got %v", ids)
	}
	if ids[missID] {
		t.Fatalf("other company should be excluded, got %v", ids)
	}
}

func TestSystemAdminListUsersReferrerFilter(t *testing.T) {
	setupAuthTestDB(t)
	referrerID, _, err := createUserWithEmailLogin("colfilter-referrer@example.com", "hash")
	if err != nil {
		t.Fatalf("referrer: %v", err)
	}
	hitID, _, err := createUserWithEmailLogin("colfilter-referred@example.com", "hash")
	if err != nil {
		t.Fatalf("referred: %v", err)
	}
	missID, _, err := createUserWithEmailLogin("colfilter-noref@example.com", "hash")
	if err != nil {
		t.Fatalf("noref: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/referrers/lookup") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"referrers": map[string]map[string]string{
				hitID: {"referrer_user_id": referrerID, "channel_code": "X"},
			},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

	code, users, _ := listSystemAdminUsers(t, "referrer="+url.QueryEscape("colfilter-referrer"), "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[hitID] {
		t.Fatalf("expected referred user, got %v", ids)
	}
	if ids[missID] {
		t.Fatalf("user without referrer should be excluded, got %v", ids)
	}
}

func TestSystemAdminListPhoneSearchIncludesArchivedOnActiveTab(t *testing.T) {
	setupAuthTestDB(t)
	const national = "13900001111"
	uid, err := createUserWithPhoneLogin("+86", national, "OldPassw0rd!", "")
	if err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	archiveUser(t, uid)

	code, users, _ := listSystemAdminUsers(t, "is_archived=false&phone="+national, "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids := userIDsInList(users)
	if !ids[uid] {
		t.Fatalf("identifier search on active tab should include archived occupant, got %v", ids)
	}

	code, users, _ = listSystemAdminUsers(t, "is_archived=false&q="+national, "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids = userIDsInList(users)
	if !ids[uid] {
		t.Fatalf("q search on active tab should include archived occupant, got %v", ids)
	}

	code, users, _ = listSystemAdminUsers(t, "is_archived=false", "bootstrap-admin")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	ids = userIDsInList(users)
	if ids[uid] {
		t.Fatalf("active tab without identifier lookup must still hide archived user, got %v", ids)
	}
}
