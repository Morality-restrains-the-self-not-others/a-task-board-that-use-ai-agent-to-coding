package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPatchUserAsAdmin_legacySuperuserFlagCannotGrantSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	actorID, _, err := createUserWithEmailLogin("flag-only-actor@test.com", "hash")
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_superuser = 1 WHERE id = ?`, actorID); err != nil {
		t.Fatalf("flag actor: %v", err)
	}
	targetID, _, err := createUserWithEmailLogin("grant-target@test.com", "hash")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	body, _ := json.Marshal(map[string]bool{"is_superuser": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/users/"+targetID+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", actorID)
	req.Header.Set("X-User-Roles", "employee")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	var isSuper int
	if err := db.QueryRow(`SELECT is_superuser FROM auth_user WHERE id = ?`, targetID).Scan(&isSuper); err != nil {
		t.Fatalf("scan target: %v", err)
	}
	if isSuper != 0 {
		t.Fatalf("target must stay non-superuser, got %d", isSuper)
	}
}

func TestPatchUserAsAdmin_superAdminCanGrantSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("grant-ok@test.com", "hash")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	body, _ := json.Marshal(map[string]bool{"is_superuser": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/users/"+targetID+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "bootstrap-admin")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var isSuper int
	if err := db.QueryRow(`SELECT is_superuser FROM auth_user WHERE id = ?`, targetID).Scan(&isSuper); err != nil {
		t.Fatalf("scan target: %v", err)
	}
	if isSuper != 1 {
		t.Fatalf("target should be superuser, got %d", isSuper)
	}
}

func TestPatchUserAsAdmin_legacyFlagCanUpdateStaffWithoutSuperuserKey(t *testing.T) {
	setupAuthTestDB(t)
	actorID, _, err := createUserWithEmailLogin("flag-staff-actor@test.com", "hash")
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_superuser = 1 WHERE id = ?`, actorID); err != nil {
		t.Fatalf("flag actor: %v", err)
	}
	targetID, _, err := createUserWithEmailLogin("staff-target@test.com", "hash")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	body, _ := json.Marshal(map[string]bool{"is_staff": true})
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/users/"+targetID+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", actorID)
	req.Header.Set("X-User-Roles", "employee")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateUserAsAdmin_legacyFlagCannotCreateSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	actorID, _, err := createUserWithEmailLogin("flag-create-actor@test.com", "hash")
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_superuser = 1 WHERE id = ?`, actorID); err != nil {
		t.Fatalf("flag actor: %v", err)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"email":        "new-super@test.com",
		"password":     "secret12",
		"is_superuser": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/users/create/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", actorID)
	req.Header.Set("X-User-Roles", "employee")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
